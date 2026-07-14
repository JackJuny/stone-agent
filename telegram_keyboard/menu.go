package keyboard

import (
	"fmt"
	"strings"
)

// MenuType 菜单类型
type MenuType string

const (
	MenuMain       MenuType = "main"
	MenuServer     MenuType = "server"
	MenuServices   MenuType = "services"
	MenuService    MenuType = "service"
	MenuSystem     MenuType = "system"
	MenuSecurity   MenuType = "security"
	MenuLogs       MenuType = "logs"
	MenuConfirm    MenuType = "confirm"
)

// MenuData 菜单数据
type MenuData struct {
	Type        MenuType
	ServerID    string
	ServiceName string
	Action      string
	Message     string
}

// FormatCallbackData 格式化callback数据
func FormatCallbackData(menu MenuType, params ...string) string {
	data := string(menu)
	for _, p := range params {
		data += ":" + p
	}
	return data
}

// ParseCallbackData 解析callback数据
func ParseCallbackData(data string) MenuData {
	menu := MenuData{}

	parts := splitCallback(data)
	if len(parts) == 0 {
		return menu
	}

	menu.Type = MenuType(parts[0])
	if len(parts) > 1 {
		menu.ServerID = parts[1]
	}
	if len(parts) > 2 {
		menu.ServiceName = parts[2]
	}
	if len(parts) > 3 {
		menu.Action = parts[3]
	}

	return menu
}

func splitCallback(data string) []string {
	var parts []string
	current := ""
	for _, c := range data {
		if c == ':' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// ServerListItem 服务器列表项
type ServerListItem struct {
	ServerID string
	Name     string
	Flag     string
	Status   string
}

// GetFlag 获取地区旗帜
func GetFlag(location string) string {
	flags := map[string]string{
		"US": "🇺🇸", "US-LA": "🇺🇸", "US-NY": "🇺🇸",
		"HK": "🇭🇰", "HK-lite": "🇭🇰",
		"JP": "🇯🇵", "Tokyo": "🇯🇵",
		"SG": "🇸🇬", "Singapore": "🇸🇬",
		"TW": "🇹🇼", "Taiwan": "🇹🇼",
		"KR": "🇰🇷", "Korea": "🇰🇷",
		"DE": "🇩🇪", "Germany": "🇩🇪",
		"UK": "🇬🇧", "London": "🇬🇧",
	}

	for key, flag := range flags {
		if strings.Contains(strings.ToUpper(location), key) {
			return flag
		}
	}
	return "🖥"
}

// FormatServerList 格式化服务器列表
func FormatServerList(servers []ServerListItem) string {
	msg := "🪨 Stone Servers\n\n"
	for _, s := range servers {
		emoji := "🟢"
		if s.Status == "offline" {
			emoji = "🔴"
		}
		msg += fmt.Sprintf("%s %s\n%s Online\n\n", s.Flag, s.Name, emoji)
	}
	return msg
}

// FormatServerStatus 格式化服务器状态
func FormatServerStatus(name, status, cpu, ram, disk, network, version string) string {
	return fmt.Sprintf(`🪨 %s

%s

CPU: %s
RAM: %s
Disk: %s
Network: %s

Version: %s`,
		name, status, cpu, ram, disk, network, version)
}

// ServiceItem 服务项
type ServiceItem struct {
	Name   string
	Status string
}

// FormatServiceList 格式化服务列表
func FormatServiceList(services []ServiceItem) string {
	msg := "⚙️ 服务管理\n\n"
	for _, s := range services {
		emoji := "🟢"
		if s.Status == "stopped" {
			emoji = "🔴"
		} else if s.Status == "missing" {
			emoji = "⚪"
		}
		msg += fmt.Sprintf("%s %s\n", emoji, s.Name)
	}
	return msg
}

// FormatServiceDetail 格式化服务详情
func FormatServiceDetail(name, status string) string {
	emoji := "🟢"
	if status == "stopped" {
		emoji = "🔴"
	} else if status == "missing" {
		emoji = "⚪"
	}

	return fmt.Sprintf(`⚙️ %s

状态: %s %s`, name, emoji, status)
}

// FormatSecurity 格式化安全状态
func FormatSecurity(ufwActive, fail2banRunning bool, failedAttempts, banCount int) string {
	ufwEmoji := "🔴"
	if ufwActive {
		ufwEmoji = "🟢"
	}

	fail2banEmoji := "🔴"
	if fail2banRunning {
		fail2banEmoji = "🟢"
	}

	return fmt.Sprintf(`🛡 安全

UFW: %s
Fail2ban: %s

SSH失败: %d
Ban: %d`, ufwEmoji, fail2banEmoji, failedAttempts, banCount)
}

// LogItem 日志项
type LogItem struct {
	Time   string
	Action string
	Result string
}

// FormatLogs 格式化日志
func FormatLogs(logs []LogItem) string {
	if len(logs) == 0 {
		return "📜 暂无操作日志"
	}

	msg := "📜 最近操作\n\n"
	for _, l := range logs {
		msg += fmt.Sprintf("%s %s %s\n", l.Time, l.Action, l.Result)
	}
	return msg
}

// FormatConfirm 格式化确认消息
func FormatConfirm(serverName, target, action string) string {
	return fmt.Sprintf(`⚠️ 操作确认

服务器: %s
目标: %s
动作: %s`, serverName, target, action)
}
