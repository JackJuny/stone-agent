package monitor

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// FormatBytes 格式化字节数
func FormatBytes(bytes float64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TB", bytes/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", bytes/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", bytes/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", bytes/KB)
	default:
		return fmt.Sprintf("%.0f B", bytes)
	}
}

// FormatBytesUint64 无符号整数版本
func FormatBytesUint64(bytes uint64) string {
	return FormatBytes(float64(bytes))
}

// GetUptime 获取系统运行时间
func GetUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "unknown"
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return "unknown"
	}

	uptimeSec, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "unknown"
	}

	days := int(uptimeSec) / 86400
	hours := (int(uptimeSec) % 86400) / 3600
	minutes := (int(uptimeSec) % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%d days", days)
	}
	if hours > 0 {
		return fmt.Sprintf("%d hours %d minutes", hours, minutes)
	}
	return fmt.Sprintf("%d minutes", minutes)
}

// GetUptimeSec 获取系统运行秒数
func GetUptimeSec() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0
	}
	sec, _ := strconv.ParseFloat(fields[0], 64)
	return int64(sec)
}

// GetLoad 获取系统负载
func GetLoad() string {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return "unknown"
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return "unknown"
	}

	return fmt.Sprintf("%s %s %s", fields[0], fields[1], fields[2])
}

// GetLoad1 获取1分钟负载
func GetLoad1() float64 {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0
	}
	var v float64
	fmt.Sscanf(fields[0], "%f", &v)
	return v
}

// GetOSInfo 获取操作系统信息
func GetOSInfo() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return runtime.GOOS
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			name := strings.TrimPrefix(line, "PRETTY_NAME=")
			name = strings.Trim(name, "\"")
			return name
		}
	}

	return runtime.GOOS
}

// GetKernelVersion 获取内核版本
func GetKernelVersion() string {
	var uts syscall.Utsname
	if err := syscall.Uname(&uts); err != nil {
		return "unknown"
	}
	// 转换 []int8 到 string
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

// GetHostname 获取主机名
func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
