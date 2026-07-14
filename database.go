package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/user/stone/monitor"
	_ "modernc.org/sqlite"
)

const (
	dbPath = "/var/lib/stone/stone.db"
)

// DB 数据库操作封装
type DB struct {
	conn *sql.DB
}

// SystemRecord 系统状态记录
type SystemRecord struct {
	ID        int64
	Timestamp int64
	CPU       float64
	Memory    float64
	Disk      float64
	NetRX     uint64
	NetTX     uint64
	Load1     float64
	Uptime    int64
}

// ActionLog 操作日志（增强版）
type ActionLog struct {
	ID           int64
	Timestamp    int64
	UserID       int64
	Username     string
	Action       string
	Target       string
	BeforeStatus string
	AfterStatus  string
	Result       string
	Error        string
}

// OpenDB 打开数据库连接，自动创建表
func OpenDB() (*DB, error) {
	dir := "/var/lib/stone"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	conn.SetMaxOpenConns(1)

	// 创建系统状态表
	schema := `
	CREATE TABLE IF NOT EXISTS system_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		cpu_usage REAL NOT NULL,
		memory_usage REAL NOT NULL,
		disk_usage REAL NOT NULL,
		net_rx INTEGER NOT NULL DEFAULT 0,
		net_tx INTEGER NOT NULL DEFAULT 0,
		load1 REAL NOT NULL DEFAULT 0,
		uptime INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON system_stats(timestamp);
	`
	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("create system_stats: %w", err)
	}

	// 迁移 action_logs 表
	var tableExists int
	if err := conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='action_logs'").Scan(&tableExists); err != nil {
		conn.Close()
		return nil, fmt.Errorf("check action_logs: %w", err)
	}

	needMigrate := false
	if tableExists > 0 {
		// 检查是否有 time 列（旧版本）
		var hasTimeCol int
		if err := conn.QueryRow("SELECT COUNT(*) FROM pragma_table_info('action_logs') WHERE name='time'").Scan(&hasTimeCol); err != nil {
			conn.Close()
			return nil, fmt.Errorf("check action_logs schema: %w", err)
		}
		if hasTimeCol > 0 {
			// 旧版本，需要迁移
			if _, err := conn.Exec("ALTER TABLE action_logs RENAME TO action_logs_old"); err != nil {
				conn.Close()
				return nil, fmt.Errorf("rename action_logs: %w", err)
			}
			needMigrate = true
		}
	}

	// 创建 action_logs 表
	actionSchema := `
	CREATE TABLE IF NOT EXISTS action_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL DEFAULT '',
		action TEXT NOT NULL,
		target TEXT NOT NULL DEFAULT '',
		before_status TEXT NOT NULL DEFAULT '',
		after_status TEXT NOT NULL DEFAULT '',
		result TEXT NOT NULL DEFAULT '',
		error TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_action_timestamp ON action_logs(timestamp);
	`
	if _, err := conn.Exec(actionSchema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("create action_logs: %w", err)
	}

	// 迁移旧数据
	if needMigrate {
		if _, err := conn.Exec("INSERT OR IGNORE INTO action_logs (id, timestamp, user_id, action, target, result) SELECT id, time, user_id, action, target, result FROM action_logs_old"); err != nil {
			logWarn("migrate action_logs data: %v", err)
		}
		if _, err := conn.Exec("DROP TABLE IF EXISTS action_logs_old"); err != nil {
			logWarn("drop action_logs_old: %v", err)
		}
	}

	// 创建安全事件表
	securitySchema := `
	CREATE TABLE IF NOT EXISTS security_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		type TEXT NOT NULL,
		source TEXT NOT NULL DEFAULT '',
		ip TEXT NOT NULL DEFAULT '',
		jail TEXT NOT NULL DEFAULT '',
		action TEXT NOT NULL,
		details TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_security_timestamp ON security_events(timestamp);
	`
	if _, err := conn.Exec(securitySchema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("create security_events: %w", err)
	}

	// 创建告警状态表
	alertStatesSchema := `
	CREATE TABLE IF NOT EXISTS alert_states (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server TEXT NOT NULL,
		category TEXT NOT NULL,
		name TEXT NOT NULL,
		current_state TEXT NOT NULL,
		previous_state TEXT NOT NULL DEFAULT '',
		updated_at INTEGER NOT NULL,
		last_notify INTEGER NOT NULL DEFAULT 0,
		UNIQUE(server, category, name)
	);
	CREATE INDEX IF NOT EXISTS idx_alert_states_server ON alert_states(server);
	`
	if _, err := conn.Exec(alertStatesSchema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("create alert_states: %w", err)
	}

	// 创建告警日志表
	alertLogsSchema := `
	CREATE TABLE IF NOT EXISTS alert_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		type TEXT NOT NULL,
		target TEXT NOT NULL,
		old_state TEXT NOT NULL DEFAULT '',
		new_state TEXT NOT NULL,
		message TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_alert_logs_timestamp ON alert_logs(timestamp);
	`
	if _, err := conn.Exec(alertLogsSchema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("create alert_logs: %w", err)
	}

	// 创建服务器表
	serversSchema := `
	CREATE TABLE IF NOT EXISTS servers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server_id TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL DEFAULT '',
		location TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL DEFAULT '',
		version TEXT NOT NULL DEFAULT '',
		last_seen INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'offline'
	);
	CREATE INDEX IF NOT EXISTS idx_servers_server_id ON servers(server_id);
	`
	if _, err := conn.Exec(serversSchema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("create servers: %w", err)
	}

	return &DB{conn: conn}, nil
}

