package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// RecycleItem 是 file_recycle 表的一条记录，对应回收站内的一个文件或目录。
type RecycleItem struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`         // 文件/目录名
	OriginalPath string `json:"originalPath"` // 删除前的绝对路径
	RecyclePath  string `json:"recyclePath"`  // 回收站中的实际存储路径
	IsDir        bool   `json:"isDir"`        // 项目类型：true=目录 false=文件
	Size         int64  `json:"size"`         // 字节；目录为递归总大小
	DeletedAt    int64  `json:"deletedAt"`    // 删除时间（unix 秒）
}

// RecycleService 实现应用内回收站：删除的文件/目录移动到回收站目录，
// 元数据记录在 system.db 的 file_recycle 表中，支持还原 / 还原到 /
// 彻底删除 / 清空。回收站目录可由用户在设置中自定义。
type RecycleService struct {
	mu  sync.Mutex
	db  *DBService
	cfg *ConfigService
}

// NewRecycleService 使用共享的 system.db 连接构造回收站服务。
// file_recycle 表由 SystemDBSchema 在 main.go 启动时统一创建。
func NewRecycleService(db *DBService, cfg *ConfigService) *RecycleService {
	return &RecycleService{db: db, cfg: cfg}
}

// Close 释放数据库连接（仅当服务持有时；共享连接由调用方管理）。
func (s *RecycleService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 共享连接不在此处关闭，由 main.go 统一管理
}

// DefaultRecycleDir 返回回收站默认目录 <userConfigDir>/XinText/recycle。
func DefaultRecycleDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base, _ = os.UserHomeDir()
	}
	return filepath.Join(base, "XinText", "recycle")
}

// resolveRecycleDir 读取配置并返回当前回收站绝对目录（空配置回退默认目录）。
func (s *RecycleService) resolveRecycleDir() string {
	dir := ""
	if s.cfg != nil {
		if cfg, err := s.cfg.GetConfig(); err == nil {
			dir = strings.TrimSpace(cfg.RecycleDir)
		}
	}
	if dir == "" {
		dir = DefaultRecycleDir()
	}
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	return dir
}

// GetRecycleDir 返回当前生效的回收站目录。
func (s *RecycleService) GetRecycleDir() (string, error) {
	return s.resolveRecycleDir(), nil
}

// SetRecycleDir 自定义回收站目录；空串表示恢复默认。目录不存在会自动创建，
// 创建失败时不持久化。返回设置后生效的绝对目录。
func (s *RecycleService) SetRecycleDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	resolved := dir
	if resolved == "" {
		resolved = DefaultRecycleDir()
	}
	if abs, err := filepath.Abs(resolved); err == nil {
		resolved = abs
	}
	if info, err := os.Stat(resolved); err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("回收站路径已被文件占用：%s", resolved)
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(resolved, 0o755); err != nil {
			return "", fmt.Errorf("创建回收站目录失败：%w", err)
		}
	} else {
		return "", err
	}
	if s.cfg != nil {
		cfg, err := s.cfg.GetConfig()
		if err != nil {
			return "", err
		}
		cfg.RecycleDir = dir
		if err := s.cfg.SetConfig(cfg); err != nil {
			return "", err
		}
	}
	return resolved, nil
}

// MoveToRecycle 把 srcPath 移动到回收站目录，并写入 file_recycle 记录。
// 跨卷（自定义目录位于其他盘）时 os.Rename 会失败，自动退回复制+删除。
func (s *RecycleService) MoveToRecycle(srcPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("回收站数据库未初始化")
	}
	info, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("源文件不存在：%s", srcPath)
	}
	root := s.resolveRecycleDir()
	container := filepath.Join(root, newToken())
	if err := os.MkdirAll(container, 0o755); err != nil {
		return fmt.Errorf("创建回收站目录失败：%w", err)
	}
	base := filepath.Base(filepath.Clean(srcPath))
	recyclePath := filepath.Join(container, base)

	size := info.Size()
	if info.IsDir() {
		size = dirSize(srcPath)
	}
	if err := movePathAcross(srcPath, recyclePath); err != nil {
		// 移动失败：清理刚创建的空容器
		os.Remove(container)
		return fmt.Errorf("移入回收站失败：%w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO file_recycle (name, original_path, recycle_path, is_dir, size, deleted_at) VALUES (?, ?, ?, ?, ?, ?)`,
		base, srcPath, recyclePath, info.IsDir(), size, time.Now().Unix(),
	)
	if err != nil {
		// 记录失败：尝试把文件移回原路径，避免数据与物理文件脱节
		_ = movePathAcross(recyclePath, srcPath)
		_ = os.Remove(container)
		return fmt.Errorf("写入回收站记录失败：%w", err)
	}
	return nil
}

