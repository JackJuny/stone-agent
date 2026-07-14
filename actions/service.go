package actions

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"
)

// ServiceAction 服务操作
type ServiceAction struct {
	ServiceName string
	ActionType  string // restart, start, stop, status
	Allowed     bool
}

// ServiceResult 服务操作结果
type ServiceResult struct {
	Service    string
	Action     string
	Before     string
	After      string
	Duration   time.Duration
	Success    bool
	Error      string
	StartTime  time.Time
	EndTime    time.Time
}

// ValidateServiceName 校验服务名（白名单：字母数字._-）
func ValidateServiceName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

// ValidateActionType 校验操作类型
func ValidateActionType(action string) bool {
	switch action {
	case "restart", "start", "stop", "status":
		return true
	default:
		return false
	}
}

// NewServiceAction 创建服务操作Action
func NewServiceAction(serviceName, actionType string, allowed bool) *ServiceAction {
	return &ServiceAction{
		ServiceName: serviceName,
		ActionType:  actionType,
		Allowed:     allowed,
	}
}

// Name Action名称
func (a *ServiceAction) Name() string {
	return fmt.Sprintf("service_%s_%s", a.ServiceName, a.ActionType)
}

// Execute 执行服务操作（带完整反馈）
func (a *ServiceAction) Execute(params map[string]string) error {
	if !a.Allowed {
		return fmt.Errorf("服务操作未授权: %s %s", a.ServiceName, a.ActionType)
	}

	var cmd *exec.Cmd
	switch a.ActionType {
	case "restart":
		cmd = exec.Command("systemctl", "restart", a.ServiceName)
	case "start":
		cmd = exec.Command("systemctl", "start", a.ServiceName)
	case "stop":
		cmd = exec.Command("systemctl", "stop", a.ServiceName)
	case "status":
		cmd = exec.Command("systemctl", "is-active", a.ServiceName)
	default:
		return fmt.Errorf("未知操作类型: %s", a.ActionType)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("服务 %s 失败: %s", a.ActionType, strings.TrimSpace(string(output)))
	}

	return nil
}

// ExecuteWithResult 执行服务操作（带详细结果）
func (a *ServiceAction) ExecuteWithResult() *ServiceResult {
	result := &ServiceResult{
		Service:   a.ServiceName,
		Action:    a.ActionType,
		StartTime: time.Now(),
	}

	// 权限检查
	if !a.Allowed {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = false
		result.Error = fmt.Sprintf("服务操作未授权: %s %s", a.ServiceName, a.ActionType)
		return result
	}

	// 获取执行前状态
	result.Before = GetServiceStatus(a.ServiceName)

	// 执行操作
	var cmd *exec.Cmd
	switch a.ActionType {
	case "restart":
		cmd = exec.Command("systemctl", "restart", a.ServiceName)
	case "start":
		cmd = exec.Command("systemctl", "start", a.ServiceName)
	case "stop":
		cmd = exec.Command("systemctl", "stop", a.ServiceName)
	case "status":
		cmd = exec.Command("systemctl", "is-active", a.ServiceName)
	default:
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = false
		result.Error = fmt.Sprintf("未知操作类型: %s", a.ActionType)
		return result
	}

	output, err := cmd.CombinedOutput()
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if err != nil {
		result.Success = false
		result.Error = strings.TrimSpace(string(output))
	} else {
		result.Success = true
	}

	// 获取执行后状态
	result.After = GetServiceStatus(a.ServiceName)

	return result
}

// NeedConfirm 是否需要确认
func (a *ServiceAction) NeedConfirm() bool {
	return a.ActionType == "restart" || a.ActionType == "stop"
}

// GetServiceStatus 获取服务状态
func GetServiceStatus(serviceName string) string {
	cmd := exec.Command("systemctl", "is-active", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "stopped"
	}

	status := strings.TrimSpace(string(output))
	if status == "active" {
		return "running"
	}
	return status
}

// FormatServiceResult 格式化服务操作结果
func FormatServiceResult(result *ServiceResult) string {
	beforeEmoji := "🔴"
	if result.Before == "running" {
		beforeEmoji = "🟢"
	}

	afterEmoji := "🔴"
	if result.After == "running" {
		afterEmoji = "🟢"
	}

	statusText := "成功"
	if !result.Success {
		statusText = "失败"
	}

	return fmt.Sprintf(`⚙️ 服务操作

目标: %s

动作: %s

执行前: %s %s

执行后: %s %s

耗时: %.1fs

结果: %s`,
		result.Service,
		result.Action,
		beforeEmoji, result.Before,
		afterEmoji, result.After,
		result.Duration.Seconds(),
		statusText,
	)
}
