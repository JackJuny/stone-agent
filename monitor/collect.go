package monitor

// Collect 收集所有系统状态
func Collect(networkIF string) (*SystemStatus, error) {
	s := &SystemStatus{}

	cpu, err := GetCPUUsage()
	if err != nil {
		return nil, err
	}
	s.CPU = cpu

	memUsed, memTotal, memPercent, err := GetMemoryUsage()
	if err != nil {
		return nil, err
	}
	s.MemoryUsed = memUsed
	s.MemoryTotal = memTotal
	s.MemoryPercent = memPercent

	diskUsed, diskTotal, diskPercent, err := GetDiskUsage("/")
	if err != nil {
		return nil, err
	}
	s.DiskUsed = diskUsed
	s.DiskTotal = diskTotal
	s.DiskPercent = diskPercent

	s.Uptime = GetUptime()
	s.UptimeSec = GetUptimeSec()
	s.Load = GetLoad()
	s.Hostname = GetHostname()
	s.OSInfo = GetOSInfo()
	s.Kernel = GetKernelVersion()
	s.CPUModel = GetCPUModel()
	s.LocalIP = GetLocalIP()

	if networkIF != "" {
		rx, tx, err := GetNetworkStats(networkIF)
		if err == nil {
			s.NetRX = rx
			s.NetTX = tx
		}
	}

	return s, nil
}
