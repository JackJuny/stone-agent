package actions

import (
	"fmt"
	"sync"
	"time"
)

// Action 远程操作接口
type Action interface {
	Name() string
	Execute(params map[string]string) error
	NeedConfirm() bool
}

// PendingConfirm 待确认操作
type PendingConfirm struct {
	Action    Action
	Params    map[string]string
	UserID    int64
	MessageID int
	CreatedAt time.Time
}

// CooldownEntry 冷却记录
type CooldownEntry struct {
	LastExec time.Time
}

// Manager Action管理器
type Manager struct {
	actions   map[string]Action
	pending   map[string]*PendingConfirm
	cooldowns map[string]*CooldownEntry
	cooldownsConfig map[string]int // action name -> cooldown seconds
	mu        sync.RWMutex
}

// NewManager 创建Action管理器
func NewManager() *Manager {
	return &Manager{
		actions:   make(map[string]Action),
		pending:   make(map[string]*PendingConfirm),
		cooldowns: make(map[string]*CooldownEntry),
		cooldownsConfig: make(map[string]int),
	}
}

// SetCooldown 设置冷却时间
func (m *Manager) SetCooldown(actionName string, seconds int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cooldownsConfig[actionName] = seconds
}

// Register 注册Action
func (m *Manager) Register(action Action) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.actions[action.Name()] = action
}

// Get 获取Action
func (m *Manager) Get(name string) Action {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.actions[name]
}

// GetAll 获取所有Action
func (m *Manager) GetAll() []Action {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Action
	for _, a := range m.actions {
		result = append(result, a)
	}
	return result
}

// Names 获取所有Action名称
func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	for name := range m.actions {
		names = append(names, name)
	}
	return names
}

// CheckCooldown 检查冷却
func (m *Manager) CheckCooldown(actionName string) (bool, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cooldownSec, ok := m.cooldownsConfig[actionName]
	if !ok || cooldownSec <= 0 {
		return false, 0
	}

	entry, ok := m.cooldowns[actionName]
	if !ok {
		return false, 0
	}

	elapsed := int(time.Since(entry.LastExec).Seconds())
	if elapsed < cooldownSec {
		return true, cooldownSec - elapsed
	}

	return false, 0
}

// RecordExecution 记录执行时间
func (m *Manager) RecordExecution(actionName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cooldowns[actionName] = &CooldownEntry{LastExec: time.Now()}
}

// Execute 执行Action（带冷却检查，原子操作）
func (m *Manager) Execute(name string, params map[string]string, userID int64) error {
	action := m.Get(name)
	if action == nil {
		return fmt.Errorf("操作不存在: %s", name)
	}

	// 原子检查冷却 + 记录执行
	m.mu.Lock()
	if entry, ok := m.cooldowns[name]; ok {
		elapsed := time.Since(entry.LastExec)
		if elapsed.Seconds() < float64(m.cooldownsConfig[name]) {
			remaining := m.cooldownsConfig[name] - int(elapsed.Seconds())
			m.mu.Unlock()
			return fmt.Errorf("操作冷却中，剩余 %d 秒", remaining)
		}
	}
	// 记录执行时间（在检查通过后立即记录，防止并发绕过）
	m.cooldowns[name] = &CooldownEntry{LastExec: time.Now()}
	m.mu.Unlock()

	if action.NeedConfirm() {
		return m.RequestConfirm(action, params, userID)
	}

	return action.Execute(params)
}

// ExecuteWithoutCooldown 无冷却执行
func (m *Manager) ExecuteWithoutCooldown(name string, params map[string]string) error {
	action := m.Get(name)
	if action == nil {
		return fmt.Errorf("操作不存在: %s", name)
	}

	m.RecordExecution(name)
	return action.Execute(params)
}

// RequestConfirm 请求确认
func (m *Manager) RequestConfirm(action Action, params map[string]string, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s_%d", action.Name(), userID)
	m.pending[key] = &PendingConfirm{
		Action:    action,
		Params:    params,
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	return nil
}

// RequestConfirmWithMessage 请求确认（保存消息ID）
func (m *Manager) RequestConfirmWithMessage(action Action, params map[string]string, userID int64, messageID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s_%d", action.Name(), userID)
	m.pending[key] = &PendingConfirm{
		Action:    action,
		Params:    params,
		UserID:    userID,
		MessageID: messageID,
		CreatedAt: time.Now(),
	}
	return nil
}

// Confirm 确认执行
func (m *Manager) Confirm(actionName string, userID int64, timeout time.Duration) (Action, map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s_%d", actionName, userID)
	pending, ok := m.pending[key]
	if !ok {
		return nil, nil, fmt.Errorf("没有待确认的操作: %s", actionName)
	}

	if time.Since(pending.CreatedAt) > timeout {
		delete(m.pending, key)
		return nil, nil, fmt.Errorf("确认已过期")
	}

	delete(m.pending, key)
	return pending.Action, pending.Params, nil
}

// GetPending 获取待确认操作
func (m *Manager) GetPending(actionName string, userID int64) *PendingConfirm {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s_%d", actionName, userID)
	return m.pending[key]
}

// Cancel 取消确认
func (m *Manager) Cancel(actionName string, userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s_%d", actionName, userID)
	delete(m.pending, key)
}

// HasPending 检查是否有待确认操作
func (m *Manager) HasPending(actionName string, userID int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s_%d", actionName, userID)
	_, ok := m.pending[key]
	return ok
}

// CleanExpired 清理过期的确认请求
func (m *Manager) CleanExpired(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, pending := range m.pending {
		if now.Sub(pending.CreatedAt) > timeout {
			delete(m.pending, key)
		}
	}
}
