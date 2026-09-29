package test

import (
	"os"
	"path/filepath"
	"testing"

	"XinText/internal/service"
)

// newTestService 在临时目录中构造隔离的回收站服务（共享 system.db + 配置）。
func newTestService(t *testing.T) (*service.RecycleService, string) {
	t.Helper()
	root := t.TempDir()
	appData := filepath.Join(root, "appdata")
	db, err := service.NewDBService(filepath.Join(appData, "system.db"), service.SystemDBSchema)
	if err != nil {
		t.Fatalf("init system.db: %v", err)
	}
	cfg := service.NewConfigService(db, "")
	svc := service.NewRecycleService(db, cfg)
	// 默认回收站目录也指向临时目录，避免触碰真实用户 AppData
	if _, err := svc.SetRecycleDir(filepath.Join(root, "recycle")); err != nil {
		t.Fatalf("seed recycle dir: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return svc, root
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertExists(t *testing.T, path string, want bool) {
	t.Helper()
	_, err := os.Stat(path)
	got := err == nil
	if got != want {
		t.Fatalf("exists(%s) = %v, want %v (err=%v)", path, got, want, err)
	}
}

// 首次构造应自动创建 system.db 与 file_recycle 表（无记录）。
func TestInitCreatesDBAndTable(t *testing.T) {
	svc, root := newTestService(t)
	if _, err := os.Stat(filepath.Join(root, "appdata", "system.db")); err != nil {
		t.Fatalf("system.db not created: %v", err)
	}
	items, err := svc.ListRecycleItems()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("want empty recycle, got %d", len(items))
	}
}

// 移入回收站后原路径消失、列表多一条；还原后文件回来且记录清除。
func TestMoveAndRestoreFile(t *testing.T) {
	svc, root := newTestService(t)
	src := filepath.Join(root, "docs", "a.md")
	writeFile(t, src, "hello")

	if err := svc.MoveToRecycle(src); err != nil {
		t.Fatal(err)
	}
	assertExists(t, src, false)

	items, err := svc.ListRecycleItems()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	it := items[0]
	if it.Name != "a.md" || it.OriginalPath != src || it.IsDir || it.Size != 5 {
		t.Fatalf("unexpected item: %+v", it)
	}
	recycleContainer := filepath.Dir(it.RecyclePath)

	if err := svc.RestoreRecycleItem(it.ID); err != nil {
		t.Fatal(err)
	}
	assertExists(t, src, true)
	items, _ = svc.ListRecycleItems()
	if len(items) != 0 {
		t.Fatalf("record not cleared after restore: %d", len(items))
	}
	// 还原后 token 容器目录应被清理，不留空目录
	assertExists(t, recycleContainer, false)
}

// 原路径父目录已不存在时，还原应自动重建。
func TestRestoreRecreatesMissingParent(t *testing.T) {
	svc, root := newTestService(t)
	src := filepath.Join(root, "deep", "nested", "b.txt")
	writeFile(t, src, "x")
	if err := svc.MoveToRecycle(src); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "deep")); err != nil {
		t.Fatal(err)
	}
	if err := svc.RestoreRecycleItem(1); err != nil {
		t.Fatal(err)
	}
	assertExists(t, src, true)
}

// 还原时原路径已被同名文件占用：应追加 “名称 (1)”。
func TestRestoreNameCollision(t *testing.T) {
	svc, root := newTestService(t)
	src := filepath.Join(root, "c.md")
	writeFile(t, src, "old")
	if err := svc.MoveToRecycle(src); err != nil {
		t.Fatal(err)
	}
	writeFile(t, src, "new") // 原位置出现同名文件

	if err := svc.RestoreRecycleItem(1); err != nil {
		t.Fatal(err)
	}
	collision := filepath.Join(root, "c (1).md")
	assertExists(t, collision, true)
}

