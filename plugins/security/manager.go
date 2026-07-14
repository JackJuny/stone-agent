package security

import "fmt"

// Manager 安全管理器
type Manager struct {
	ufw      *UFWChecker
	fail2ban *Fail2banChecker
	enabled  bool
}

// NewManager 创建安全管理器
func NewManager(enabled bool) *Manager {
	return &Manager{
		ufw:      NewUFWChecker(),
		fail2ban: NewFail2banChecker(),
		enabled:  enabled,
	}
}

// IsEnabled 检查是否启用
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// GetStatus 获取安全状态
func (m *Manager) GetStatus() SecurityStatus {
	status := SecurityStatus{}

	if m.enabled {
		status.UFW = m.ufw.Check()
		status.Fail2ban = m.fail2ban.Check()
		status.TotalFailed = m.fail2ban.GetFailedAttempts()
	}

	return status
}

// GetUFWStatus 获取UFW状态
func (m *Manager) GetUFWStatus() UFWStatus {
	if !m.enabled {
		return UFWStatus{}
	}
	return m.ufw.Check()
}

// GetFail2banStatus 获取Fail2ban状态
func (m *Manager) GetFail2banStatus() Fail2banStatus {
	if !m.enabled {
		return Fail2banStatus{}
	}
	return m.fail2ban.Check()
}

// Name 插件名称
func (m *Manager) Name() string {
	return "security"
}

// Report 生成安全报告
func (m *Manager) Report() string {
	if !m.enabled {
		return "Security: disabled"
	}

	status := m.GetStatus()
	return formatSecurityStatus(&status)
}

func formatSecurityStatus(status *SecurityStatus) string {
	ufwEmoji := "🔴"
	if status.UFW.Active {
		ufwEmoji = "🟢"
	}

	fail2banEmoji := "🔴"
	if status.Fail2ban.Running {
		fail2banEmoji = "🟢"
	}

	result := "🛡 Security\n\n"
	result += "UFW " + ufwEmoji + "\n"
	result += "Fail2ban " + fail2banEmoji + "\n"

	if status.Fail2ban.Running && len(status.Fail2ban.Jails) > 0 {
		result += "\nJails:\n"
		for _, jail := range status.Fail2ban.Jails {
			result += "  " + jail + "\n"
		}
		result += "\nBan: " + fmt.Sprintf("%d", status.Fail2ban.BanCount) + "\n"
	}

	if status.TotalFailed > 0 {
		result += "\nSSH失败: " + fmt.Sprintf("%d", status.TotalFailed) + "\n"
	}

	return result
}
