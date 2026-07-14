package alerts

// State 告警状态
type State string

const (
	StateUnknown  State = "unknown"
	StateRunning  State = "running"
	StateStopped  State = "stopped"
	StateMissing  State = "missing"
	StateError    State = "error"
)

// Category 告警类别
type Category string

const (
	CategoryService Category = "service"
	CategorySystem  Category = "system"
	CategorySecurity Category = "security"
	CategoryMetric  Category = "metric"
)

// Alert 告警记录
type Alert struct {
	ID           int64
	Server       string
	Category     Category
	Name         string
	CurrentState State
	PreviousState State
	Message      string
	UpdatedAt    int64
	LastNotify   int64
}

// AlertLog 告警日志
type AlertLog struct {
	ID        int64
	Timestamp int64
	Type      string
	Target    string
	OldState  string
	NewState  string
	Message   string
}
