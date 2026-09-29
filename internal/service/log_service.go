package service

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"XinText/internal/model"
)

// LogService 把 Go 后端 stdout 日志和前端 console 日志分别写入
// `go.log` 与 `frontend.log` 两个文件，供日志浮层查看与清理。
type LogService struct {
	mu        sync.Mutex
	logDir    string
	goPath    string
	fePath    string
	goFile    *os.File
	feFile    *os.File
	goLogger  *log.Logger // 只写 go.log
	feLogger  *log.Logger // 只写 frontend.log
	stdLogger *log.Logger // 写 stdout + go.log（供 GoPrintf 使用）
}

// NewLogService 根据 cfg.LogDir 初始化 LogService。空目录回退到
// <userConfigDir>/XinText/logs。
func NewLogService(cfg model.Config) *LogService {
	dir := cfg.LogDir
	if dir == "" {
		dir = defaultLogDir()
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	s := &LogService{logDir: abs}
	s.openFiles()
	// 主动写一条启动日志，让 go.log 不为空（也用于验证日志通道工作）
	if s.goLogger != nil {
		s.goLogger.Printf("=== LogService 启动：logDir=%s ===", s.logDir)
	}
	return s
}

func defaultLogDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base, _ = os.UserHomeDir()
	}
	return filepath.Join(base, "XinText", "logs")
}

// openFiles 创建日志目录并打开 go.log / frontend.log 文件句柄与 logger。
// 调用方需自行持有锁。
func (s *LogService) openFiles() {
	if err := os.MkdirAll(s.logDir, 0o755); err != nil {
		// 目录创建失败不影响主流程，只记录到 stderr
		log.Printf("LogService: mkdir %s failed: %v", s.logDir, err)
	}
	s.goPath = filepath.Join(s.logDir, "go.log")
	s.fePath = filepath.Join(s.logDir, "frontend.log")

	goFile, err := os.OpenFile(s.goPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("LogService: open %s failed: %v", s.goPath, err)
	} else {
		s.goFile = goFile
		s.goLogger = log.New(goFile, "", log.LstdFlags)
		s.stdLogger = log.New(io.MultiWriter(os.Stdout, goFile), "", log.LstdFlags)
	}

	feFile, err := os.OpenFile(s.fePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("LogService: open %s failed: %v", s.fePath, err)
	} else {
		s.feFile = feFile
		s.feLogger = log.New(feFile, "", log.LstdFlags)
	}
}

// InitStdLogger 把标准 log 的输出重定向为「go.log + stdout」。
// goFile 必须放在 MultiWriter 第一位：wails3 build 的 -H windowsgui 标志
// 使 os.Stdout 变为无效句柄，io.MultiWriter 写 stdout 失败后会跳过后续 writer，
// 把 goFile 放前面可确保日志先写入文件，stdout 失败不影响文件写入。
// 注意：Wails 在 application.New() 内部可能再次覆盖标准 log 的输出，
// 因此关键启动日志应使用 GoPrintf 显式写入，不依赖此全局重定向。
func (s *LogService) InitStdLogger() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.goFile == nil {
		return
	}
	log.SetOutput(io.MultiWriter(s.goFile, os.Stdout))
}

// GoPrintf 显式写入 go.log，不依赖全局 log.SetOutput，也不写 os.Stdout。
// 必须用 goLogger（只写 goFile）而非 stdLogger（MultiWriter 含 os.Stdout）：
// wails3 build 的 -H windowsgui 标志使 os.Stdout 变为无效句柄，
// io.MultiWriter 写 stdout 失败后会跳过 goFile，导致日志丢失。
// 如需 dev 模式 stdout 输出，请单独调用 log.Printf。
func (s *LogService) GoPrintf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.goLogger == nil {
		log.Printf(format, args...)
		return
	}
	s.goLogger.Printf(format, args...)
}

// WriteFrontend 由前端调用，把 console 日志写入 frontend.log。
// level 形如 "log"/"info"/"warn"/"error"。
func (s *LogService) WriteFrontend(level, msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.feLogger == nil {
		return nil
	}
	s.feLogger.Printf("[%s] %s", level, msg)
	return nil
}

// ReadGoLog 读取 go.log 全文返回。
func (s *LogService) ReadGoLog() (string, error) {
	s.mu.Lock()
	goPath := s.goPath
	s.mu.Unlock()
	data, err := os.ReadFile(goPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// ReadFrontendLog 读取 frontend.log 全文返回。
func (s *LogService) ReadFrontendLog() (string, error) {
	s.mu.Lock()
	fePath := s.fePath
	s.mu.Unlock()
	data, err := os.ReadFile(fePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// ClearGoLog 清空 go.log。
func (s *LogService) ClearGoLog() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.goFile != nil {
		_ = s.goFile.Sync()
	}
	return os.Truncate(s.goPath, 0)
}

// ClearFrontendLog 清空 frontend.log。
func (s *LogService) ClearFrontendLog() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.feFile != nil {
		_ = s.feFile.Sync()
	}
	return os.Truncate(s.fePath, 0)
}

// SetLogDir 切换日志目录：关闭旧文件 → 创建新目录 → 打开新文件。
// 空字符串回退到默认目录。返回新的日志目录绝对路径。
func (s *LogService) SetLogDir(newDir string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	target := newDir
	if target == "" {
		target = defaultLogDir()
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		abs = target
	}
	// 关闭旧文件
	s.closeLocked()
	s.logDir = abs
	s.openFiles()
	// 标准日志重定向到新文件（goFile 在前，windowsgui 模式 stdout 无效不影响文件写入）
	if s.goFile != nil {
		log.SetOutput(io.MultiWriter(s.goFile, os.Stdout))
	}
	return abs, nil
}

// GetLogDir 返回当前日志目录绝对路径。
func (s *LogService) GetLogDir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.logDir
}

// GetDefaultLogDir 返回默认日志目录路径，供前端显示提示。
func (s *LogService) GetDefaultLogDir() string {
	return defaultLogDir()
}

// Close 关闭文件句柄，main 退出时 defer 调用。
func (s *LogService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeLocked()
}

// closeLocked 关闭文件句柄，调用方需持锁。
func (s *LogService) closeLocked() {
	if s.goFile != nil {
		_ = s.goFile.Close()
		s.goFile = nil
		s.goLogger = nil
		s.stdLogger = nil
	}
	if s.feFile != nil {
		_ = s.feFile.Close()
		s.feFile = nil
		s.feLogger = nil
	}
}
