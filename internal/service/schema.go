package service

// SystemDBSchema 是 system.db 的完整建表语句，集中维护所有持久化表，
// 由 main.go 在启动时一次性执行（CREATE TABLE IF NOT EXISTS 幂等）。
// 各业务服务（ConfigService、RecycleService）共享同一个 *DBService，
// 避免多连接竞争同一 SQLite 文件。
const SystemDBSchema = `
CREATE TABLE IF NOT EXISTS file_recycle (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	name          TEXT    NOT NULL,
	original_path TEXT    NOT NULL,
	recycle_path  TEXT    NOT NULL,
	is_dir        INTEGER NOT NULL DEFAULT 0,
	size          INTEGER NOT NULL DEFAULT 0,
	deleted_at    INTEGER NOT NULL
);

-- system_config：应用配置键值表。
-- 每条记录对应 Config 结构体的一个字段，value 为 JSON 编码后的值，
-- 因此可统一存储 string / int / bool / []string 等任意类型；
-- 新增配置项只需插入新 key，无需改表结构（可拓展）。
CREATE TABLE IF NOT EXISTS system_config (
	key         TEXT    PRIMARY KEY,
	value       TEXT    NOT NULL DEFAULT '',
	description TEXT    NOT NULL DEFAULT '',
	updated_at  INTEGER NOT NULL
);
`
