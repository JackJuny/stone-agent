package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/user/stone/monitor"
	"github.com/user/stone/plugins/security"
)

// getServiceStatus 获取服务状态
func getServiceStatus(service string) string {
	cmd := exec.Command("systemctl", "is-active", service)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "stopped"
	}
	status := strings.TrimSpace(string(output))
	if status == "active" {
		return "running"
	}
	return status
}

// isServiceInstalled 检查服务是否安装
func isServiceInstalled(service string) bool {
	cmd := exec.Command("systemctl", "list-unit-files", service+".service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), service+".service")
}

const maxMsgLen = 3000

// truncate 截断过长消息
func truncate(s string) string {
	if len(s) <= maxMsgLen {
		return s
	}
	return s[:maxMsgLen-3] + "..."
}

// shortName 短名称（截断过长的CPU型号等）
func shortName(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// FormatReport 格式化日报
func FormatReport(config *Config, status *monitor.SystemStatus, db *DB) string {
	loc := config.Report.Location()
	now := time.Now().In(loc)

	todayRX, todayTX, todayErr := db.GetTodayTraffic()

	todayStr := ""
	if todayErr == nil {
		todayStr = fmt.Sprintf("↓%s ↑%s", monitor.FormatBytesUint64(todayRX), monitor.FormatBytesUint64(todayTX))
	} else {
		todayStr = "暂无数据"
	}

	// 服务状态
	var svcLines []string
	for _, svc := range config.Services {
		svcStatus := getServiceStatus(svc)
		emoji := "⚪"
		if svcStatus == "running" {
			emoji = "🟢"
		} else if svcStatus == "stopped" && isServiceInstalled(svc) {
			emoji = "🔴"
		}
		svcLines = append(svcLines, fmt.Sprintf("%s %s", svc, emoji))
	}
	svcStr := strings.Join(svcLines, " | ")

	msg := fmt.Sprintf(`🪨 %s日报

📅 %s

⏱ 运行 %s

📊 CPU %d%% | RAM %d%% | Load %s

💾 Disk %d%%

🌐 今日 %s

⚙️ %s

Stone %s`,
		config.Server.Name,
		now.Format("2006-01-02"),
		status.Uptime,
		int(status.CPU),
		int(status.MemoryPercent),
		status.Load,
		int(status.DiskPercent),
		todayStr,
		svcStr,
		Version,
	)

	return truncate(msg)
}

// FormatStatus 格式化状态消息
func FormatStatus(config *Config, status *monitor.SystemStatus) string {
	msg := fmt.Sprintf(`🪨 %s

🟢 Online
⏱ 运行 %s

📊 CPU %d%% | RAM %d%% | Load %s

💾 Disk %d%%

🌐 ↓%s ↑%s

Stone %s`,
		config.Server.Name,
		status.Uptime,
		int(status.CPU),
		int(status.MemoryPercent),
		status.Load,
		int(status.DiskPercent),
		monitor.FormatBytesUint64(status.NetRX),
		monitor.FormatBytesUint64(status.NetTX),
		Version,
	)

	return truncate(msg)
}

func formatServiceCompact(status *monitor.SystemStatus) string {
	// 实际服务检查在command.go的handleServices中
	return ""
}

// FormatInfo 格式化系统信息
func FormatInfo(config *Config, status *monitor.SystemStatus) string {
	msg := fmt.Sprintf(`🖥 %s

OS %s
Kernel %s
CPU %s
RAM %s
IP %s

Stone %s`,
		config.Server.Name,
		status.OSInfo,
		status.Kernel,
		shortName(status.CPUModel, 30),
		monitor.FormatBytes(float64(monitor.GetMemTotalBytes())),
		status.LocalIP,
		Version,
	)

	return truncate(msg)
}

// FormatNetwork 格式化网络状态
func FormatNetwork(config *Config, status *monitor.SystemStatus, db *DB) string {
	todayRX, todayTX, todayErr := db.GetTodayTraffic()
	periodRX, periodTX, periodErr := db.GetPeriodTraffic()

	todayStr := "暂无数据"
	periodStr := "暂无数据"
	if todayErr == nil {
		todayStr = fmt.Sprintf("↓%s ↑%s", monitor.FormatBytesUint64(todayRX), monitor.FormatBytesUint64(todayTX))
	}
	if periodErr == nil {
		periodStr = fmt.Sprintf("↓%s ↑%s", monitor.FormatBytesUint64(periodRX), monitor.FormatBytesUint64(periodTX))
	}

	msg := fmt.Sprintf(`🌐 网络

今日:
%s

累计:
%s`,
		todayStr,
		periodStr,
	)

	return truncate(msg)
}

// FormatDisk 格式化磁盘状态
func FormatDisk(status *monitor.SystemStatus) string {
	msg := fmt.Sprintf(`💾 磁盘

%s
已用 %s (%d%%)
剩余 %s`,
		monitor.FormatBytes(status.DiskTotal),
		monitor.FormatBytes(status.DiskUsed),
		int(status.DiskPercent),
		monitor.FormatBytes(status.DiskTotal-status.DiskUsed),
	)

	return truncate(msg)
}

// FormatServices 格式化服务状态
func FormatServices(config *Config) string {
	var lines []string
	for _, svc := range config.Services {
		status := getServiceStatus(svc)
		emoji := "⚪"
		if status == "running" {
			emoji = "🟢"
		} else if status == "stopped" && isServiceInstalled(svc) {
			emoji = "🔴"
		}
		lines = append(lines, fmt.Sprintf("%s %s", emoji, svc))
	}

	msg := fmt.Sprintf(`⚙️ 服务

%s`, strings.Join(lines, "\n"))

	return truncate(msg)
}

// FormatPlugins 格式化插件列表
func FormatPlugins(names []string) string {
	var lines []string
	for _, name := range names {
		lines = append(lines, fmt.Sprintf("✓ %s", name))
	}

	msg := fmt.Sprintf(`🔌 插件

%s`, strings.Join(lines, "\n"))

	return truncate(msg)
}

// FormatHealth 格式化健康检查
func FormatHealth(agentOK bool, dbOK bool, telegramOK bool, lastCollect string, memPercent float64) string {
	agentEmoji := "🔴"
	if agentOK {
		agentEmoji = "🟢"
	}
	dbEmoji := "🔴"
	if dbOK {
		dbEmoji = "🟢"
	}
	tgEmoji := "🔴"
	if telegramOK {
		tgEmoji = "🟢"
	}

	msg := fmt.Sprintf(`❤️ 健康

%s Agent | %s DB | %s Telegram

⏱ 最近采集: %s
🧠 RAM %.0f%%`,
		agentEmoji,
		dbEmoji,
		tgEmoji,
		lastCollect,
		memPercent,
	)

	return truncate(msg)
}

// FormatAlert 格式化异常告警
func FormatAlert(config *Config, alertType, detail string) string {
	msg := fmt.Sprintf(`🚨 %s异常

⚠️ %s

时间 %s`,
		config.Server.Name,
		detail,
		time.Now().Format("15:04"),
	)

	return truncate(msg)
}

// FormatServiceAlert 格式化服务异常告警
func FormatServiceAlert(config *Config, service, status string) string {
	msg := fmt.Sprintf(`🚨 %s异常

⚙️ %s %s

时间 %s`,
		config.Server.Name,
		service,
		status,
		time.Now().Format("15:04"),
	)

	return truncate(msg)
}

// FormatSecurity 格式化安全状态
func FormatSecurity(config *Config, status *security.SecurityStatus, db *DB) string {
	ufwEmoji := "🔴"
	if status.UFW.Active {
		ufwEmoji = "🟢"
	}

	fail2banEmoji := "🔴"
	if status.Fail2ban.Running {
		fail2banEmoji = "🟢"
	}

	// 获取最近事件
	events, _ := db.GetRecentSecurityEvents(5)
	var eventLines []string
	for _, event := range events {
		t := time.Unix(event.Timestamp, 0)
		eventLines = append(eventLines, fmt.Sprintf("%s %s %s %s", t.Format("15:04"), event.Source, event.Action, event.IP))
	}
	eventStr := "暂无"
	if len(eventLines) > 0 {
		eventStr = strings.Join(eventLines, "\n")
	}

	msg := fmt.Sprintf(`🛡 安全

UFW %s
Fail2ban %s

SSH失败 %d
Ban %d

最近事件:
%s`,
		ufwEmoji,
		fail2banEmoji,
		status.TotalFailed,
		status.Fail2ban.BanCount,
		eventStr,
	)

	return truncate(msg)
}
