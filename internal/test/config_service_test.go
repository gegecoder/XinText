package test

import (
	"os"
	"path/filepath"
	"testing"

	"XinText/internal/model"
	"XinText/internal/service"
)

// newTestConfig 在临时目录构造隔离的配置服务（共享 system.db）。
func newTestConfig(t *testing.T) (*service.ConfigService, string) {
	t.Helper()
	root := t.TempDir()
	db, err := service.NewDBService(filepath.Join(root, "system.db"), service.SystemDBSchema)
	if err != nil {
		t.Fatalf("init system.db: %v", err)
	}
	cfg := service.NewConfigService(db, "")
	t.Cleanup(func() { db.Close() })
	return cfg, root
}

// TestConfigRoundTrip 写入再读出，所有字段类型正确还原。
func TestConfigRoundTrip(t *testing.T) {
	cfg, _ := newTestConfig(t)
	autoSave := true
	want := model.Config{
		WindowWidth:   1280,
		WindowHeight:  720,
		Theme:         "dark",
		Language:      "en",
		RecentFiles:   []string{"a.md", "b.md"},
		RecentFolders: []string{"D:/docs"},
		ProjectPath:   "D:/docs",
		PandocPath:    "C:/pandoc/pandoc.exe",
		ImageDir:      "D:/images",
		LogDir:        "D:/logs",
		RecycleDir:    "D:/recycle",
		AutoSave:      &autoSave,
	}
	if err := cfg.SetConfig(want); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	got, err := cfg.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if got.WindowWidth != want.WindowWidth || got.WindowHeight != want.WindowHeight {
		t.Errorf("size mismatch: got %dx%d, want %dx%d", got.WindowWidth, got.WindowHeight, want.WindowWidth, want.WindowHeight)
	}
	if got.Theme != want.Theme || got.Language != want.Language {
		t.Errorf("theme/lang mismatch: got %q/%q, want %q/%q", got.Theme, got.Language, want.Theme, want.Language)
	}
	if len(got.RecentFiles) != len(want.RecentFiles) || got.RecentFiles[0] != "a.md" {
		t.Errorf("recent files mismatch: %v", got.RecentFiles)
	}
	if got.AutoSave == nil || *got.AutoSave != true {
		t.Errorf("autoSave mismatch: got %v", got.AutoSave)
	}
	if got.PandocPath != want.PandocPath {
		t.Errorf("pandoc path mismatch: got %q", got.PandocPath)
	}
}

// TestConfigDefaults 空表时 GetConfig 返回默认值且不报错。
func TestConfigDefaults(t *testing.T) {
	cfg, _ := newTestConfig(t)
	got, err := cfg.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig on empty table: %v", err)
	}
	def := model.DefaultConfig()
	if got.WindowWidth != def.WindowWidth || got.Theme != def.Theme {
		t.Errorf("defaults not applied: %+v", got)
	}
}

// TestConfigTypedGetSet 单 key 读写方法（GetString/GetInt/GetBool/GetStrings）。
func TestConfigTypedGetSet(t *testing.T) {
	cfg, _ := newTestConfig(t)
	if err := cfg.SetString(service.ConfigKeyTheme, "dark"); err != nil {
		t.Fatal(err)
	}
	if got := cfg.GetString(service.ConfigKeyTheme); got != "dark" {
		t.Errorf("GetString = %q, want dark", got)
	}
	if err := cfg.SetInt(service.ConfigKeyWindowWidth, 1024); err != nil {
		t.Fatal(err)
	}
	if got := cfg.GetInt(service.ConfigKeyWindowWidth); got != 1024 {
		t.Errorf("GetInt = %d, want 1024", got)
	}
	if err := cfg.SetBool(service.ConfigKeyAutoSave, true); err != nil {
		t.Fatal(err)
	}
	if !cfg.GetBool(service.ConfigKeyAutoSave) {
		t.Error("GetBool = false, want true")
	}
	if err := cfg.SetStrings(service.ConfigKeyRecentFolders, []string{"x", "y"}); err != nil {
		t.Fatal(err)
	}
	if got := cfg.GetStrings(service.ConfigKeyRecentFolders); len(got) != 2 || got[0] != "x" {
		t.Errorf("GetStrings = %v", got)
	}
}

// TestConfigRecentFolders AddRecentFolder 去重 + 上限 10 + Remove/Clear。
func TestConfigRecentFolders(t *testing.T) {
	cfg, _ := newTestConfig(t)
	for i := 0; i < 12; i++ {
		if err := cfg.AddRecentFolder("p" + string(rune('a'+i))); err != nil {
			t.Fatal(err)
		}
	}
	list := cfg.GetStrings(service.ConfigKeyRecentFolders)
	if len(list) != 10 {
		t.Fatalf("want 10 recent folders, got %d", len(list))
	}
	// 最新添加的 "pl" 应在首位；最早的 "pa"/"pb" 已被挤出
	if list[0] != "pl" {
		t.Errorf("newest should be first, got %q", list[0])
	}
	if err := cfg.RemoveRecentFolder("pl"); err != nil {
		t.Fatal(err)
	}
	if len(cfg.GetStrings(service.ConfigKeyRecentFolders)) != 9 {
		t.Error("remove failed")
	}
	if err := cfg.ClearRecentFolders(); err != nil {
		t.Fatal(err)
	}
	if len(cfg.GetStrings(service.ConfigKeyRecentFolders)) != 0 {
		t.Error("clear failed")
	}
}

// TestConfigMigrateFromLegacyJSON 旧版 config.json 存在时自动迁移到 system_config。
func TestConfigMigrateFromLegacyJSON(t *testing.T) {
	root := t.TempDir()
	db, err := service.NewDBService(filepath.Join(root, "system.db"), service.SystemDBSchema)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()
	legacyPath := filepath.Join(root, "config.json")
	data := []byte(`{"windowWidth":999,"windowHeight":666,"theme":"dark","language":"zh"}`)
	if err := os.WriteFile(legacyPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := service.NewConfigService(db, legacyPath)
	if err := cfg.MigrateFromLegacyJSON(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	got, err := cfg.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig after migrate: %v", err)
	}
	if got.WindowWidth != 999 || got.Theme != "dark" {
		t.Errorf("migrate values not applied: %+v", got)
	}
	// 迁移后旧文件被删除
	if _, err := os.Stat(legacyPath); err == nil {
		t.Error("legacy config.json should be removed after migration")
	}
}
