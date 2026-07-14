package monitor

import (
	"os"
	"strconv"
	"strings"
)

// GetMemoryUsage 获取内存使用情况
func GetMemoryUsage() (used float64, total float64, percent float64, err error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, err
	}

	memInfo := make(map[string]uint64)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		valStr = strings.TrimSuffix(valStr, " kB")
		val, err := strconv.ParseUint(valStr, 10, 64)
		if err != nil {
			continue
		}
		memInfo[key] = val * 1024
	}

	total = float64(memInfo["MemTotal"])
	memFree := float64(memInfo["MemFree"])
	buffers := float64(memInfo["Buffers"])
	cached := float64(memInfo["Cached"])

	used = total - memFree - buffers - cached
	if total > 0 {
		percent = used / total * 100
	}

	return used, total, percent, nil
}

// GetMemTotalBytes 获取总内存字节数
func GetMemTotalBytes() uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, _ := strconv.ParseUint(parts[1], 10, 64)
				return val * 1024
			}
		}
	}
	return 0
}
