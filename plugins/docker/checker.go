package docker

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/user/stone/plugins"
)

// Checker Docker检测插件
type Checker struct {
	Containers []string
}

// New 创建Docker检测器
func New(containers []string) *Checker {
	return &Checker{Containers: containers}
}

// Name 插件名称
func (c *Checker) Name() string {
	return "docker"
}

// Check 检测Docker状态
func (c *Checker) Check() plugins.Status {
	cmd := exec.Command("systemctl", "is-active", "docker")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return plugins.Status{OK: false, Message: "docker not running"}
	}

	status := strings.TrimSpace(string(output))
	if status == "active" {
		return plugins.Status{OK: true, Message: "docker running"}
	}
	return plugins.Status{OK: false, Message: fmt.Sprintf("docker status: %s", status)}
}

// Report 生成Docker报告
func (c *Checker) Report() string {
	// 检查docker服务
	cmd := exec.Command("systemctl", "is-active", "docker")
	output, err := cmd.CombinedOutput()
	dockerStatus := "stopped"
	if err == nil {
		s := strings.TrimSpace(string(output))
		if s == "active" {
			dockerStatus = "running"
		} else {
			dockerStatus = s
		}
	}

	// 获取运行中的容器
	running := c.getRunningContainers()

	// 生成容器报告
	var containerResults []string
	for _, name := range c.Containers {
		status := "stopped"
		if _, ok := running[name]; ok {
			status = "running"
		}
		emoji := "🔴"
		if status == "running" {
			emoji = "🟢"
		}
		containerResults = append(containerResults, fmt.Sprintf("  %s %s", emoji, name))
	}

	return fmt.Sprintf(`Docker:
  Service: %s
  Containers:
%s`, dockerStatus, strings.Join(containerResults, "\n"))
}

func (c *Checker) getRunningContainers() map[string]bool {
	running := make(map[string]bool)

	cmd := exec.Command("docker", "ps", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return running
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name != "" {
			running[name] = true
		}
	}

	return running
}
