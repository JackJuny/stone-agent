package monitor

import "syscall"

// GetDiskUsage 获取磁盘使用情况
func GetDiskUsage(path string) (used float64, total float64, percent float64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0, err
	}

	total = float64(stat.Blocks) * float64(stat.Bsize)
	free := float64(stat.Bavail) * float64(stat.Bsize)
	used = total - free

	if total > 0 {
		percent = used / total * 100
	}

	return used, total, percent, nil
}