// ListRecycleItems 返回回收站全部记录，按删除时间倒序。
func (s *RecycleService) ListRecycleItems() ([]RecycleItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, errors.New("回收站数据库未初始化")
	}
	rows, err := s.db.Query(
		`SELECT id, name, original_path, recycle_path, is_dir, size, deleted_at
		 FROM file_recycle ORDER BY deleted_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]RecycleItem, 0)
	for rows.Next() {
		var it RecycleItem
		var isDir int
		if err := rows.Scan(&it.ID, &it.Name, &it.OriginalPath, &it.RecyclePath, &isDir, &it.Size, &it.DeletedAt); err != nil {
			return nil, err
		}
		it.IsDir = isDir != 0
		items = append(items, it)
	}
	return items, rows.Err()
}

// RestoreRecycleItem 还原到原路径；原路径的父目录不存在时自动创建，
// 同名冲突按 “名称 (n)” 追加。还原成功后删除回收站记录。
func (s *RecycleService) RestoreRecycleItem(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, err := s.getItem(id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(it.RecyclePath); err != nil {
		return fmt.Errorf("回收站中的文件已不存在：%s", it.RecyclePath)
	}
	if err := os.MkdirAll(filepath.Dir(it.OriginalPath), 0o755); err != nil {
		return fmt.Errorf("创建原路径目录失败：%w", err)
	}
	dest := it.OriginalPath
	if _, err := os.Stat(dest); err == nil {
		dest = availableCopyPath(filepath.Dir(dest), filepath.Base(dest))
	}
	if err := movePathAcross(it.RecyclePath, dest); err != nil {
		return fmt.Errorf("还原失败：%w", err)
	}
	return s.afterItemRemoved(id, it.RecyclePath)
}

// RestoreRecycleItemTo 还原到指定目录（目录不存在会创建），同名冲突自动追加
// “名称 (n)”。还原成功后删除回收站记录。
func (s *RecycleService) RestoreRecycleItemTo(id int64, destDirPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, err := s.getItem(id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(destDirPath) == "" {
		return errors.New("目标文件夹不能为空")
	}
	if _, err := os.Stat(it.RecyclePath); err != nil {
		return fmt.Errorf("回收站中的文件已不存在：%s", it.RecyclePath)
	}
	if err := os.MkdirAll(destDirPath, 0o755); err != nil {
		return fmt.Errorf("创建目标文件夹失败：%w", err)
	}
	dest := availableCopyPath(destDirPath, it.Name)
	if err := movePathAcross(it.RecyclePath, dest); err != nil {
		return fmt.Errorf("还原失败：%w", err)
	}
	return s.afterItemRemoved(id, it.RecyclePath)
}

// DeleteRecycleItemPermanent 彻底删除单个项目并清除记录。
func (s *RecycleService) DeleteRecycleItemPermanent(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, err := s.getItem(id)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(it.RecyclePath); err != nil {
		return fmt.Errorf("彻底删除失败：%w", err)
	}
	return s.afterItemRemoved(id, it.RecyclePath)
}

// EmptyRecycle 彻底删除回收站中的全部项目并清空记录表。
// 单项物理删除失败时保留其记录并返回错误，其余项目继续清理。
func (s *RecycleService) EmptyRecycle() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("回收站数据库未初始化")
	}
	rows, err := s.db.Query(`SELECT id, recycle_path FROM file_recycle`)
	if err != nil {
		return err
	}
	type pair struct {
		id   int64
		path string
	}
	var list []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.id, &p.path); err != nil {
			rows.Close()
			return err
		}
		list = append(list, p)
	}
	rows.Close()

	var firstErr error
	for _, p := range list {
		if err := os.RemoveAll(p.path); err != nil && firstErr == nil {
			firstErr = err
			continue
		}
		if _, err := s.db.Exec(`DELETE FROM file_recycle WHERE id = ?`, p.id); err != nil && firstErr == nil {
			firstErr = err
		} else {
			removeContainerDir(p.path)
		}
	}
	// 清理空的 token 容器目录（记录已删但容器残留的情况）
	removeEmptyTokenDirs(s.resolveRecycleDir())
	return firstErr
}

// getItem 查询单条记录；不存在返回错误。
func (s *RecycleService) getItem(id int64) (RecycleItem, error) {
	if s.db == nil {
		return RecycleItem{}, errors.New("回收站数据库未初始化")
	}
	var it RecycleItem
	var isDir int
	err := s.db.QueryRow(
		`SELECT id, name, original_path, recycle_path, is_dir, size, deleted_at
		 FROM file_recycle WHERE id = ?`, id,
	).Scan(&it.ID, &it.Name, &it.OriginalPath, &it.RecyclePath, &isDir, &it.Size, &it.DeletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return RecycleItem{}, fmt.Errorf("回收站记录不存在（id=%d）", id)
	}
	if err != nil {
		return RecycleItem{}, err
	}
	it.IsDir = isDir != 0
	return it, nil
}

// afterItemRemoved 删除记录并清理空的 token 容器目录。
func (s *RecycleService) afterItemRemoved(id int64, recyclePath string) error {
	if _, err := s.db.Exec(`DELETE FROM file_recycle WHERE id = ?`, id); err != nil {
		return fmt.Errorf("清除回收站记录失败：%w", err)
	}
	removeContainerDir(recyclePath)
	return nil
}

// removeContainerDir 物理文件移除后，清理其所在的 token 容器目录（空目录才删）。
func removeContainerDir(recyclePath string) {
	container := filepath.Dir(recyclePath)
	if entries, err := os.ReadDir(container); err == nil && len(entries) == 0 {
		_ = os.Remove(container)
	}
}

// removeEmptyTokenDirs 扫描回收站根目录，清理空的 token 容器。
func removeEmptyTokenDirs(root string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(root, e.Name())
		if children, err := os.ReadDir(p); err == nil && len(children) == 0 {
			_ = os.Remove(p)
		}
	}
}

// newToken 生成回收站容器目录名：纳秒时间戳 + 4 字节随机后缀。
func newToken() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%d_%s", time.Now().UnixNano(), hex.EncodeToString(b[:]))
}

// dirSize 递归统计目录内全部常规文件大小之和。
func dirSize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

// movePathAcross 移动文件/目录；跨卷 Rename 失败时退回复制 + 删除。
func movePathAcross(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else {
		// 仅在跨设备等"可降级"错误时复制；其余错误直接返回
		var linkErr *os.LinkError
		if !errors.As(err, &linkErr) {
			return err
		}
	}
	if err := copyTree(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

// copyTree 递归复制文件/目录（跨卷移动的降级路径），保留文件权限位。
func copyTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyTree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		// 非常规文件（符号链接等）回收站场景直接跳过
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
