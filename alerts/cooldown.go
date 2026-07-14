package alerts

import (
	"time"
)

// CooldownManager 冷却管理器
type CooldownManager struct {
	stateMgr    *StateManager
	cooldownSec int
}

// NewCooldownManager 创建冷却管理器
func NewCooldownManager(stateMgr *StateManager, cooldownMin int) *CooldownManager {
	if cooldownMin <= 0 {
		cooldownMin = 30
	}
	return &CooldownManager{
		stateMgr:    stateMgr,
		cooldownSec: cooldownMin * 60,
	}
}

// CanNotify 检查是否可以通知
func (m *CooldownManager) CanNotify(server, category, name string) bool {
	lastNotify, err := m.stateMgr.GetLastNotify(server, category, name)
	if err != nil {
		return true
	}

	if lastNotify.IsZero() {
		return true
	}

	elapsed := time.Since(lastNotify)
	return elapsed.Seconds() >= float64(m.cooldownSec)
}
