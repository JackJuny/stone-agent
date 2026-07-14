package main

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/user/stone/actions"
	"github.com/user/stone/events"
	"github.com/user/stone/monitor"
	"github.com/user/stone/plugins"
	"github.com/user/stone/plugins/security"
	"github.com/user/stone/telegram_keyboard"
)

// Command 命令处理器
type Command struct {
	telegram       *Telegram
	config         *Config
	db             *DB
	checker        monitor.ServiceChecker
	pluginMgr      *plugins.Manager
	actionMgr      *actions.Manager
	confirmHandler *actions.ConfirmHandler
	securityMgr    *security.Manager
	eventMgr       *events.Manager
	lastCollect    time.Time
	lastCollectMu  sync.RWMutex
}

// NewCommand 创建命令处理器
func NewCommand(telegram *Telegram, config *Config, db *DB, checker monitor.ServiceChecker, pluginMgr *plugins.Manager, actionMgr *actions.Manager, confirmHandler *actions.ConfirmHandler, securityMgr *security.Manager, eventMgr *events.Manager) *Command {
	return &Command{
		telegram:       telegram,
		config:         config,
		db:             db,
		checker:        checker,
		pluginMgr:      pluginMgr,
		actionMgr:      actionMgr,
		confirmHandler: confirmHandler,
		securityMgr:    securityMgr,
		eventMgr:       eventMgr,
		lastCollect:    time.Now(),
	}
}

// UpdateLastCollect 更新最后采集时间
func (c *Command) UpdateLastCollect() {
	c.lastCollectMu.Lock()
	defer c.lastCollectMu.Unlock()
	c.lastCollect = time.Now()
}

// Handle 处理Telegram消息
func (c *Command) Handle(message *TelegramMessage) {
	if message.Text == "" {
		return
	}

	if !c.telegram.VerifyChatID(message.Chat.ID) {
		return
	}

	text := strings.TrimSpace(message.Text)
	args := strings.Fields(text)

	if len(args) == 0 {
		return
	}

	switch args[0] {
	case "/status":
		c.handleStatus()
	case "/report":
		c.handleReport()
	case "/services":
		c.handleServices()
	case "/plugins":
		c.handlePlugins()
	case "/actions":
		c.handleActions()
	case "/health":
		c.handleHealth()
	case "/security":
		c.handleSecurity()
	case "/servers":
		c.handleServers()
	case "/logs":
		c.handleLogs()
	case "/info":
		c.handleInfo()
	case "/hostname":
		c.handleHostname()
	case "/network":
		c.handleNetwork()
	case "/disk":
		c.handleDisk()
	case "/servers":
		c.handleServers()
	case "/service":
		if len(args) < 3 {
			c.telegram.SendMessage("用法: /service <服务名> <动作>\n动作: restart, start, stop, status")
			return
		}
		c.handleServiceControl(args[1], args[2])
	default:
		c.telegram.SendMessage("未知命令。\n\n可用命令:\n/status 状态\n/health 健康\n/security 安全\n/info 信息\n/network 网络\n/disk 磁盘\n/services 服务\n/plugins 插件\n/logs 日志\n/report 日报\n/servers 服务器列表\n/service <名> <动作>\n\n⚠️ /restart /reboot 已禁用\n请使用 /servers 选择服务器操作")
	}
}

func (c *Command) handleStatus() {
	status, err := monitor.Collect(c.config.GetNetworkIF())
	if err != nil {
		c.telegram.SendMessage(fmt.Sprintf("获取状态失败: %s", EscapeMarkdown(err.Error())))
		return
	}

	c.telegram.SendMessage(FormatStatus(c.config, status))
}

func (c *Command) handleReport() {
	status, err := monitor.Collect(c.config.GetNetworkIF())
	if err != nil {
		c.telegram.SendMessage(fmt.Sprintf("获取状态失败: %s", EscapeMarkdown(err.Error())))
		return
	}

	c.telegram.SendMessage(FormatReport(c.config, status, c.db))
}

func (c *Command) handleServices() {
	c.telegram.SendMessage(FormatServices(c.config))
}

func (c *Command) handlePlugins() {
	c.telegram.SendMessage(FormatPlugins(c.pluginMgr.Names()))
}

