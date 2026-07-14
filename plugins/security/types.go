package security

// Event 安全事件
type Event struct {
	ID        int64
	Timestamp int64
	Type      string // fail2ban, ufw, ssh
	Source    string // sshd, nginx, etc
	IP        string
	Jail      string
	Action    string // ban, unban, alert
	Details   string
}

// UFWStatus UFW状态
type UFWStatus struct {
	Active   bool
	Rules    []string
	Count    int
}

// Fail2banStatus Fail2ban状态
type Fail2banStatus struct {
	Running   bool
	Jails     []string
	BanCount  int
	FailedCount int
}

// SecurityStatus 安全状态汇总
type SecurityStatus struct {
	UFW       UFWStatus
	Fail2ban  Fail2banStatus
	Events    []Event
	TotalBans int
	TotalFailed int
}
