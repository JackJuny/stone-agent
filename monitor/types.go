package monitor

// SystemStatus 系统状态
type SystemStatus struct {
	CPU           float64
	MemoryUsed    float64
	MemoryTotal   float64
	MemoryPercent float64
	DiskUsed      float64
	DiskTotal     float64
	DiskPercent   float64
	Uptime        string
	UptimeSec     int64
	Load          string
	NetRX         uint64
	NetTX         uint64
	Hostname      string
	OSInfo        string
	Kernel        string
	CPUModel      string
	LocalIP       string
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name   string
	Active bool
	Status string
}

// ServiceChecker 服务检测接口（未来可扩展）
type ServiceChecker interface {
	// Check 检测服务状态
	Check(name string) ServiceStatus
	// Status 获取服务状态字符串
	Status(name string) string
	// IsRunning 检查服务是否运行
	IsRunning(name string) bool
	// Report 生成服务报告
	Report(services []string) string
}
