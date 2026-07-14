package server

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Heartbeat 心跳管理器
type Heartbeat struct {
	db       *sql.DB
	serverID string
	interval time.Duration
}

// NewHeartbeat 创建心跳管理器
func NewHeartbeat(db *sql.DB, serverID string, intervalSec int) *Heartbeat {
	if intervalSec <= 0 {
		intervalSec = 60
	}
	return &Heartbeat{
		db:       db,
		serverID: serverID,
		interval: time.Duration(intervalSec) * time.Second,
	}
}

// Start 启动心跳
func (h *Heartbeat) Start(ctx context.Context) {
	h.loop(ctx)
}

func (h *Heartbeat) loop(ctx context.Context) {
	// 立即发送一次心跳
	h.send()

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[INFO] 心跳已停止")
			return
		case <-ticker.C:
			h.send()
		}
	}
}

// Send 发送心跳（公开方法）
func (h *Heartbeat) send() {
	if h.db == nil {
		return
	}

	_, err := h.db.Exec(`
		INSERT INTO servers (server_id, last_seen, status)
		VALUES (?, ?, 'online')
		ON CONFLICT(server_id) DO UPDATE SET
			last_seen = excluded.last_seen,
			status = 'online'`,
		h.serverID, time.Now().Unix())
	if err != nil {
		logError("heartbeat failed: %v", err)
	}
}

// GetStatus 获取服务器状态
func (h *Heartbeat) GetStatus(serverID string) (status string, lastSeen time.Time, err error) {
	var ts int64
	err = h.db.QueryRow(`
		SELECT status, last_seen FROM servers WHERE server_id = ?`,
		serverID).Scan(&status, &ts)
	if err != nil {
		return "unknown", time.Time{}, err
	}
	return status, time.Unix(ts, 0), nil
}

// CheckOffline 检查离线服务器
func (h *Heartbeat) CheckOffline(thresholdMin int) ([]string, error) {
	if thresholdMin <= 0 {
		thresholdMin = 3
	}

	threshold := time.Now().Add(-time.Duration(thresholdMin) * time.Minute).Unix()

	rows, err := h.db.Query(`
		SELECT server_id FROM servers 
		WHERE last_seen < ? AND status != 'offline'`,
		threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var offline []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		offline = append(offline, id)
	}

	// 标记为离线
	for _, id := range offline {
		h.db.Exec("UPDATE servers SET status = 'offline' WHERE server_id = ?", id)
	}

	return offline, nil
}

func logError(format string, v ...interface{}) {
	// 简单的日志实现
	fmt.Printf("[ERROR] "+format+"\n", v...)
}
