package singbox

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/user/stone/plugins"
)

// Checker sing-box检测插件
type Checker struct {
	Ports []int
}

// New 创建sing-box检测器
func New(ports []int) *Checker {
	return &Checker{Ports: ports}
}

// Name 插件名称
func (c *Checker) Name() string {
	return "singbox"
}

// Check 检测sing-box状态
func (c *Checker) Check() plugins.Status {
	cmd := exec.Command("systemctl", "is-active", "sing-box")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return plugins.Status{OK: false, Message: "sing-box not running"}
	}

	status := strings.TrimSpace(string(output))
	if status == "active" {
		return plugins.Status{OK: true, Message: "sing-box running"}
	}
	return plugins.Status{OK: false, Message: fmt.Sprintf("sing-box status: %s", status)}
}

// Report 生成sing-box报告
func (c *Checker) Report() string {
	// 检查服务状态
	cmd := exec.Command("systemctl", "is-active", "sing-box")
	output, err := cmd.CombinedOutput()
	status := "stopped"
	if err == nil {
		s := strings.TrimSpace(string(output))
		if s == "active" {
			status = "running"
		} else {
			status = s
		}
	}

	// 检查端口
	var portResults []string
	for _, port := range c.Ports {
		result := checkPort(port)
		portResults = append(portResults, fmt.Sprintf("  %d %s", port, result))
	}

	return fmt.Sprintf(`sing-box:
  Status: %s
  Ports:
%s`, status, strings.Join(portResults, "\n"))
}

func checkPort(port int) string {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return "FAIL"
	}
	conn.Close()
	return "OK"
}
