package security

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Fail2banChecker Fail2ban检测器
type Fail2banChecker struct{}

// NewFail2banChecker 创建Fail2ban检测器
func NewFail2banChecker() *Fail2banChecker {
	return &Fail2banChecker{}
}

// Check 检测Fail2ban状态
func (c *Fail2banChecker) Check() Fail2banStatus {
	status := Fail2banStatus{}

	// 检查服务状态
	cmd := exec.Command("systemctl", "is-active", "fail2ban")
	output, err := cmd.CombinedOutput()
	if err == nil && strings.TrimSpace(string(output)) == "active" {
		status.Running = true
	}

	if !status.Running {
		return status
	}

	// 获取jail列表
	cmd = exec.Command("fail2ban-client", "status")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return status
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-") {
			jail := strings.TrimPrefix(line, "- ")
			jail = strings.TrimSpace(jail)
			if jail != "" {
				status.Jails = append(status.Jails, jail)
			}
		}
	}

	// 获取每个jail的ban数量
	for _, jail := range status.Jails {
		bans := c.getJailBans(jail)
		status.BanCount += bans
	}

	return status
}

func (c *Fail2banChecker) getJailBans(jail string) int {
	cmd := exec.Command("fail2ban-client", "status", jail)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Currently banned") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				num, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				return num
			}
		}
	}
	return 0
}

// GetFailedAttempts 获取SSH失败次数（从日志）
func (c *Fail2banChecker) GetFailedAttempts() int {
	cmd := exec.Command("grep", "-c", "Failed password", "/var/log/auth.log")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0
	}

	count, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	return count
}

// Name 插件名称
func (c *Fail2banChecker) Name() string {
	return "fail2ban"
}

// Report 生成Fail2ban报告
func (c *Fail2banChecker) Report() string {
	status := c.Check()
	emoji := "🔴"
	if status.Running {
		emoji = "🟢"
	}

	return fmt.Sprintf("Fail2ban %s\nJails: %d\nBan: %d", emoji, len(status.Jails), status.BanCount)
}
