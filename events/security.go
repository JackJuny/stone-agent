package events

import (
	"database/sql"
	"fmt"
	"time"
)

// SecurityHandler 安全事件处理器
type SecurityHandler struct {
	db       *sql.DB
	sendMsg  func(string) error
	serverName string
}

// NewSecurityHandler 创建安全事件处理器
func NewSecurityHandler(db *sql.DB, sendMsg func(string) error, serverName string) *SecurityHandler {
	return &SecurityHandler{
		db:         db,
		sendMsg:    sendMsg,
		serverName: serverName,
	}
}

// Handle 处理安全事件
func (h *SecurityHandler) Handle(event Event) error {
	// 写入数据库
	if err := h.insertEvent(event); err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	// 发送Telegram通知
	msg := h.formatMessage(event)
	if err := h.sendMsg(msg); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

func (h *SecurityHandler) insertEvent(event Event) error {
	_, err := h.db.Exec(`
		INSERT INTO security_events (timestamp, type, source, ip, jail, action, details)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		time.Now().Unix(),
		event.Type,
		event.Source,
		event.IP,
		event.Source, // jail字段使用source值
		event.Action,
		event.Details,
	)
	return err
}

func (h *SecurityHandler) formatMessage(event Event) string {
	switch event.Action {
	case "ban":
		return fmt.Sprintf(`🚨 SSH攻击拦截

服务器: %s

攻击IP: %s
规则: %s
动作: 已封禁

时间 %s`,
			h.serverName,
			event.IP,
			event.Source,
			time.Now().Format("15:04"),
		)
	case "unban":
		return fmt.Sprintf(`🛡 SSH封禁解除

服务器: %s

IP: %s
规则: %s

时间 %s`,
			h.serverName,
			event.IP,
			event.Source,
			time.Now().Format("15:04"),
		)
	default:
		return fmt.Sprintf(`⚠️ 安全事件

服务器: %s

类型: %s
IP: %s
动作: %s

时间 %s`,
			h.serverName,
			event.Type,
			event.IP,
			event.Action,
			time.Now().Format("15:04"),
		)
	}
}
