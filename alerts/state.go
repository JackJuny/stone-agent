package alerts

import (
	"database/sql"
	"fmt"
	"time"
)

// StateManager 状态管理器
type StateManager struct {
	db *sql.DB
}

// NewStateManager 创建状态管理器
func NewStateManager(db *sql.DB) *StateManager {
	return &StateManager{db: db}
}

// GetState 获取当前状态
func (m *StateManager) GetState(server, category, name string) (State, error) {
	var state State
	err := m.db.QueryRow(`
		SELECT current_state FROM alert_states 
		WHERE server = ? AND category = ? AND name = ?`,
		server, category, name).Scan(&state)
	if err == sql.ErrNoRows {
		return StateUnknown, nil
	}
	return state, err
}

// SetState 设置状态
func (m *StateManager) SetState(server, category, name string, newState State) (changed bool, oldState State, err error) {
	oldState, err = m.GetState(server, category, name)
	if err != nil {
		return false, "", err
	}

	if oldState == newState {
		return false, oldState, nil
	}

	// 更新或插入状态
	_, err = m.db.Exec(`
		INSERT INTO alert_states (server, category, name, current_state, previous_state, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(server, category, name) DO UPDATE SET
			previous_state = current_state,
			current_state = excluded.current_state,
			updated_at = excluded.updated_at`,
		server, category, name, newState, oldState, time.Now().Unix())
	if err != nil {
		return false, "", fmt.Errorf("set state: %w", err)
	}

	return true, oldState, nil
}

// GetLastNotify 获取最后通知时间
func (m *StateManager) GetLastNotify(server, category, name string) (time.Time, error) {
	var ts int64
	err := m.db.QueryRow(`
		SELECT last_notify FROM alert_states 
		WHERE server = ? AND category = ? AND name = ?`,
		server, category, name).Scan(&ts)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(ts, 0), nil
}

// UpdateLastNotify 更新最后通知时间
func (m *StateManager) UpdateLastNotify(server, category, name string) error {
	_, err := m.db.Exec(`
		UPDATE alert_states SET last_notify = ?
		WHERE server = ? AND category = ? AND name = ?`,
		time.Now().Unix(), server, category, name)
	return err
}

// GetAllStates 获取所有状态
func (m *StateManager) GetAllStates() ([]Alert, error) {
	rows, err := m.db.Query(`
		SELECT server, category, name, current_state, previous_state, updated_at, last_notify
		FROM alert_states ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.Server, &a.Category, &a.Name, &a.CurrentState, &a.PreviousState, &a.UpdatedAt, &a.LastNotify); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, nil
}

// GetStatesByServer 获取指定服务器的状态
func (m *StateManager) GetStatesByServer(server string) ([]Alert, error) {
	rows, err := m.db.Query(`
		SELECT server, category, name, current_state, previous_state, updated_at, last_notify
		FROM alert_states WHERE server = ? ORDER BY updated_at DESC`, server)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.Server, &a.Category, &a.Name, &a.CurrentState, &a.PreviousState, &a.UpdatedAt, &a.LastNotify); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, nil
}

// GetActiveAlerts 获取活跃告警（状态异常的）
func (m *StateManager) GetActiveAlerts() ([]Alert, error) {
	rows, err := m.db.Query(`
		SELECT server, category, name, current_state, previous_state, updated_at, last_notify
		FROM alert_states 
		WHERE current_state != 'running' AND current_state != 'unknown'
		ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.Server, &a.Category, &a.Name, &a.CurrentState, &a.PreviousState, &a.UpdatedAt, &a.LastNotify); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, nil
}