// 还原到指定目录；目录不存在时自动创建。
func TestRestoreTo(t *testing.T) {
	svc, root := newTestService(t)
	src := filepath.Join(root, "d.md")
	writeFile(t, src, "z")
	if err := svc.MoveToRecycle(src); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "target", "sub")
	if err := svc.RestoreRecycleItemTo(1, dest); err != nil {
		t.Fatal(err)
	}
	assertExists(t, filepath.Join(dest, "d.md"), true)
	items, _ := svc.ListRecycleItems()
	if len(items) != 0 {
		t.Fatalf("want record cleared, got %d", len(items))
	}
}

// 目录整体移入、递归大小统计、还原。
func TestMoveAndRestoreDirectory(t *testing.T) {
	svc, root := newTestService(t)
	dir := filepath.Join(root, "proj")
	writeFile(t, filepath.Join(dir, "f1.md"), "12345")     // 5 bytes
	writeFile(t, filepath.Join(dir, "sub", "f2.md"), "12") // 2 bytes
	if err := svc.MoveToRecycle(dir); err != nil {
		t.Fatal(err)
	}
	assertExists(t, dir, false)
	items, _ := svc.ListRecycleItems()
	if len(items) != 1 || !items[0].IsDir || items[0].Size != 7 {
		t.Fatalf("unexpected dir item: %+v", items)
	}
	if err := svc.RestoreRecycleItem(items[0].ID); err != nil {
		t.Fatal(err)
	}
	assertExists(t, filepath.Join(dir, "sub", "f2.md"), true)
}

// 彻底删除：物理文件与记录一并消失。
func TestDeletePermanent(t *testing.T) {
	svc, root := newTestService(t)
	src := filepath.Join(root, "e.md")
	writeFile(t, src, "q")
	if err := svc.MoveToRecycle(src); err != nil {
		t.Fatal(err)
	}
	items, _ := svc.ListRecycleItems()
	stored := items[0].RecyclePath
	assertExists(t, stored, true)

	if err := svc.DeleteRecycleItemPermanent(items[0].ID); err != nil {
		t.Fatal(err)
	}
	assertExists(t, stored, false)
	items, _ = svc.ListRecycleItems()
	if len(items) != 0 {
		t.Fatalf("want empty, got %d", len(items))
	}
}

// 清空回收站：物理文件与全部记录消失。
func TestEmptyRecycle(t *testing.T) {
	svc, root := newTestService(t)
	for _, n := range []string{"g.md", "h.md"} {
		writeFile(t, filepath.Join(root, n), n)
		if err := svc.MoveToRecycle(filepath.Join(root, n)); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.EmptyRecycle(); err != nil {
		t.Fatal(err)
	}
	items, _ := svc.ListRecycleItems()
	if len(items) != 0 {
		t.Fatalf("want empty, got %d", len(items))
	}
}

// 自定义回收站目录：设置后新删除的文件进入该目录。
func TestSetRecycleDir(t *testing.T) {
	svc, root := newTestService(t)
	custom := filepath.Join(root, "custom-bin")
	got, err := svc.SetRecycleDir(custom)
	if err != nil {
		t.Fatal(err)
	}
	if got != custom {
		t.Fatalf("resolved dir = %s, want %s", got, custom)
	}
	src := filepath.Join(root, "i.md")
	writeFile(t, src, "i")
	if err := svc.MoveToRecycle(src); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(custom)
	if err != nil {
		t.Fatalf("custom recycle dir not used: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 token dir in custom bin, got %d", len(entries))
	}

	// 切换到另一个自定义目录：新删除的文件进入新目录
	custom2 := filepath.Join(root, "custom-bin-2")
	if _, err := svc.SetRecycleDir(custom2); err != nil {
		t.Fatal(err)
	}
	src2 := filepath.Join(root, "j.md")
	writeFile(t, src2, "j")
	if err := svc.MoveToRecycle(src2); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(custom2); err != nil || len(entries) != 1 {
		t.Fatalf("new custom bin not used (entries=%d, err=%v)", len(entries), err)
	}
	// 旧目录中之前的项目仍在（记录表保留绝对路径，不受换目录影响）
	if entries, err := os.ReadDir(custom); err != nil || len(entries) != 1 {
		t.Fatalf("old custom bin changed unexpectedly (entries=%d, err=%v)", len(entries), err)
	}
}
