package server

import (
	"database/sql"
	"time"
)

// ServerInfo 服务器信息
type ServerInfo struct {
	ID         int64
	ServerID   string
	Name       string
	Location   string
	Role       string
	Version    string
	LastSeen   time.Time
	Status     string
}

// Registry 服务器注册表
type Registry struct {
	db *sql.DB
}

// NewRegistry 创建注册表
func NewRegistry(db *sql.DB) *Registry {
	return &Registry{db: db}
}

// Register 注册服务器
func (r *Registry) Register(serverID, name, location, role, version string) error {
	_, err := r.db.Exec(`
		INSERT INTO servers (server_id, name, location, role, version, last_seen, status)
		VALUES (?, ?, ?, ?, ?, ?, 'online')
		ON CONFLICT(server_id) DO UPDATE SET
			name = excluded.name,
			location = excluded.location,
			role = excluded.role,
			version = excluded.version,
			last_seen = excluded.last_seen,
			status = 'online'`,
		serverID, name, location, role, version, time.Now().Unix())
	return err
}

// GetAll 获取所有服务器
func (r *Registry) GetAll() ([]ServerInfo, error) {
	rows, err := r.db.Query(`
		SELECT id, server_id, name, location, role, version, last_seen, status
		FROM servers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []ServerInfo
	for rows.Next() {
		var s ServerInfo
		var lastSeen int64
		if err := rows.Scan(&s.ID, &s.ServerID, &s.Name, &s.Location, &s.Role, &s.Version, &lastSeen, &s.Status); err != nil {
			continue
		}
		s.LastSeen = time.Unix(lastSeen, 0)
		servers = append(servers, s)
	}
	return servers, nil
}

// GetByServerID 获取指定服务器
func (r *Registry) GetByServerID(serverID string) (*ServerInfo, error) {
	var s ServerInfo
	var lastSeen int64
	err := r.db.QueryRow(`
		SELECT id, server_id, name, location, role, version, last_seen, status
		FROM servers WHERE server_id = ?`, serverID).Scan(
		&s.ID, &s.ServerID, &s.Name, &s.Location, &s.Role, &s.Version, &lastSeen, &s.Status)
	if err != nil {
		return nil, err
	}
	s.LastSeen = time.Unix(lastSeen, 0)
	return &s, nil
}

// UpdateStatus 更新服务器状态
func (r *Registry) UpdateStatus(serverID, status string) error {
	_, err := r.db.Exec(`
		UPDATE servers SET status = ?, last_seen = ?
		WHERE server_id = ?`,
		status, time.Now().Unix(), serverID)
	return err
}

// UpdateVersion 更新版本
func (r *Registry) UpdateVersion(serverID, version string) error {
	_, err := r.db.Exec(`
		UPDATE servers SET version = ?
		WHERE server_id = ?`,
		version, serverID)
	return err
}

// GetOnlineCount 获取在线服务器数量
func (r *Registry) GetOnlineCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM servers WHERE status = 'online'").Scan(&count)
	return count, err
}

// GetOfflineServers 获取离线服务器
func (r *Registry) GetOfflineServers(thresholdMin int) ([]ServerInfo, error) {
	if thresholdMin <= 0 {
		thresholdMin = 3
	}

	threshold := time.Now().Add(-time.Duration(thresholdMin) * time.Minute).Unix()

	rows, err := r.db.Query(`
		SELECT server_id, name, last_seen FROM servers
		WHERE last_seen < ? AND status != 'offline'`,
		threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []ServerInfo
	for rows.Next() {
		var s ServerInfo
		var lastSeen int64
		if err := rows.Scan(&s.ServerID, &s.Name, &lastSeen); err != nil {
			continue
		}
		s.LastSeen = time.Unix(lastSeen, 0)
		servers = append(servers, s)
	}
	return servers, nil
}
