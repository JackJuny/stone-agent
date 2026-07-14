package system

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/user/stone/plugins"
)

// Checker 系统检测插件
type Checker struct{}

// New 创建系统检测器
func New() *Checker {
	return &Checker{}
}

// Name 插件名称
func (c *Checker) Name() string {
	return "system"
}

// Check 检测系统状态
func (c *Checker) Check() plugins.Status {
	return plugins.Status{OK: true, Message: "system ok"}
}

// Report 生成系统报告
func (c *Checker) Report() string {
	hostname, _ := os.Hostname()
	kernel := getKernel()
	osInfo := getOSInfo()

	return fmt.Sprintf(`System:
  Hostname: %s
  OS: %s
  Kernel: %s`, hostname, osInfo, kernel)
}

func getKernel() string {
	var uts syscall.Utsname
	if err := syscall.Uname(&uts); err != nil {
		return "unknown"
	}
	var buf [65]byte
	n := 0
	for i, b := range uts.Release {
		if b == 0 {
			break
		}
		buf[i] = byte(b)
		n = i + 1
	}
	return string(buf[:n])
}

func getOSInfo() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return runtime.GOOS
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			name := strings.TrimPrefix(line, "PRETTY_NAME=")
			return strings.Trim(name, "\"")
		}
	}
	return runtime.GOOS
}
