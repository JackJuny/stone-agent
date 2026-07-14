package keyboard

import (
	"fmt"
	"strings"
)

// CallbackHandler 回调处理器
type CallbackHandler struct {
	serverMgr   ServerManager
	actionMgr   ActionManager
	telegram    TelegramSender
	config      ConfigProvider
}

// ServerManager 服务器管理器接口
type ServerManager interface {
	GetAll() ([]ServerListItem, error)
	GetByID(id string) (*ServerInfo, error)
}

// ServerInfo 服务器信息
type ServerInfo struct {
	ServerID string
	Name     string
	Location string
	Status   string
	Version  string
}

// ActionManager 操作管理器接口
type ActionManager interface {
	Execute(name string, params map[string]string, userID int64) error
}

// TelegramSender Telegram发送接口
type TelegramSender interface {
	EditMessage(chatID int64, messageID int64, text string) error
	EditMessageWithKeyboard(chatID int64, messageID int64, text string, keyboard *Keyboard) error
	AnswerCallback(callbackID, text string) error
}

// ConfigProvider 配置提供接口
type ConfigProvider interface {
	GetServerName() string
	GetServerID() string
	GetServices() []string
}

// NewCallbackHandler 创建回调处理器
func NewCallbackHandler(serverMgr ServerManager, actionMgr ActionManager, telegram TelegramSender, config ConfigProvider) *CallbackHandler {
	return &CallbackHandler{
		serverMgr: serverMgr,
		actionMgr: actionMgr,
		telegram:  telegram,
		config:    config,
	}
}

// Handle 处理回调
func (h *CallbackHandler) Handle(chatID int64, messageID int64, callbackID, data string, userID int64) {
	menu := ParseCallbackData(data)

	switch menu.Type {
	case MenuMain:
		h.handleMain(chatID, messageID, callbackID)
	case MenuServer:
		h.handleServer(chatID, messageID, callbackID, menu.ServerID)
	case MenuServices:
		h.handleServices(chatID, messageID, callbackID, menu.ServerID)
	case MenuService:
		h.handleService(chatID, messageID, callbackID, menu.ServerID, menu.ServiceName)
	case MenuSystem:
		h.handleSystem(chatID, messageID, callbackID, menu.ServerID)
	case MenuSecurity:
		h.handleSecurity(chatID, messageID, callbackID, menu.ServerID)
	case MenuLogs:
		h.handleLogs(chatID, messageID, callbackID, menu.ServerID)
	case MenuConfirm:
		h.handleConfirm(chatID, messageID, callbackID, menu.ServerID, menu.ServiceName, menu.Action, userID)
	default:
		if strings.HasPrefix(data, "service_") {
			h.handleServiceAction(chatID, messageID, callbackID, data, userID)
		} else if strings.HasPrefix(data, "confirm_") {
			h.handleConfirmAction(chatID, messageID, callbackID, data, userID)
		} else if data == "cancel" {
			h.handleCancel(chatID, messageID, callbackID)
		}
	}
}

func (h *CallbackHandler) handleMain(chatID, messageID int64, callbackID string) {
	h.telegram.AnswerCallback(callbackID, "")

	servers, _ := h.serverMgr.GetAll()
	keyboard := NewKeyboard()

	for _, s := range servers {
		emoji := "🟢"
		if s.Status == "offline" {
			emoji = "🔴"
		}
		text := fmt.Sprintf("%s %s %s", s.Flag, s.Name, emoji)
		callbackData := FormatCallbackData(MenuServer, s.ServerID)
		keyboard.AddButton(text, callbackData)
	}

	text := FormatServerList(servers)
	h.telegram.EditMessageWithKeyboard(chatID, messageID, text, keyboard)
}

func (h *CallbackHandler) handleServer(chatID, messageID int64, callbackID, serverID string) {
	h.telegram.AnswerCallback(callbackID, "")

	server, err := h.serverMgr.GetByID(serverID)
	if err != nil {
		h.telegram.EditMessage(chatID, messageID, "服务器不存在")
		return
	}

	// 获取服务器状态
	status := "🟢 Online"
	cpu := "N/A"
	ram := "N/A"
	disk := "N/A"
	network := "N/A"

	text := FormatServerStatus(server.Name, status, cpu, ram, disk, network, server.Version)

	keyboard := NewKeyboard()
	keyboard.AddRow(
		Button{Text: "📊 刷新", CallbackData: FormatCallbackData(MenuServer, serverID)},
		Button{Text: "⚙️ 服务", CallbackData: FormatCallbackData(MenuServices, serverID)},
	)
	keyboard.AddRow(
		Button{Text: "🛡 安全", CallbackData: FormatCallbackData(MenuSecurity, serverID)},
		Button{Text: "📜 日志", CallbackData: FormatCallbackData(MenuLogs, serverID)},
	)
	keyboard.AddRow(
		Button{Text: "🔄 系统", CallbackData: FormatCallbackData(MenuSystem, serverID)},
	)
	keyboard.AddRow(BackButton(string(MenuMain)))

	h.telegram.EditMessageWithKeyboard(chatID, messageID, text, keyboard)
}

func (h *CallbackHandler) handleServices(chatID, messageID int64, callbackID, serverID string) {
	h.telegram.AnswerCallback(callbackID, "")

	services := h.config.GetServices()
	keyboard := NewKeyboard()

	for _, svc := range services {
		callbackData := FormatCallbackData(MenuService, serverID, svc)
		keyboard.AddButton(svc, callbackData)
	}

	keyboard.AddRow(BackButton(FormatCallbackData(MenuServer, serverID)))

	text := FormatServiceList([]ServiceItem{})
	h.telegram.EditMessageWithKeyboard(chatID, messageID, text, keyboard)
}

