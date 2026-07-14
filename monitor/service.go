package monitor

import (
	"fmt"
	"os/exec"
	"strings"
)

// SystemdChecker systemd服务检测器
type SystemdChecker struct{}

// NewSystemdChecker 创建systemd检测器
func NewSystemdChecker() *SystemdChecker {
	return &SystemdChecker{}
}

// Check 检测服务状态
func (c *SystemdChecker) Check(name string) ServiceStatus {
	cmd := exec.Command("systemctl", "is-active", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ServiceStatus{
			Name:   name,
			Active: false,
			Status: "stopped",
		}
	}

	status := strings.TrimSpace(string(output))
	active := status == "active"
	displayStatus := status
	if active {
		displayStatus = "running"
	}

	return ServiceStatus{
		Name:   name,
		Active: active,
		Status: displayStatus,
	}
}

// Status 获取服务状态字符串
func (c *SystemdChecker) Status(name string) string {
	return c.Check(name).Status
}

// Report 生成服务报告
func (c *SystemdChecker) Report(services []string) string {
	var results []string
	for _, svc := range services {
		status := c.Check(svc)
		results = append(results, fmt.Sprintf("• %s: %s", svc, status.Status))
	}
	return strings.Join(results, "\n")
}

// IsRunning 检查服务是否运行
func (c *SystemdChecker) IsRunning(name string) bool {
	return c.Check(name).Active
}
