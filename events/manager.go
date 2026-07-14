package events

import (
	"fmt"
	"time"
)

// Event 事件
type Event struct {
	ID        int64
	Type      string // security
	Action    string // ban, unban
	IP        string
	Source    string // sshd, nginx
	Details   string
	Timestamp time.Time
}

// Handler 事件处理器接口
type Handler interface {
	Handle(event Event) error
}

// Manager 事件管理器
type Manager struct {
	handlers map[string]Handler
}

// NewManager 创建事件管理器
func NewManager() *Manager {
	return &Manager{
		handlers: make(map[string]Handler),
	}
}

// Register 注册事件处理器
func (m *Manager) Register(eventType string, handler Handler) {
	m.handlers[eventType] = handler
}

// Process 处理事件
func (m *Manager) Process(event Event) error {
	key := fmt.Sprintf("%s_%s", event.Type, event.Action)
	handler, ok := m.handlers[key]
	if !ok {
		// 尝试类型处理器
		handler, ok = m.handlers[event.Type]
		if !ok {
			return fmt.Errorf("no handler for event: %s", key)
		}
	}
	return handler.Handle(event)
}
