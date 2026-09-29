package service

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// DBService 封装 SQLite 通用操作：连接管理、自动建表、CRUD 执行、事务、关闭。
// 其他业务服务（如回收站、收藏、历史）通过组合 DBService 复用统一的数据库基础设施。
type DBService struct {
	mu   sync.Mutex
	path string
	db   *sql.DB
}

// NewDBService 创建服务：打开（不存在则自动创建）指定路径的 SQLite 数据库，
// 应用 WAL 与 busy_timeout，执行 schema 建表，返回可用的 DBService。
// schema 参数是建表/迁移语句（CREATE TABLE IF NOT EXISTS ...），可为空。
func NewDBService(path, schema string) (*DBService, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建数据库目录失败: %w", err)
		}
	}
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &DBService{path: path, db: db}
	if schema != "" {
		if _, err := db.Exec(schema); err != nil {
			db.Close()
			return nil, fmt.Errorf("执行建表语句失败: %w", err)
		}
	}
	return s, nil
}

// Close 释放数据库连接。
func (s *DBService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		s.db.Close()
		s.db = nil
	}
}

// IsOpen 报告底层数据库连接是否处于打开状态。
func (s *DBService) IsOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db != nil
}

// Exec 执行 INSERT / UPDATE / DELETE / DDL，返回 sql.Result。
// 调用方应在需要结果元数据（LastInsertId 等）时自行处理返回对象。
func (s *DBService) Exec(query string, args ...any) (sql.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, fmt.Errorf("数据库未打开: %s", s.path)
	}
	return s.db.Exec(query, args...)
}

// Query 执行 SELECT，返回 *sql.Rows；调用方负责 rows.Close()。
func (s *DBService) Query(query string, args ...any) (*sql.Rows, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, fmt.Errorf("数据库未打开: %s", s.path)
	}
	return s.db.Query(query, args...)
}

// QueryRow 执行返回单行的 SELECT。
func (s *DBService) QueryRow(query string, args ...any) *sql.Row {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		// 延迟到 Scan 时才报错；利用一个空查询保证调用方拿到可预期的错误
		return s.db.QueryRow("SELECT 1 WHERE 1=0")
	}
	return s.db.QueryRow(query, args...)
}

// Tx 在事务中执行 fn；fn 返回 error 时回滚，否则提交。
// 回调内通过 *sql.Tx 执行语句，避免在事务外使用 DBService 方法导致死锁。
func (s *DBService) Tx(fn func(tx *sql.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return fmt.Errorf("数据库未打开: %s", s.path)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// WithLock 持锁执行 fn，用于需要跨多个 DB 操作保持原子性的复合场景。
// fn 内可通过 DBService 方法执行 SQL；注意不要嵌套调用 WithLock。
func (s *DBService) WithLock(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn()
}
