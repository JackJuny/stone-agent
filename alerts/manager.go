package alerts

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Manager 告警管理器
type Manager struct {
	stateMgr      *StateManager
	cooldownMgr   *CooldownManager
	sendMsg       func(string) error
	serverName    string
	enabled       bool
	notifyRecovery bool
}

// NewManager 创建告警管理器
func NewManager(db *sql.DB, sendMsg func(string) error, serverName string, enabled bool, notifyRecovery bool, cooldownMin int) *Manager {
	stateMgr := NewStateManager(db)
	cooldownMgr := NewCooldownManager(stateMgr, cooldownMin)

	return &Manager{
		stateMgr:      stateMgr,
		cooldownMgr:   cooldownMgr,
		sendMsg:       sendMsg,
		serverName:    serverName,
		enabled:       enabled,
		notifyRecovery: notifyRecovery,
	}
}

// CheckService 检查服务状态并发送告警
func (m *Manager) CheckService(name string, installed bool, running bool) {
	if !m.enabled {
		return
	}

	var newState State
	if !installed {
		newState = StateMissing
	} else if running {
		newState = StateRunning
	} else {
		newState = StateStopped
	}

	changed, oldState, err := m.stateMgr.SetState(m.serverName, string(CategoryService), name, newState)
	if err != nil {
		log.Printf("[ERROR] alert state: %v", err)
		return
	}

	if !changed {
		return
	}

	// 检查冷却
	if !m.cooldownMgr.CanNotify(m.serverName, string(CategoryService), name) {
		return
	}

	// 发送通知
	msg := m.formatServiceMessage(name, oldState, newState)
	if msg == "" {
		return
	}

	if err := m.sendMsg(msg); err != nil {
		log.Printf("[ERROR] send alert: %v", err)
		return
	}

	// 更新通知时间
	m.stateMgr.UpdateLastNotify(m.serverName, string(CategoryService), name)

	// 记录日志
	m.logAlert(string(CategoryService), name, string(oldState), string(newState), msg)
}

// CheckMetric 检查指标告警
func (m *Manager) CheckMetric(name string, value float64, threshold float64) {
	if !m.enabled {
		return
	}

	var newState State
	if value > threshold {
		newState = StateError
	} else {
		newState = StateRunning
	}

	changed, oldState, err := m.stateMgr.SetState(m.serverName, string(CategoryMetric), name, newState)
	if err != nil {
		log.Printf("[ERROR] alert state: %v", err)
		return
	}

	if !changed {
		return
	}

	if !m.cooldownMgr.CanNotify(m.serverName, string(CategoryMetric), name) {
		return
	}

	msg := m.formatMetricMessage(name, value, threshold, oldState, newState)
	if msg == "" {
		return
	}

	if err := m.sendMsg(msg); err != nil {
		log.Printf("[ERROR] send alert: %v", err)
		return
	}

	m.stateMgr.UpdateLastNotify(m.serverName, string(CategoryMetric), name)
	m.logAlert(string(CategoryMetric), name, string(oldState), string(newState), msg)
}

func (m *Manager) formatServiceMessage(name string, oldState, newState State) string {
	switch newState {
	case StateMissing:
		if oldState == StateUnknown {
			return fmt.Sprintf(`⚠️ 服务未安装

服务器: %s

服务: %s

时间 %s`,
				m.serverName, name, time.Now().Format("15:04"))
		}
		return fmt.Sprintf(`⚠️ 服务已移除

服务器: %s

服务: %s

时间 %s`,
			m.serverName, name, time.Now().Format("15:04"))
	case StateStopped:
		return fmt.Sprintf(`🚨 服务异常停止

服务器: %s

服务: %s

时间 %s`,
			m.serverName, name, time.Now().Format("15:04"))
	case StateRunning:
		if oldState == StateStopped {
			return fmt.Sprintf(`✅ 服务恢复

服务器: %s

服务: %s

时间 %s`,
				m.serverName, name, time.Now().Format("15:04"))
		}
		if oldState == StateMissing {
			return fmt.Sprintf(`ℹ️ 服务已安装

服务器: %s

服务: %s

时间 %s`,
				m.serverName, name, time.Now().Format("15:04"))
		}
		if m.notifyRecovery {
			return fmt.Sprintf(`✅ 服务启动

服务器: %s

服务: %s

时间 %s`,
				m.serverName, name, time.Now().Format("15:04"))
		}
	}
	return ""
}

func (m *Manager) formatMetricMessage(name string, value, threshold float64, oldState, newState State) string {
	if newState == StateError {
		return fmt.Sprintf(`🚨 %s异常

服务器: %s

当前: %.0f%%
阈值: %.0f%%

时间 %s`,
			name, m.serverName, value, threshold, time.Now().Format("15:04"))
	}
	if oldState == StateError && m.notifyRecovery {
		return fmt.Sprintf(`✅ %s恢复

服务器: %s

当前: %.0f%%

时间 %s`,
			name, m.serverName, value, time.Now().Format("15:04"))
	}
	return ""
}

func (m *Manager) logAlert(alertType, target, oldState, newState, message string) {
	db := m.stateMgr.db
	if db == nil {
		return
	}

	db.Exec(`
		INSERT INTO alert_logs (timestamp, type, target, old_state, new_state, message)
		VALUES (?, ?, ?, ?, ?, ?)`,
		time.Now().Unix(), alertType, target, oldState, newState, message)
}

// GetStateMgr 获取状态管理器
func (m *Manager) GetStateMgr() *StateManager {
	return m.stateMgr
}

// IsEnabled 检查是否启用
func (m *Manager) IsEnabled() bool {
	return m.enabled
}
