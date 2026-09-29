package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"

	"XinText/internal/closeconfirm"
	"XinText/internal/i18n"
	"XinText/internal/model"
	"XinText/internal/service"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// quitting 标记用户已确认退出（托盘菜单「退出」或关闭确认框选「否」），
// 置位后窗口关闭不再弹确认框。
var quitting atomic.Bool

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIconPNG []byte

//go:embed app.config.json
var appConfigJSON []byte

// appConfigFile app.config.json 中 Go 端需要的字段
type appConfigFile struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Update  struct {
		APIURL          string `json:"apiUrl"`
		ReleasePageBase string `json:"releasePageBase"`
	} `json:"update"`
}

func main() {

	// Bootstrap services so they can be shared between Wails service registration
	// and the main function (e.g. window sizing from persisted config).
	//
	// 共享 system.db：file_recycle 与 system_config 表在启动时由
	// systemDBSchema 一次性创建，ConfigService 与 RecycleService 共用同一
	// *DBService，避免多连接竞争同一 SQLite 文件。
	baseDir, err := os.UserConfigDir()
	if err != nil {
		baseDir, _ = os.UserHomeDir()
	}
	appDataDir := filepath.Join(baseDir, "XinText")
	systemDB, err := service.NewDBService(filepath.Join(appDataDir, "system.db"), service.SystemDBSchema)
	if err != nil {
		log.Fatalf("system.db init failed: %v", err)
	}
	legacyConfigPath := filepath.Join(appDataDir, "config.json")

	configService := service.NewConfigService(systemDB, legacyConfigPath)
	// 首次启动：若存在旧版 config.json，迁移到 system_config 表后删除旧文件
	if err := configService.MigrateFromLegacyJSON(); err != nil {
		log.Printf("config migration failed: %v", err)
	}

	watchService := service.NewWatchService()
	imageService := service.NewImageService(configService)
	// 嵌入的 vditor 静态资源子树（CSS、KaTeX 字体等），供 ExportPDFFromHTML
	// 解压到临时目录后由 msedge headless 加载；构造导出服务时直接注入。
	var vditorFS fs.FS
	if sub, err := fs.Sub(assets, "frontend/dist/vditor"); err == nil {
		vditorFS = sub
	} else {
		log.Printf("vditor assets sub fs unavailable: %v", err)
	}
	exportService := service.NewExportService(configService, vditorFS)
	searchService := service.NewSearchService()
	recycleService := service.NewRecycleService(systemDB, configService)

	// 解析 app.config.json 供更新检测服务使用
	var appConf appConfigFile
	if err := json.Unmarshal(appConfigJSON, &appConf); err != nil {
		log.Printf("app.config.json parse failed: %v", err)
	}
	updateService := service.NewUpdateService(appConf.Name, appConf.Version, appConf.Update.APIURL, appConf.Update.ReleasePageBase)

	// Persisted window size; fall back to defaults if missing.
	cfg, err := configService.GetConfig()
	if err != nil {
		log.Printf("config load failed, using defaults: %v", err)
		cfg = model.DefaultConfig()
	}
	// 界面语言（首次启动 / 旧配置无该字段时已在 GetConfig 中按系统语言解析）
	appLang := i18n.ResolveLanguage(cfg.Language)
	nativeText := i18n.Native(appLang)

	// 日志服务：把标准 log 输出重定向为「stdout + go.log」双写，
	// 同时为前端 console 拦截提供写入入口（frontend.log）。
	logService := service.NewLogService(cfg)
	logService.InitStdLogger()
	logService.GoPrintf("XinText v%s 启动；日志目录：%s", appConf.Version, logService.GetLogDir())

	// Create a new Wails application by providing the necessary options.
	app := application.New(application.Options{
		Name:        "XinText",
		Description: "A Markdown editor built with Wails v3 + Muya.",
		Services: []application.Service{
			application.NewService(service.NewFileService()),
			application.NewService(configService),
			application.NewService(watchService),
			application.NewService(imageService),
			application.NewService(exportService),
			application.NewService(updateService),
			application.NewService(searchService),
			application.NewService(logService),
			application.NewService(recycleService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// 自定义 Windows 事件映射：移除 Windows.WindowClosing -> Common.WindowClosing
	// 的默认翻译。否则框架内部监听器收到 Common.WindowClosing 后会无条件销毁
	// 窗口，关闭确认将无法拦截。其余事件翻译保持默认行为不变。
	eventMapping := make(map[events.WindowEventType]events.WindowEventType, len(events.DefaultWindowEventMapping()))
	for source, target := range events.DefaultWindowEventMapping() {
		eventMapping[source] = target
	}
	delete(eventMapping, events.Windows.WindowClosing)

	// Create a new window with the persisted size. The application menu is
	// rendered in the frontend as a custom "文件" dropdown (VSCode style);
	// keyboard shortcuts are wired via global shortcuts below and a webview
	// keydown handler in the frontend.
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "XinText",
		Width:            cfg.WindowWidth,
		Height:           cfg.WindowHeight,
		BackgroundColour: application.NewRGB(255, 255, 255),
		URL:              "/",
		Windows: application.WindowsWindow{
			EventMapping: eventMapping,
		},
	})

	// 点击标题栏 X（含 Alt+F4 / 任务栏关闭）时询问用户，业务同 SuperSender：
	// 「是」最小化到系统托盘继续后台运行，「否」直接退出，「取消」保持窗口。
	win.OnWindowEvent(events.Windows.WindowClosing, func(*application.WindowEvent) {
		if quitting.Load() {
			// 托盘「退出」已确认，放行真正关闭
			win.Close()
			return
		}
		switch closeconfirm.AskCloseAction("XinText", appLang) {
		case closeconfirm.ActionMinimize:
			win.Hide()
		case closeconfirm.ActionQuit:
			quitting.Store(true)
			app.Quit()
		default:
			// 取消：WM_CLOSE 已返回 0，窗口保持原状
		}
	})

	// System tray icon (任务栏角标), 参考 SuperSender:
	// 左键单击显示并聚焦主窗口；右键弹出菜单（显示主窗口 / 退出）。
	trayMenu := app.Menu.New()
	trayMenu.Add(nativeText.ShowMainWindow).OnClick(func(*application.Context) {
		win.Show().Focus()
	})
	trayMenu.AddSeparator()
	trayMenu.Add(nativeText.Quit).OnClick(func(*application.Context) {
		// 已确认退出，避免窗口关闭流程再次弹出确认框
		quitting.Store(true)
		app.Quit()
	})
	tray := app.SystemTray.New()
	tray.SetIcon(appIconPNG)
	tray.SetTooltip("XinText")
	tray.SetMenu(trayMenu)
	tray.OnClick(func() {
		win.Show().Focus()
	})

	// Global (system-wide) shortcuts. These fire even when the window does not
	// have focus. Registration failures (e.g. accelerator owned by another app)
	// are non-fatal and only logged.
	globalShortcuts := []struct {
		accel string
		event string
	}{
		{"CmdOrCtrl+Alt+N", "menu:new"},
		{"CmdOrCtrl+Alt+O", "menu:open"},
		{"CmdOrCtrl+Alt+S", "menu:save"},
	}
	for _, sc := range globalShortcuts {
		event := sc.event
		if err := app.GlobalShortcut.Register(sc.accel, func() {
			app.Event.Emit(event)
		}); err != nil {
			log.Printf("global shortcut %s not registered: %v", sc.accel, err)
			logService.GoPrintf("global shortcut %s not registered: %v", sc.accel, err)
		}
	}

	// Ensure the filesystem watcher is torn down on exit.
	defer watchService.Close()
	defer logService.Close()
	defer recycleService.Close()
	defer systemDB.Close()

	// Run the application. This blocks until the application has been exited.
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
