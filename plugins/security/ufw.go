package security

import (
	"os/exec"
	"strings"
)

// UFWChecker UFW检测器
type UFWChecker struct{}

// NewUFWChecker 创建UFW检测器
func NewUFWChecker() *UFWChecker {
	return &UFWChecker{}
}

// Check 检测UFW状态
func (c *UFWChecker) Check() UFWStatus {
	status := UFWStatus{}

	cmd := exec.Command("ufw", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return status
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Status:") {
			if strings.Contains(line, "active") {
				status.Active = true
			}
		} else if strings.HasPrefix(line, "[") || strings.Contains(line, "ALLOW") || strings.Contains(line, "DENY") {
			status.Rules = append(status.Rules, line)
			status.Count++
		}
	}

	return status
}

// Name 插件名称
func (c *UFWChecker) Name() string {
	return "ufw"
}

// Report 生成UFW报告
func (c *UFWChecker) Report() string {
	status := c.Check()
	emoji := "🔴"
	if status.Active {
		emoji = "🟢"
	}

	return "UFW " + emoji
}
