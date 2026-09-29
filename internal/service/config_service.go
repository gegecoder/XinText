package service

import (
	"encoding/json"
	"os"
	"time"

	"XinText/internal/i18n"
	"XinText/internal/model"
)

// ConfigService 持久化应用配置到 system.db 的 system_config 表。
// 采用 key-value 设计：Config 结构体的每个字段对应一条记录，value 为
// JSON 编码后的值，可统一存储 string / int / bool / []string。
// 新增配置项只需插入新 key，无需改表结构（可拓展）。
type ConfigService struct {
	db               *DBService
	legacyConfigPath string // 旧版 config.json 路径，首次启动迁移用；空=跳过
}

// NewConfigService 用共享的 system.db 连接构造配置服务。
// legacyPath 是旧版 config.json 的路径（首次启动时迁移用），可为空。
func NewConfigService(db *DBService, legacyPath string) *ConfigService {
	return &ConfigService{db: db, legacyConfigPath: legacyPath}
}

// 配置项 key 常量，集中定义便于检索与避免拼写错误。
const (
	ConfigKeyWindowWidth    = "window_width"
	ConfigKeyWindowHeight   = "window_height"
	ConfigKeyTheme          = "theme"
	ConfigKeyLanguage       = "language"
	ConfigKeyRecentFiles    = "recent_files"
	ConfigKeyRecentFolders  = "recent_folders"
	ConfigKeyProjectPath    = "project_path"
	ConfigKeyPandocPath     = "pandoc_path"
	ConfigKeyImageDir       = "image_dir"
	ConfigKeyLogDir         = "log_dir"
	ConfigKeyRecycleDir     = "recycle_dir"
	ConfigKeyBrowserHomeURL = "browser_home_url"
	ConfigKeyAutoSave       = "auto_save"
)

// configDescriptions 记录每个配置项的业务说明，写入 description 列便于
// 直接查库时理解用途（业务清晰）。
var configDescriptions = map[string]string{
	ConfigKeyWindowWidth:    "窗口宽度（像素）",
	ConfigKeyWindowHeight:   "窗口高度（像素）",
	ConfigKeyTheme:          "界面主题：light / dark",
	ConfigKeyLanguage:       "界面语言：zh / en，空=跟随系统",
	ConfigKeyRecentFiles:    "最近打开的文件列表",
	ConfigKeyRecentFolders:  "最近打开的文件夹列表",
	ConfigKeyProjectPath:    "当前打开的项目目录",
	ConfigKeyPandocPath:     "pandoc 可执行文件路径",
	ConfigKeyImageDir:       "图片统一保存目录，空=默认",
	ConfigKeyLogDir:         "日志目录，空=默认",
	ConfigKeyRecycleDir:     "回收站目录，空=默认",
	ConfigKeyBrowserHomeURL: "浏览器标签页主页 URL，空=内置默认",
	ConfigKeyAutoSave:       "自动保存开关：编辑后 10s 无操作自动写盘",
}

// GetConfig 从 system_config 表读取全部配置，缺失字段用默认值兜底。
func (s *ConfigService) GetConfig() (model.Config, error) {
	cfg := model.DefaultConfig()
	rows, err := s.db.Query("SELECT key, value FROM system_config")
	if err != nil {
		return cfg, err
	}
	defer rows.Close()
	vals := make(map[string]string, 16)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return cfg, err
		}
		vals[k] = v
	}
	if err := rows.Err(); err != nil {
		return cfg, err
	}

	readInt(vals, ConfigKeyWindowWidth, &cfg.WindowWidth)
	readInt(vals, ConfigKeyWindowHeight, &cfg.WindowHeight)
	readString(vals, ConfigKeyTheme, &cfg.Theme)
	readString(vals, ConfigKeyLanguage, &cfg.Language)
	readStrings(vals, ConfigKeyRecentFiles, &cfg.RecentFiles)
	readStrings(vals, ConfigKeyRecentFolders, &cfg.RecentFolders)
	readString(vals, ConfigKeyProjectPath, &cfg.ProjectPath)
	readString(vals, ConfigKeyPandocPath, &cfg.PandocPath)
	readString(vals, ConfigKeyImageDir, &cfg.ImageDir)
	readString(vals, ConfigKeyLogDir, &cfg.LogDir)
	readString(vals, ConfigKeyRecycleDir, &cfg.RecycleDir)
	readString(vals, ConfigKeyBrowserHomeURL, &cfg.BrowserHomeURL)
	if v, ok := vals[ConfigKeyAutoSave]; ok {
		var b bool
		if json.Unmarshal([]byte(v), &b) == nil {
			cfg.AutoSave = &b
		}
	}

	def := model.DefaultConfig()
	if cfg.WindowWidth == 0 {
		cfg.WindowWidth = def.WindowWidth
	}
	if cfg.WindowHeight == 0 {
		cfg.WindowHeight = def.WindowHeight
	}
	if cfg.Theme == "" {
		cfg.Theme = def.Theme
	}
	if cfg.RecentFiles == nil {
		cfg.RecentFiles = def.RecentFiles
	}
	if cfg.RecentFolders == nil {
		cfg.RecentFolders = def.RecentFolders
	}
	if cfg.AutoSave == nil {
		cfg.AutoSave = def.AutoSave
	}
	if cfg.BrowserHomeURL == "" {
		cfg.BrowserHomeURL = def.BrowserHomeURL
	}
	cfg.Language = i18n.ResolveLanguage(cfg.Language)
	return cfg, nil
}

