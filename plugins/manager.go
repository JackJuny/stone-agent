package plugins

import (
	"fmt"
	"strings"
)

// Plugin 插件接口
type Plugin interface {
	Name() string
	Check() Status
	Report() string
}

// Status 插件状态
type Status struct {
	OK      bool
	Message string
}

// Manager 插件管理器
type Manager struct {
	plugins []Plugin
}

// NewManager 创建插件管理器
func NewManager() *Manager {
	return &Manager{}
}

// Register 注册插件
func (m *Manager) Register(p Plugin) {
	m.plugins = append(m.plugins, p)
}

// GetAll 获取所有插件
func (m *Manager) GetAll() []Plugin {
	return m.plugins
}

// GetByName 按名称获取插件
func (m *Manager) GetByName(name string) Plugin {
	for _, p := range m.plugins {
		if p.Name() == name {
			return p
		}
	}
	return nil
}

// Names 获取所有插件名称
func (m *Manager) Names() []string {
	var names []string
	for _, p := range m.plugins {
		names = append(names, p.Name())
	}
	return names
}

// ReportAll 生成所有插件报告
func (m *Manager) ReportAll() string {
	var reports []string
	for _, p := range m.plugins {
		reports = append(reports, p.Report())
	}
	return strings.Join(reports, "\n\n")
}

// ReportPlugin 生成指定插件报告
func (m *Manager) ReportPlugin(name string) string {
	p := m.GetByName(name)
	if p == nil {
		return fmt.Sprintf("插件不存在: %s", name)
	}
	return p.Report()
}

// StatusAll 获取所有插件状态
func (m *Manager) StatusAll() []Status {
	var statuses []Status
	for _, p := range m.plugins {
		statuses = append(statuses, p.Check())
	}
	return statuses
}
