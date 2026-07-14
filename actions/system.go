package actions

import (
	"fmt"
	"os"
	"os/exec"
)

// RebootAction 重启服务器
type RebootAction struct{}

// NewRebootAction 创建重启服务器Action
func NewRebootAction() *RebootAction {
	return &RebootAction{}
}

// Name Action名称
func (a *RebootAction) Name() string {
	return "reboot"
}

// Execute 执行重启
func (a *RebootAction) Execute(params map[string]string) error {
	cmd := exec.Command("systemctl", "reboot")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("重启失败: %w", err)
	}

	// 等待重启命令生效
	fmt.Fprintln(os.Stderr, "服务器重启中...")
	return nil
}

// NeedConfirm 是否需要确认
func (a *RebootAction) NeedConfirm() bool {
	return true
}

// RestartAgentAction 重启Stone Agent
type RestartAgentAction struct{}

// NewRestartAgentAction 创建重启Agent Action
func NewRestartAgentAction() *RestartAgentAction {
	return &RestartAgentAction{}
}

// Name Action名称
func (a *RestartAgentAction) Name() string {
	return "restart_agent"
}

// Execute 执行重启Agent
func (a *RestartAgentAction) Execute(params map[string]string) error {
	cmd := exec.Command("systemctl", "restart", "stone")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("重启Agent失败: %w", err)
	}
	return nil
}

// NeedConfirm 是否需要确认
func (a *RestartAgentAction) NeedConfirm() bool {
	return true
}