// SetConfig 把整个 Config 写入 system_config 表（逐条 upsert）。
func (s *ConfigService) SetConfig(cfg model.Config) error {
	if err := s.SetInt(ConfigKeyWindowWidth, cfg.WindowWidth); err != nil {
		return err
	}
	if err := s.SetInt(ConfigKeyWindowHeight, cfg.WindowHeight); err != nil {
		return err
	}
	if err := s.SetString(ConfigKeyTheme, cfg.Theme); err != nil {
		return err
	}
	if err := s.SetString(ConfigKeyLanguage, cfg.Language); err != nil {
		return err
	}
	if err := s.SetStrings(ConfigKeyRecentFiles, cfg.RecentFiles); err != nil {
		return err
	}
	if err := s.SetStrings(ConfigKeyRecentFolders, cfg.RecentFolders); err != nil {
		return err
	}
	if err := s.SetString(ConfigKeyProjectPath, cfg.ProjectPath); err != nil {
		return err
	}
	if err := s.SetString(ConfigKeyPandocPath, cfg.PandocPath); err != nil {
		return err
	}
	if err := s.SetString(ConfigKeyImageDir, cfg.ImageDir); err != nil {
		return err
	}
	if err := s.SetString(ConfigKeyLogDir, cfg.LogDir); err != nil {
		return err
	}
	if err := s.SetString(ConfigKeyRecycleDir, cfg.RecycleDir); err != nil {
		return err
	}
	if err := s.SetString(ConfigKeyBrowserHomeURL, cfg.BrowserHomeURL); err != nil {
		return err
	}
	autoSave := false
	if cfg.AutoSave != nil {
		autoSave = *cfg.AutoSave
	}
	return s.SetBool(ConfigKeyAutoSave, autoSave)
}

// —— 通用单 key 读写方法（集成友好，业务模块可直接按 key 操作） ——

// GetString 读取字符串配置；不存在返回空字符串。
func (s *ConfigService) GetString(key string) string {
	var v string
	if err := s.db.QueryRow("SELECT value FROM system_config WHERE key = ?", key).Scan(&v); err != nil {
		return ""
	}
	var out string
	if json.Unmarshal([]byte(v), &out) == nil {
		return out
	}
	return ""
}

// SetString 写入字符串配置。
func (s *ConfigService) SetString(key, value string) error {
	return s.upsert(key, mustJSON(value))
}

// GetInt 读取整数配置；不存在返回 0。
func (s *ConfigService) GetInt(key string) int {
	var v string
	if err := s.db.QueryRow("SELECT value FROM system_config WHERE key = ?", key).Scan(&v); err != nil {
		return 0
	}
	var n int
	if json.Unmarshal([]byte(v), &n) == nil {
		return n
	}
	return 0
}

// SetInt 写入整数配置。
func (s *ConfigService) SetInt(key string, value int) error {
	return s.upsert(key, mustJSON(value))
}