func (h *CallbackHandler) handleService(chatID, messageID int64, callbackID, serverID, serviceName string) {
	h.telegram.AnswerCallback(callbackID, "")

	status := "running"
	text := FormatServiceDetail(serviceName, status)

	keyboard := NewKeyboard()
	keyboard.AddRow(
		Button{Text: "▶️ 启动", CallbackData: fmt.Sprintf("service_%s_%s_start", serverID, serviceName)},
		Button{Text: "⏹ 停止", CallbackData: fmt.Sprintf("service_%s_%s_stop", serverID, serviceName)},
	)
	keyboard.AddRow(
		Button{Text: "🔄 重启", CallbackData: fmt.Sprintf("service_%s_%s_restart", serverID, serviceName)},
	)
	keyboard.AddRow(BackButton(FormatCallbackData(MenuServices, serverID)))

	h.telegram.EditMessageWithKeyboard(chatID, messageID, text, keyboard)
}

func (h *CallbackHandler) handleSystem(chatID, messageID int64, callbackID, serverID string) {
	h.telegram.AnswerCallback(callbackID, "")

	text := "🔄 系统操作"

	keyboard := NewKeyboard()
	keyboard.AddButton("🔄 重启Stone Agent", fmt.Sprintf("confirm_%s_stone_restart", serverID))
	keyboard.AddButton("🔄 重启服务器", fmt.Sprintf("confirm_%s_server_reboot", serverID))
	keyboard.AddRow(BackButton(FormatCallbackData(MenuServer, serverID)))

	h.telegram.EditMessageWithKeyboard(chatID, messageID, text, keyboard)
}

func (h *CallbackHandler) handleSecurity(chatID, messageID int64, callbackID, serverID string) {
	h.telegram.AnswerCallback(callbackID, "")

	text := FormatSecurity(true, true, 0, 0)

	keyboard := NewKeyboard()
	keyboard.AddRow(BackButton(FormatCallbackData(MenuServer, serverID)))

	h.telegram.EditMessageWithKeyboard(chatID, messageID, text, keyboard)
}

func (h *CallbackHandler) handleLogs(chatID, messageID int64, callbackID, serverID string) {
	h.telegram.AnswerCallback(callbackID, "")

	logs := []LogItem{}
	text := FormatLogs(logs)

	keyboard := NewKeyboard()
	keyboard.AddRow(BackButton(FormatCallbackData(MenuServer, serverID)))

	h.telegram.EditMessageWithKeyboard(chatID, messageID, text, keyboard)
}

func (h *CallbackHandler) handleConfirm(chatID, messageID int64, callbackID, serverID, target, action string, userID int64) {
	h.telegram.AnswerCallback(callbackID, "")

	serverName := h.config.GetServerName()
	text := FormatConfirm(serverName, target, action)

	keyboard := NewKeyboard()
	keyboard.AddRow(
		ConfirmButton(fmt.Sprintf("%s_%s_%s", serverID, target, action)),
		CancelButton(),
	)

	h.telegram.EditMessageWithKeyboard(chatID, messageID, text, keyboard)
}

func (h *CallbackHandler) handleServiceAction(chatID, messageID int64, callbackID, data string, userID int64) {
	h.telegram.AnswerCallback(callbackID, "")

	// 解析: service_serverID_serviceName_action
	parts := strings.Split(data, "_")
	if len(parts) < 4 {
		return
	}

	serverID := parts[1]
	serviceName := parts[2]
	action := parts[3]

	// 检查是否是当前服务器
	if serverID != h.config.GetServerID() {
		h.telegram.EditMessage(chatID, messageID, "此操作需要在对应服务器上执行")
		return
	}

	// 执行操作
	params := map[string]string{"service": serviceName, "action": action}
	err := h.actionMgr.Execute(fmt.Sprintf("service_%s_%s", serviceName, action), params, userID)

	if err != nil {
		h.telegram.EditMessage(chatID, messageID, fmt.Sprintf("操作失败: %s", err))
		return
	}

	h.telegram.EditMessage(chatID, messageID, fmt.Sprintf("✅ %s %s 已执行", serviceName, action))
}

func (h *CallbackHandler) handleConfirmAction(chatID, messageID int64, callbackID, data string, userID int64) {
	h.telegram.AnswerCallback(callbackID, "")

	// 解析: confirm_serverID_target_action
	parts := strings.Split(data, "_")
	if len(parts) < 4 {
		return
	}

	serverID := parts[1]
	target := parts[2]
	action := parts[3]

	if serverID != h.config.GetServerID() {
		h.telegram.EditMessage(chatID, messageID, "此操作需要在对应服务器上执行")
		return
	}

	var actionName string
	var params map[string]string

	if target == "stone" && action == "restart" {
		actionName = "restart_agent"
		params = nil
	} else if target == "server" && action == "reboot" {
		actionName = "reboot"
		params = nil
	} else {
		h.telegram.EditMessage(chatID, messageID, "未知操作")
		return
	}

	err := h.actionMgr.Execute(actionName, params, userID)
	if err != nil {
		h.telegram.EditMessage(chatID, messageID, fmt.Sprintf("操作失败: %s", err))
		return
	}

	h.telegram.EditMessage(chatID, messageID, fmt.Sprintf("✅ %s 已执行", actionName))
}

func (h *CallbackHandler) handleCancel(chatID, messageID int64, callbackID string) {
	h.telegram.AnswerCallback(callbackID, "已取消")
	h.telegram.EditMessage(chatID, messageID, "❌ 操作已取消")
}