func (c *Command) handleActions() {
	var sysActions []string
	var svcActions []string

	if c.config.Actions.Allow.RestartAgent {
		sysActions = append(sysActions, "✓ restart agent")
	}
	if c.config.Actions.Allow.Reboot {
		sysActions = append(sysActions, "✓ reboot")
	}

	for svc, svcConfig := range c.config.Actions.Services {
		for _, action := range svcConfig.Allow {
			svcActions = append(svcActions, fmt.Sprintf("✓ %s %s", svc, action))
		}
	}

	msg := fmt.Sprintf(`🎯 可用操作

系统:
%s

服务:
%s`,
		strings.Join(sysActions, "\n"),
		strings.Join(svcActions, "\n"),
	)

	c.telegram.SendMessage(msg)
}

func (c *Command) handleHealth() {
	// 检查Agent状态
	agentOK := true

	// 检查Telegram连接
	telegramOK := c.telegram.IsConnected()

	// 检查数据库
	dbOK := true
	db, err := OpenDB()
	if err != nil {
		dbOK = false
	} else {
		db.Close()
	}

	// 最近采集时间
	lastCollectStr := "未知"
	c.lastCollectMu.RLock()
	lastCollect := c.lastCollect
	c.lastCollectMu.RUnlock()
	if !lastCollect.IsZero() {
		elapsed := time.Since(lastCollect)
		if elapsed < time.Minute {
			lastCollectStr = fmt.Sprintf("%d秒前", int(elapsed.Seconds()))
		} else if elapsed < time.Hour {
			lastCollectStr = fmt.Sprintf("%d分钟前", int(elapsed.Minutes()))
		} else {
			lastCollectStr = fmt.Sprintf("%d小时前", int(elapsed.Hours()))
		}
	}

	// 内存占用
	memPercent := 0.0
	status, err := monitor.Collect(c.config.GetNetworkIF())
	if err == nil {
		memPercent = status.MemoryPercent
	}

	c.telegram.SendMessage(FormatHealth(agentOK, dbOK, telegramOK, lastCollectStr, memPercent))
}

func (c *Command) handleSecurity() {
	if c.securityMgr == nil || !c.securityMgr.IsEnabled() {
		c.telegram.SendMessage("安全模块未启用")
		return
	}

	status := c.securityMgr.GetStatus()
	msg := FormatSecurity(c.config, &status, c.db)
	c.telegram.SendMessage(msg)
}

func (c *Command) handleServers() {
	// 获取所有服务器
	servers := []keyboard.ServerListItem{
		{ServerID: c.config.Server.ServerID, Name: c.config.Server.Name, Flag: keyboard.GetFlag(c.config.Server.Location), Status: "online"},
	}

	// 创建键盘
	keyboardMsg := &InlineKeyboard{
		InlineKeyboard: [][]InlineKeyboardButton{},
	}

	for _, s := range servers {
		emoji := "🟢"
		if s.Status == "offline" {
			emoji = "🔴"
		}
		text := fmt.Sprintf("%s %s %s", s.Flag, s.Name, emoji)
		callbackData := keyboard.FormatCallbackData(keyboard.MenuServer, s.ServerID)
		keyboardMsg.InlineKeyboard = append(keyboardMsg.InlineKeyboard, []InlineKeyboardButton{
			{Text: text, CallbackData: callbackData},
		})
	}

	msg := keyboard.FormatServerList(servers)
	c.telegram.SendMessageWithKeyboard(msg, keyboardMsg)
}

func (c *Command) handleLogs() {
	logs, err := c.db.GetRecentActionLogs(5)
	if err != nil {
		c.telegram.SendMessage(fmt.Sprintf("获取日志失败: %s", EscapeMarkdown(err.Error())))
		return
	}

	if len(logs) == 0 {
		c.telegram.SendMessage("📋 暂无操作日志")
		return
	}

	var results []string
	for _, log := range logs {
		t := time.Unix(log.Timestamp, 0)
		results = append(results, fmt.Sprintf(`时间: %s
用户: %d
操作: %s %s
结果: %s`,
			t.Format("2006-01-02 15:04"),
			log.UserID,
			log.Action,
			log.Target,
			log.Result,
		))
	}

	msg := fmt.Sprintf(`📋 Action Logs

%s`,
		strings.Join(results, "\n\n"),
	)

	c.telegram.SendMessage(msg)
}