// InsertStats 写入一条系统状态记录
func (db *DB) InsertStats(status *monitor.SystemStatus, netRX, netTX uint64, uptimeSec int64) error {
	_, err := db.conn.Exec(`
		INSERT INTO system_stats (timestamp, cpu_usage, memory_usage, disk_usage, net_rx, net_tx, load1, uptime)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		time.Now().Unix(), status.CPU, status.MemoryPercent, status.DiskPercent,
		netRX, netTX, parseLoad1(status.Load), uptimeSec,
	)
	return err
}

// CleanOldRecords 删除过期记录
func (db *DB) CleanOldRecords(retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = 90
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays).Unix()
	_, err := db.conn.Exec("DELETE FROM system_stats WHERE timestamp < ?", cutoff)
	return err
}

// GetRecordCount 获取记录总数
func (db *DB) GetRecordCount() (int64, error) {
	var count int64
	err := db.conn.QueryRow("SELECT COUNT(*) FROM system_stats").Scan(&count)
	return count, err
}

// GetTodayTraffic 获取今日流量
func (db *DB) GetTodayTraffic() (rxDelta, txDelta uint64, err error) {
	today := time.Now().Truncate(24 * time.Hour).Unix()

	var baseRX, baseTX uint64
	err = db.conn.QueryRow(`
		SELECT net_rx, net_tx FROM system_stats
		WHERE timestamp >= ? ORDER BY timestamp ASC LIMIT 1`, today).Scan(&baseRX, &baseTX)
	if err != nil {
		return 0, 0, err
	}

	var curRX, curTX uint64
	err = db.conn.QueryRow(`
		SELECT net_rx, net_tx FROM system_stats
		ORDER BY timestamp DESC LIMIT 1`).Scan(&curRX, &curTX)
	if err != nil {
		return 0, 0, err
	}

	if curRX >= baseRX {
		rxDelta = curRX - baseRX
	}
	if curTX >= baseTX {
		txDelta = curTX - baseTX
	}
	return rxDelta, txDelta, nil
}

// GetPeriodTraffic 获取周期内总流量
func (db *DB) GetPeriodTraffic() (rxDelta, txDelta uint64, err error) {
	var baseRX, baseTX uint64
	err = db.conn.QueryRow(`
		SELECT net_rx, net_tx FROM system_stats ORDER BY timestamp ASC LIMIT 1`).Scan(&baseRX, &baseTX)
	if err != nil {
		return 0, 0, err
	}

	var curRX, curTX uint64
	err = db.conn.QueryRow(`
		SELECT net_rx, net_tx FROM system_stats ORDER BY timestamp DESC LIMIT 1`).Scan(&curRX, &curTX)
	if err != nil {
		return 0, 0, err
	}

	if curRX >= baseRX {
		rxDelta = curRX - baseRX
	}
	if curTX >= baseTX {
		txDelta = curTX - baseTX
	}
	return rxDelta, txDelta, nil
}

// GetDBSize 获取数据库文件大小
func GetDBSize() (int64, error) {
	info, err := os.Stat(dbPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// InsertActionLog 写入操作日志（增强版）
func (db *DB) InsertActionLog(userID int64, username, action, target, beforeStatus, afterStatus, result, errMsg string) error {
	_, err := db.conn.Exec(`
		INSERT INTO action_logs (timestamp, user_id, username, action, target, before_status, after_status, result, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		time.Now().Unix(), userID, username, action, target, beforeStatus, afterStatus, result, errMsg,
	)
	return err
}

// GetRecentActionLogs 获取最近的操作日志
func (db *DB) GetRecentActionLogs(limit int) ([]ActionLog, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := db.conn.Query(`
		SELECT id, timestamp, user_id, username, action, target, before_status, after_status, result, error
		FROM action_logs
		ORDER BY timestamp DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []ActionLog
	for rows.Next() {
		var log ActionLog
		if err := rows.Scan(&log.ID, &log.Timestamp, &log.UserID, &log.Username, &log.Action, &log.Target, &log.BeforeStatus, &log.AfterStatus, &log.Result, &log.Error); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

// Close 关闭数据库
func (db *DB) Close() error {
	return db.conn.Close()
}

// InsertSecurityEvent 写入安全事件
func (db *DB) InsertSecurityEvent(eventType, source, ip, jail, action, details string) error {
	_, err := db.conn.Exec(`
		INSERT INTO security_events (timestamp, type, source, ip, jail, action, details)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		time.Now().Unix(), eventType, source, ip, jail, action, details,
	)
	return err
}

// GetRecentSecurityEvents 获取最近的安全事件
func (db *DB) GetRecentSecurityEvents(limit int) ([]SecurityEvent, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := db.conn.Query(`
		SELECT id, timestamp, type, source, ip, jail, action, details
		FROM security_events
		ORDER BY timestamp DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []SecurityEvent
	for rows.Next() {
		var event SecurityEvent
		if err := rows.Scan(&event.ID, &event.Timestamp, &event.Type, &event.Source, &event.IP, &event.Jail, &event.Action, &event.Details); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

// GetSecurityStats 获取安全统计
func (db *DB) GetSecurityStats() (failed int, bans int, err error) {
	err = db.conn.QueryRow("SELECT COUNT(*) FROM security_events WHERE action = 'ban'").Scan(&bans)
	if err != nil {
		return 0, 0, err
	}

	// 从auth.log获取失败次数
	// 这里简化处理，实际可以从fail2ban获取
	return 0, bans, nil
}

// SecurityEvent 安全事件
type SecurityEvent struct {
	ID        int64
	Timestamp int64
	Type      string
	Source    string
	IP        string
	Jail      string
	Action    string
	Details   string
}

// parseLoad1 从 "0.45 0.30 0.20" 格式中提取第一个值
func parseLoad1(load string) float64 {
	var v float64
	fmt.Sscanf(load, "%f", &v)
	return v
}
