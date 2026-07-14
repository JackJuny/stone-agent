package monitor

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// GetCPUUsage 获取CPU使用率
func GetCPUUsage() (float64, error) {
	initial, err := readCPUStat()
	if err != nil {
		return 0, err
	}

	time.Sleep(500 * time.Millisecond)

	final, err := readCPUStat()
	if err != nil {
		return 0, err
	}

	idle := final.idle - initial.idle
	total := final.total - initial.total

	if total == 0 {
		return 0, nil
	}

	return float64(total-idle) / float64(total) * 100, nil
}

type cpuStat struct {
	idle  uint64
	total uint64
}

func readCPUStat() (cpuStat, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuStat{}, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return cpuStat{}, fmt.Errorf("invalid cpu stat line")
			}

			var total uint64
			var values []uint64
			for _, f := range fields[1:] {
				v, err := strconv.ParseUint(f, 10, 64)
				if err != nil {
					return cpuStat{}, err
				}
				values = append(values, v)
				total += v
			}

			idle := values[3]
			return cpuStat{idle: idle, total: total}, nil
		}
	}

	return cpuStat{}, fmt.Errorf("cpu stat not found")
}

// GetCPUModel 获取CPU型号
func GetCPUModel() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "unknown"
}