func (c *Command) handleInfo() {
	status, err := monitor.Collect(c.config.GetNetworkIF())
	if err != nil {
		c.telegram.SendMessage(fmt.Sprintf("获取系统信息失败: %s", EscapeMarkdown(err.Error())))
		return
	}

	c.telegram.SendMessage(FormatInfo(c.config, status))
}

func (c *Command) handleHostname() {
	msg := fmt.Sprintf(`🖥 %s

IP %s
OS %s

Stone %s`,
		monitor.GetHostname(),
		monitor.GetLocalIP(),
		monitor.GetOSInfo(),
		Version,
	)

	c.telegram.SendMessage(truncate(msg))
}

func (c *Command) handleNetwork() {
	status, err := monitor.Collect(c.config.GetNetworkIF())
	if err != nil {
		c.telegram.SendMessage(fmt.Sprintf("获取网络状态失败: %s", EscapeMarkdown(err.Error())))
		return
	}

	c.telegram.SendMessage(FormatNetwork(c.config, status, c.db))
}

func (c *Command) handleDisk() {
	status, err := monitor.Collect(c.config.GetNetworkIF())
	if err != nil {
		c.telegram.SendMessage(fmt.Sprintf("获取磁盘状态失败: %s", EscapeMarkdown(err.Error())))
		return
	}

	c.telegram.SendMessage(FormatDisk(status))
}

func (c *Command) handleRestartAgent() {
	if !c.config.Actions.Enabled {
		c.telegram.SendMessage("操作未启用")
		return
	}

	if !c.config.Actions.Allow.RestartAgent {
		c.telegram.SendMessage("重启Agent操作未授权")
		return
	}

	// 检查冷却
	if cooling, remaining := c.actionMgr.CheckCooldown("restart_agent"); cooling {
		c.telegram.SendMessage(fmt.Sprintf("⚠️ 操作冷却中\n\n剩余: %d秒", remaining))
		return
	}

	keyboard := &InlineKeyboard{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "确认执行", CallbackData: "confirm_restart"},
				{Text: "取消", CallbackData: "cancel_restart"},
			},
		},
	}

	msg := fmt.Sprintf(`⚠️ 确认重启 Stone Agent

操作: restart agent

%d秒内有效`,
		c.config.Actions.ConfirmTimeout,
	)
	c.telegram.SendMessageWithKeyboard(msg, keyboard)
}

func (c *Command) handleReboot() {
	if !c.config.Actions.Enabled {
		c.telegram.SendMessage("操作未启用")
		return
	}

	if !c.config.Actions.Allow.Reboot {
		c.telegram.SendMessage("重启服务器操作未授权")
		return
	}

	// 检查冷却
	if cooling, remaining := c.actionMgr.CheckCooldown("reboot"); cooling {
		c.telegram.SendMessage(fmt.Sprintf("⚠️ 操作冷却中\n\n剩余: %d秒", remaining))
		return
	}

	keyboard := &InlineKeyboard{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "确认执行", CallbackData: "confirm_reboot"},
				{Text: "取消", CallbackData: "cancel_reboot"},
			},
		},
	}

	msg := fmt.Sprintf(`⚠️ 即将重启服务器

主机: %s

操作: reboot

%d秒内有效`,
		EscapeMarkdown(c.config.Server.Name),
		c.config.Actions.ConfirmTimeout,
	)
	c.telegram.SendMessageWithKeyboard(msg, keyboard)
}

func (c *Command) handleServiceControl(service, action string) {
	if !c.config.Actions.Enabled {
		c.telegram.SendMessage("操作未启用")
		return
	}

	// 验证服务名
	if !isValidServiceName(service) {
		c.telegram.SendMessage("无效的服务名")
		return
	}

	// 验证动作类型
	validActions := map[string]bool{"restart": true, "start": true, "stop": true, "status": true}
	if !validActions[action] {
		c.telegram.SendMessage("无效的操作类型，支持: restart, start, stop, status")
		return
	}

	if !c.config.IsServiceActionAllowed(service, action) {
		c.telegram.SendMessage(fmt.Sprintf("❌ 服务操作未授权: %s %s", service, action))
		c.db.InsertActionLog(0, "", fmt.Sprintf("service_%s", action), service, "", "", "denied", "权限不足")
		return
	}

	// 检查冷却
	cooldownKey := fmt.Sprintf("service_%s", action)
	if cooling, remaining := c.actionMgr.CheckCooldown(cooldownKey); cooling {
		c.telegram.SendMessage(fmt.Sprintf("⚠️ 操作冷却中\n\n剩余: %d秒", remaining))
		return
	}

	// 创建服务操作Action
	svcAction := actions.NewServiceAction(service, action, true)
	c.actionMgr.Register(svcAction)

	// 执行操作（带详细结果）
	result := svcAction.ExecuteWithResult()

	// 记录执行
	c.actionMgr.RecordExecution(cooldownKey)

	// 发送结果
	c.telegram.SendMessage(actions.FormatServiceResult(result))

	// 记录日志
	resultStr := "success"
	errMsg := ""
	if !result.Success {
		resultStr = "failed"
		errMsg = result.Error
	}
	c.db.InsertActionLog(0, "", fmt.Sprintf("service_%s", action), service, result.Before, result.After, resultStr, errMsg)
}

