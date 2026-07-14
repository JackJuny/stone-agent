package actions

import (
	"fmt"
	"time"
)

// ConfirmHandler 确认处理器
type ConfirmHandler struct {
	manager *Manager
	timeout time.Duration
}

// NewConfirmHandler 创建确认处理器
func NewConfirmHandler(manager *Manager, timeoutSec int) *ConfirmHandler {
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	return &ConfirmHandler{
		manager: manager,
		timeout: time.Duration(timeoutSec) * time.Second,
	}
}

// HandleConfirm 处理确认请求
func (h *ConfirmHandler) HandleConfirm(actionName string, userID int64) (string, error) {
	action, params, err := h.manager.Confirm(actionName, userID, h.timeout)
	if err != nil {
		return "", err
	}

	if err := action.Execute(params); err != nil {
		return "", err
	}

	return fmt.Sprintf("✅ %s 执行成功", actionName), nil
}

// GetTimeout 获取超时时间
func (h *ConfirmHandler) GetTimeout() time.Duration {
	return h.timeout
}

// CleanExpired 清理过期确认
func (h *ConfirmHandler) CleanExpired() {
	h.manager.CleanExpired(h.timeout)
}