// GetBool 读取布尔配置；不存在返回 false。
func (s *ConfigService) GetBool(key string) bool {
	var v string
	if err := s.db.QueryRow("SELECT value FROM system_config WHERE key = ?", key).Scan(&v); err != nil {
		return false
	}
	var b bool
	if json.Unmarshal([]byte(v), &b) == nil {
		return b
	}
	return false
}

// SetBool 写入布尔配置。
func (s *ConfigService) SetBool(key string, value bool) error {
	return s.upsert(key, mustJSON(value))
}

// GetStrings 读取字符串切片配置；不存在返回 nil。
func (s *ConfigService) GetStrings(key string) []string {
	var v string
	if err := s.db.QueryRow("SELECT value FROM system_config WHERE key = ?", key).Scan(&v); err != nil {
		return nil
	}
	var arr []string
	if json.Unmarshal([]byte(v), &arr) == nil {
		return arr
	}
	return nil
}

// SetStrings 写入字符串切片配置。
func (s *ConfigService) SetStrings(key string, value []string) error {
	if value == nil {
		value = []string{}
	}
	return s.upsert(key, mustJSON(value))
}

// upsert 写入或更新一条配置记录。
func (s *ConfigService) upsert(key, value string) error {
	desc := configDescriptions[key]
	_, err := s.db.Exec(
		`INSERT INTO system_config (key, value, description, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
		key, value, desc, time.Now().Unix(),
	)
	return err
}

// mustJSON 将任意值序列化为 JSON 字符串，失败返回 "null"。
func mustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

// —— map 反序列化辅助 ——

func readInt(vals map[string]string, key string, out *int) {
	if v, ok := vals[key]; ok {
		_ = json.Unmarshal([]byte(v), out)
	}
}

func readString(vals map[string]string, key string, out *string) {
	if v, ok := vals[key]; ok {
		_ = json.Unmarshal([]byte(v), out)
	}
}

func readStrings(vals map[string]string, key string, out *[]string) {
	if v, ok := vals[key]; ok {
		_ = json.Unmarshal([]byte(v), out)
	}
}

// —— 近期文件 / 文件夹列表操作 ——

// AddRecentFile 将 path 加入近期文件列表（去重、上限 10）。
func (s *ConfigService) AddRecentFile(path string) error {
	if path == "" {
		return nil
	}
	list := s.GetStrings(ConfigKeyRecentFiles)
	filtered := make([]string, 0, 10)
	filtered = append(filtered, path)
	for _, p := range list {
		if p != path {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) > 10 {
		filtered = filtered[:10]
	}
	return s.SetStrings(ConfigKeyRecentFiles, filtered)
}

// AddRecentFolder 将 path 加入近期文件夹列表（去重、上限 10）。
func (s *ConfigService) AddRecentFolder(path string) error {
	if path == "" {
		return nil
	}
	list := s.GetStrings(ConfigKeyRecentFolders)
	filtered := make([]string, 0, 10)
	filtered = append(filtered, path)
	for _, p := range list {
		if p != path {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) > 10 {
		filtered = filtered[:10]
	}
	return s.SetStrings(ConfigKeyRecentFolders, filtered)
}

// RemoveRecentFolder 从近期文件夹列表移除 path。
func (s *ConfigService) RemoveRecentFolder(path string) error {
	list := s.GetStrings(ConfigKeyRecentFolders)
	filtered := make([]string, 0, len(list))
	for _, p := range list {
		if p != path {
			filtered = append(filtered, p)
		}
	}
	return s.SetStrings(ConfigKeyRecentFolders, filtered)
}

// ClearRecentFolders 清空近期文件夹列表。
func (s *ConfigService) ClearRecentFolders() error {
	return s.SetStrings(ConfigKeyRecentFolders, []string{})
}

// PathExists 判断路径在文件系统中是否存在。
func (s *ConfigService) PathExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// MigrateFromLegacyJSON 若旧版 config.json 存在且 system_config 尚无记录，
// 把它的字段逐条导入 system_config，然后删除旧文件。幂等：表中已有记录则跳过。
func (s *ConfigService) MigrateFromLegacyJSON() error {
	if s.legacyConfigPath == "" {
		return nil
	}
	data, err := os.ReadFile(s.legacyConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM system_config").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	if err := s.SetConfig(cfg); err != nil {
		return err
	}
	_ = os.Remove(s.legacyConfigPath)
	return nil
}