func (c *Command) HandleCallbackQuery(query *CallbackQuery) {
	if query == nil || query.From == nil {
		return
	}

	data := query.Data
	userID := query.From.ID

	// 检查用户权限
	if !c.config.IsUserAllowed(userID) {
		c.telegram.AnswerCallbackQuery(query.ID, "权限不足")
		return
	}

	switch data {
	case "confirm_restart":
		c.executeConfirmRestart(query)
	case "cancel_restart":
		c.cancelConfirm(query, "restart_agent")
	case "confirm_reboot":
		c.executeConfirmReboot(query)
	case "cancel_reboot":
		c.cancelConfirm(query, "reboot")
	default:
		c.telegram.AnswerCallbackQuery(query.ID, "Unknown action")
	}
}

func (c *Command) executeConfirmRestart(query *CallbackQuery) {
	if query.Message == nil {
		c.telegram.AnswerCallbackQuery(query.ID, "消息不存在")
		return
	}

	action := actions.NewRestartAgentAction()
	c.actionMgr.Register(action)

	c.actionMgr.RecordExecution("restart_agent")

	if err := action.Execute(nil); err != nil {
		c.telegram.AnswerCallbackQuery(query.ID, "执行失败")
		c.telegram.EditMessage(query.Message.Chat.ID, int64(query.Message.MessageID), "❌ 重启Agent失败")
		c.db.InsertActionLog(query.From.ID, query.From.Username, "restart_agent", "stone", "", "", "failed", err.Error())
		return
	}

	c.telegram.AnswerCallbackQuery(query.ID, "执行成功")
	c.telegram.EditMessage(query.Message.Chat.ID, int64(query.Message.MessageID), "✅ 正在重启 Stone Agent...")
	c.db.InsertActionLog(query.From.ID, query.From.Username, "restart_agent", "stone", "", "", "success", "")
}

func (c *Command) executeConfirmReboot(query *CallbackQuery) {
	if query.Message == nil {
		c.telegram.AnswerCallbackQuery(query.ID, "消息不存在")
		return
	}

	action := actions.NewRebootAction()
	c.actionMgr.Register(action)

	c.actionMgr.RecordExecution("reboot")

	if err := action.Execute(nil); err != nil {
		c.telegram.AnswerCallbackQuery(query.ID, "执行失败")
		c.telegram.EditMessage(query.Message.Chat.ID, int64(query.Message.MessageID), "❌ 重启服务器失败")
		c.db.InsertActionLog(query.From.ID, query.From.Username, "reboot", c.config.Server.Name, "", "", "failed", err.Error())
		return
	}

	c.telegram.AnswerCallbackQuery(query.ID, "执行成功")
	c.telegram.EditMessage(query.Message.Chat.ID, int64(query.Message.MessageID), "🔄 正在重启服务器...")
	c.db.InsertActionLog(query.From.ID, query.From.Username, "reboot", c.config.Server.Name, "", "", "success", "")
}

func (c *Command) cancelConfirm(query *CallbackQuery, actionName string) {
	if query.Message == nil {
		c.telegram.AnswerCallbackQuery(query.ID, "已取消")
		return
	}
	c.actionMgr.Cancel(actionName, query.From.ID)
	c.telegram.AnswerCallbackQuery(query.ID, "已取消")
	c.telegram.EditMessage(query.Message.Chat.ID, int64(query.Message.MessageID), "❌ 操作已取消")
}

func isValidServiceName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}

func runCommand(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Run()
}
