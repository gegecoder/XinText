package model

// Config holds persisted application preferences.
// It is stored as JSON under the OS user config dir (e.g. %AppData%/XinText).
type Config struct {
	WindowWidth   int      `json:"windowWidth"`
	WindowHeight  int      `json:"windowHeight"`
	Theme         string   `json:"theme"`
	Language      string   `json:"language"` // 界面语言：zh / en；空 = 跟随系统
	RecentFiles   []string `json:"recentFiles"`
	RecentFolders []string `json:"recentFolders"`
	ProjectPath   string   `json:"projectPath"`
	PandocPath    string   `json:"pandocPath"`
	ImageDir      string   `json:"imageDir"`   // 图片统一保存目录；空 = <userConfigDir>/XinText/images
	LogDir        string   `json:"logDir"`     // 日志目录；空 = <userConfigDir>/XinText/logs
	RecycleDir    string   `json:"recycleDir"` // 回收站目录；空 = <userConfigDir>/XinText/recycle
	// 浏览器标签页主页 URL；空 = 内置默认（kevin.blog.csdn.net）
	BrowserHomeURL string `json:"browserHomeURL"`
	// 自动保存：编辑后 10s 无操作自动写盘；nil/未设置 = 关闭（默认 false）
	AutoSave *bool `json:"autoSave,omitempty"`
}

// DefaultConfig returns the built-in default configuration.
func DefaultConfig() Config {
	autoSave := false
	return Config{
		WindowWidth:    1400,
		WindowHeight:   800,
		Theme:          "light",
		RecentFiles:    []string{},
		RecentFolders:  []string{},
		BrowserHomeURL: "https://kevin.blog.csdn.net",
		AutoSave:       &autoSave,
	}
}
