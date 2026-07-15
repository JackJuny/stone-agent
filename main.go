package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/user/stone/actions"
	"github.com/user/stone/alerts"
	"github.com/user/stone/events"
	"github.com/user/stone/monitor"
	"github.com/user/stone/plugins"
	"github.com/user/stone/plugins/docker"
	"github.com/user/stone/plugins/security"
	"github.com/user/stone/plugins/singbox"
	"github.com/user/stone/plugins/system"
	"github.com/user/stone/server"
)

var (
	Version   = "v0.6.2"
	Commit    = "none"
	buildDate = "unknown"
)

const (
	configPath = "/etc/stone/config.yaml"
	logPath    = "/var/log/stone/stone.log"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("Stone Agent %s (commit: %s, built: %s)\n", Version, Commit, buildDate)
			return
		case "status":
			cmdStatus()
			return
		case "report":
			cmdReport()
			return
		case "check":
			cmdCheck()
			return
		case "health":
			cmdHealth()
			return
		case "db":
			cmdDB()
			return
		case "plugins":
			cmdPlugins()
			return
		case "services":
			cmdServices()
			return
		case "actions":
			cmdActions()
			return
		case "security":
			cmdSecurity()
			return
		case "event":
			cmdEvent()
			return
		case "alerts":
			cmdAlerts()
			return
		case "logs":
			cmdLogs()
			return
		case "uninstall":
			cmdUninstall()
			return
		}
	}

	runDaemon()
}

func cmdStatus() {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %s\n", err)
		os.Exit(1)
	}

	status, err := monitor.Collect(cfg.GetNetworkIF())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	fmt.Println(FormatStatus(cfg, status))
}

func cmdReport() {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Stone启动失败:\n%s\n", err)
		os.Exit(1)
	}

	db, err := OpenDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Database error: %s\n", err)
		os.Exit(1)
	}
	defer db.Close()

	status, err := monitor.Collect(cfg.GetNetworkIF())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Status error: %s\n", err)
		os.Exit(1)
	}

	telegram := NewTelegram(cfg.Telegram.BotToken, cfg.Telegram.ChatID)
	msg := FormatReport(cfg, status, db)

	if err := telegram.SendMessage(msg); err != nil {
		fmt.Fprintf(os.Stderr, "Send error: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("报告发送成功")
}

func cmdCheck() {
	fmt.Printf("Stone Agent %s - 系统检查\n", Version)
	fmt.Println()

	fmt.Print("✓ 配置: ")
	cfg, err := LoadConfig(configPath)
	if err != nil {
		fmt.Printf("失败 - %s\n", err)
	} else {
		fmt.Printf("正常 (%s)\n", cfg.Server.Name)
	}

	fmt.Print("✓ 数据库: ")
	db, err := OpenDB()
	if err != nil {
		fmt.Printf("失败 - %s\n", err)
	} else {
		count, _ := db.GetRecordCount()
		fmt.Printf("正常 (%d 条记录)\n", count)
		db.Close()
	}

	fmt.Print("✓ Telegram: ")
	if cfg != nil && cfg.Telegram.BotToken != "" {
		telegram := NewTelegram(cfg.Telegram.BotToken, cfg.Telegram.ChatID)
		if err := telegram.SendMessage(fmt.Sprintf("Stone Agent %s 连接测试", Version)); err != nil {
			fmt.Printf("失败 - %s\n", err)
		} else {
			fmt.Println("正常")
		}
	} else {
		fmt.Println("跳过 (未配置)")
	}

	fmt.Print("✓ 操作: ")
	if cfg != nil && cfg.Actions.Enabled {
		fmt.Println("已启用")
	} else {
		fmt.Println("已禁用")
	}
}

func cmdHealth() {
	fmt.Printf("Stone Agent %s - 健康检查\n", Version)
	fmt.Println()

	fmt.Print("Agent:     ")
	fmt.Println("正常")

	fmt.Print("Database:  ")
	db, err := OpenDB()
	if err != nil {
		fmt.Println("异常")
	} else {
		fmt.Println("正常")
		db.Close()
	}

	fmt.Print("Telegram:  ")
	cfg, err := LoadConfig(configPath)
	if err == nil && cfg.Telegram.BotToken != "" {
		telegram := NewTelegram(cfg.Telegram.BotToken, cfg.Telegram.ChatID)
		if telegram.IsConnected() {
			fmt.Println("已连接")
		} else {
			fmt.Println("未连接")
		}
	} else {
		fmt.Println("未配置")
	}

	fmt.Print("Memory:    ")
	if cfg != nil {
		status, err := monitor.Collect(cfg.GetNetworkIF())
		if err == nil {
			fmt.Printf("%.0f%%\n", status.MemoryPercent)
		} else {
			fmt.Println("未知")
		}
	} else {
		fmt.Println("未知")
	}
}

func cmdDB() {
	fmt.Printf("Stone Agent %s - 数据库信息\n", Version)
	fmt.Println()

	db, err := OpenDB()
	if err != nil {
		fmt.Printf("数据库错误: %s\n", err)
		os.Exit(1)
	}
	defer db.Close()

	count, _ := db.GetRecordCount()
	size, _ := GetDBSize()

	fmt.Printf("路径:   %s\n", dbPath)
	fmt.Printf("记录数: %d\n", count)
	fmt.Printf("大小:   %s\n", monitor.FormatBytes(float64(size)))
}

func cmdPlugins() {
	fmt.Printf("Stone Agent %s - 可用插件\n", Version)
	fmt.Println()

	pluginMgr := initPlugins(nil)
	for _, name := range pluginMgr.Names() {
		fmt.Printf("  %s\n", name)
	}
}

func cmdServices() {
	fmt.Printf("Stone Agent %s - 服务状态\n", Version)
	fmt.Println()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Config error: %s\n", err)
		os.Exit(1)
	}

	checker := monitor.NewSystemdChecker()
	for _, svc := range cfg.Services {
		status := checker.Status(svc)
		emoji := "🔴"
		if status == "running" {
			emoji = "🟢"
		}
		fmt.Printf("  %s %s\n", emoji, svc)
	}
}

func cmdActions() {
	fmt.Printf("Stone Agent %s - 可用操作\n", Version)
	fmt.Println()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		fmt.Printf("配置错误: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("可用操作:")
	fmt.Println()

	fmt.Println("  restart-agent (重启Agent)")
	if cfg.Actions.Allow.Reboot {
		fmt.Println("  reboot (重启服务器)")
	}

	fmt.Println()
	fmt.Println("  service-control (服务控制)")
}

func cmdSecurity() {
	fmt.Printf("Stone Agent %s - 安全状态\n", Version)
	fmt.Println()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		fmt.Printf("配置错误: %s\n", err)
		os.Exit(1)
	}

	if !cfg.Security.Enabled {
		fmt.Println("安全模块未启用")
		return
	}

	mgr := security.NewManager(cfg.Security.Enabled)
	status := mgr.GetStatus()

	ufwEmoji := "🔴"
	if status.UFW.Active {
		ufwEmoji = "🟢"
	}

	fail2banEmoji := "🔴"
	if status.Fail2ban.Running {
		fail2banEmoji = "🟢"
	}

	fmt.Printf("🛡 安全\n\n")
	fmt.Printf("UFW:      %s\n", ufwEmoji)
	fmt.Printf("Fail2ban: %s\n", fail2banEmoji)

	if status.Fail2ban.Running && len(status.Fail2ban.Jails) > 0 {
		fmt.Printf("\nJails:\n")
		for _, jail := range status.Fail2ban.Jails {
			fmt.Printf("  %s\n", jail)
		}
		fmt.Printf("\nBan: %d\n", status.Fail2ban.BanCount)
	}

	fmt.Printf("\nSSH失败: %d\n", status.TotalFailed)

	// 显示最近事件
	db, err := OpenDB()
	if err == nil {
		defer db.Close()
		events, _ := db.GetRecentSecurityEvents(5)
		if len(events) > 0 {
			fmt.Printf("\n最近事件:\n")
			for _, event := range events {
				t := time.Unix(event.Timestamp, 0)
				fmt.Printf("  %s %s %s %s\n", t.Format("15:04"), event.Source, event.Action, event.IP)
			}
		}
	}
}

func cmdEvent() {
	if len(os.Args) < 4 {
		fmt.Println("用法: stone event <类型> <动作> <IP> [来源]")
		fmt.Println("示例: stone event security ban 1.2.3.4 sshd")
		return
	}

	eventType := os.Args[1]
	action := os.Args[2]
	ip := os.Args[3]
	source := ""
	if len(os.Args) > 4 {
		source = os.Args[4]
	}

	// 加载配置
	cfg, err := LoadConfig(configPath)
	if err != nil {
		fmt.Printf("配置错误: %s\n", err)
		os.Exit(1)
	}

	// 打开数据库
	db, err := OpenDB()
	if err != nil {
		fmt.Printf("数据库错误: %s\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// 写入数据库
	if err := db.InsertSecurityEvent(eventType, source, ip, source, action, ""); err != nil {
		fmt.Printf("写入事件失败: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("事件已记录: %s %s %s %s\n", eventType, action, ip, source)

	// 发送Telegram通知
	if cfg.Security.Notifications.Security && cfg.Telegram.BotToken != "" {
		telegram := NewTelegram(cfg.Telegram.BotToken, cfg.Telegram.ChatID)
		var msg string
		if action == "ban" {
			msg = fmt.Sprintf(`🚨 SSH攻击拦截

服务器: %s

攻击IP: %s
规则: %s
动作: 已封禁

时间 %s`,
				cfg.Server.Name,
				ip,
				source,
				time.Now().Format("15:04"),
			)
		} else if action == "unban" {
			msg = fmt.Sprintf(`🛡 SSH封禁解除

服务器: %s

IP: %s
规则: %s

时间 %s`,
				cfg.Server.Name,
				ip,
				source,
				time.Now().Format("15:04"),
			)
		} else {
			msg = fmt.Sprintf(`⚠️ 安全事件

服务器: %s

类型: %s
IP: %s
动作: %s

时间 %s`,
				cfg.Server.Name,
				eventType,
				ip,
				action,
				time.Now().Format("15:04"),
			)
		}

		if err := telegram.SendMessage(msg); err != nil {
			fmt.Printf("发送通知失败: %s\n", err)
		} else {
			fmt.Println("通知已发送")
		}
	}
}

func cmdAlerts() {
	fmt.Printf("Stone Agent %s - 告警状态\n", Version)
	fmt.Println()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		fmt.Printf("配置错误: %s\n", err)
		os.Exit(1)
	}

	db, err := OpenDB()
	if err != nil {
		fmt.Printf("数据库错误: %s\n", err)
		os.Exit(1)
	}
	defer db.Close()

	stateMgr := alerts.NewStateManager(db.conn)
	states, err := stateMgr.GetStatesByServer(cfg.Server.Name)
	if err != nil {
		fmt.Printf("查询错误: %s\n", err)
		os.Exit(1)
	}

	// 统计
	activeCount := 0
	for _, s := range states {
		if s.CurrentState != "running" && s.CurrentState != "unknown" {
			activeCount++
		}
	}

	fmt.Printf("🔔 告警\n\n")
	fmt.Printf("服务器: %s\n", cfg.Server.Name)
	fmt.Printf("活跃告警: %d\n", activeCount)
	fmt.Printf("总监控项: %d\n\n", len(states))

	if len(states) == 0 {
		fmt.Println("  暂无监控状态")
		return
	}

	fmt.Println("状态:")
	for _, s := range states {
		emoji := "🟢"
		if s.CurrentState == "stopped" || s.CurrentState == "missing" {
			emoji = "🔴"
		} else if s.CurrentState == "error" {
			emoji = "⚠️"
		} else if s.CurrentState == "unknown" {
			emoji = "⚪"
		}

		stateStr := string(s.CurrentState)
		if s.PreviousState != "" && s.PreviousState != s.CurrentState {
			stateStr = fmt.Sprintf("%s→%s", s.PreviousState, s.CurrentState)
		}

		fmt.Printf("  %s %s %s (%s)\n", emoji, s.Category, s.Name, stateStr)
	}
}

func cmdLogs() {
	fmt.Printf("Stone Agent %s - 操作日志\n", Version)
	fmt.Println()

	db, err := OpenDB()
	if err != nil {
		fmt.Printf("数据库错误: %s\n", err)
		os.Exit(1)
	}
	defer db.Close()

	limit := 20
	if len(os.Args) > 2 {
		if n, err := strconv.Atoi(os.Args[2]); err == nil && n > 0 {
			limit = n
		}
	}

	logs, err := db.GetRecentActionLogs(limit)
	if err != nil {
		fmt.Printf("查询错误: %s\n", err)
		os.Exit(1)
	}

	if len(logs) == 0 {
		fmt.Println("  暂无操作日志")
		return
	}

	for _, entry := range logs {
		t := time.Unix(entry.Timestamp, 0)
		fmt.Printf("  [%s] user=%d action=%s target=%s result=%s\n",
			t.Format("2006-01-02 15:04:05"),
			entry.UserID,
			entry.Action,
			entry.Target,
			entry.Result,
		)
	}
}

func cmdUninstall() {
	fmt.Println("正在卸载 Stone...")

	fmt.Println("停止服务...")
	runCommand("systemctl", "stop", "stone")
	runCommand("systemctl", "disable", "stone")

	os.Remove("/etc/systemd/system/stone.service")
	runCommand("systemctl", "daemon-reload")

	os.Remove("/usr/local/bin/stone")

	os.RemoveAll("/var/log/stone")
	os.RemoveAll("/var/lib/stone")

	fmt.Println("Stone 已卸载 (配置保留在 /etc/stone/)")
}

func initPlugins(cfg *Config) *plugins.Manager {
	mgr := plugins.NewManager()

	mgr.Register(system.New())

	ports := []int{443, 52536}
	if cfg != nil && len(cfg.Ports) > 0 {
		ports = cfg.Ports
	}
	mgr.Register(singbox.New(ports))

	var containers []string
	if cfg != nil && len(cfg.Containers) > 0 {
		containers = cfg.Containers
	}
	mgr.Register(docker.New(containers))

	return mgr
}

func initActions(cfg *Config) *actions.Manager {
	mgr := actions.NewManager()

	if cfg.Actions.Allow.Reboot {
		mgr.Register(actions.NewRebootAction())
	}
	if cfg.Actions.Allow.RestartAgent {
		mgr.Register(actions.NewRestartAgentAction())
	}

	// 设置冷却时间
	if cfg.Actions.Cooldown.RestartAgent > 0 {
		mgr.SetCooldown("restart_agent", cfg.Actions.Cooldown.RestartAgent)
	}
	if cfg.Actions.Cooldown.Reboot > 0 {
		mgr.SetCooldown("reboot", cfg.Actions.Cooldown.Reboot)
	}
	if cfg.Actions.Cooldown.ServiceRestart > 0 {
		mgr.SetCooldown("service_restart", cfg.Actions.Cooldown.ServiceRestart)
	}
	if cfg.Actions.Cooldown.ServiceStart > 0 {
		mgr.SetCooldown("service_start", cfg.Actions.Cooldown.ServiceStart)
	}
	if cfg.Actions.Cooldown.ServiceStop > 0 {
		mgr.SetCooldown("service_stop", cfg.Actions.Cooldown.ServiceStop)
	}

	return mgr
}

func runDaemon() {
	if err := setupLogging(); err != nil {
		fmt.Fprintf(os.Stderr, "Stone启动失败:\n日志初始化错误: %s\n", err)
		os.Exit(1)
	}

	logInfo("Stone Agent %s 启动中...", Version)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		logFatal("Stone启动失败:\n%s", err)
	}

	logInfo("服务器: %s (%s) | Profile: %s", cfg.Server.Name, cfg.Server.ID, cfg.Profile)

	db, err := OpenDB()
	if err != nil {
		logFatal("数据库初始化失败: %s", err)
	}
	logInfo("数据库: %s", dbPath)

	// 创建context和WaitGroup管理goroutine生命周期
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	pluginMgr := initPlugins(cfg)
	logInfo("插件已加载: %s", joinStrings(pluginMgr.Names()))

	// 初始化服务器身份
	identity := server.NewIdentity(cfg.Server.Name, cfg.Server.Location, cfg.Server.Role, configPath)
	if cfg.Server.ServerID == "" || cfg.Server.ServerID == "auto" {
		cfg.Server.ServerID = identity.LoadOrGenerate()
		logInfo("Server ID: %s (auto-generated)", cfg.Server.ServerID)
	} else {
		identity.LoadOrGenerate()
		logInfo("Server ID: %s", cfg.Server.ServerID)
	}

	actionMgr := initActions(cfg)
	confirmHandler := actions.NewConfirmHandler(actionMgr, cfg.Actions.ConfirmTimeout)
	logInfo("操作已启用: %v | 冷却已配置", cfg.Actions.Enabled)

	// 初始化安全模块
	securityMgr := security.NewManager(cfg.Security.Enabled)
	eventMgr := events.NewManager()
	logInfo("安全模块: %v", cfg.Security.Enabled)

	checker := monitor.NewSystemdChecker()

	telegram := NewTelegram(cfg.Telegram.BotToken, cfg.Telegram.ChatID)

	// 初始化告警引擎
	alertMgr := alerts.NewManager(db.conn, telegram.SendMessage, cfg.Server.Name, cfg.Alerts.Enabled, cfg.Alerts.NotifyRecovery, cfg.Alerts.Cooldown)
	logInfo("告警引擎: %v (冷却: %dm)", cfg.Alerts.Enabled, cfg.Alerts.Cooldown)

	// 注册安全事件处理器
	if cfg.Security.Enabled && cfg.Security.Notifications.Security {
		securityHandler := events.NewSecurityHandler(db.conn, telegram.SendMessage, cfg.Server.Name)
		eventMgr.Register("security", securityHandler)
	}

	// 初始化服务器注册表
	registry := server.NewRegistry(db.conn)
	registry.Register(cfg.Server.ServerID, cfg.Server.Name, cfg.Server.Location, cfg.Server.Role, Version)

	// 初始化心跳
	heartbeat := server.NewHeartbeat(db.conn, cfg.Server.ServerID, 60)
	wg.Add(1)
	go func() {
		defer wg.Done()
		heartbeat.Start(ctx)
	}()
	logInfo("心跳已启动 (间隔: 60s)")

	if err := telegram.SendMessage(fmt.Sprintf("🪨 Stone Agent %s 已启动 - %s", Version, EscapeMarkdown(cfg.Server.Name))); err != nil {
		logFatal("Telegram连接失败: %s", err)
	}
	logInfo("Telegram 已连接")

	cmd := NewCommand(telegram, cfg, db, checker, pluginMgr, actionMgr, confirmHandler, securityMgr, eventMgr)

	scheduler := NewScheduler(telegram, cfg, db, checker, cmd, alertMgr)
	wg.Add(1)
	go func() {
		defer wg.Done()
		scheduler.Start(ctx)
	}()
	logInfo("调度器已启动 (日报: %s, 监控: 5m)", cfg.Report.DailyTime)

	wg.Add(1)
	go func() {
		defer wg.Done()
		pollUpdates(ctx, telegram, cmd)
	}()
	logInfo("Telegram 轮询已启动")

	logInfo("Stone Agent %s 运行中", Version)

	// 等待信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logInfo("Stone 正在关闭...")
	cancel() // 通知所有goroutine退出

	// 等待goroutine退出
	wg.Wait()

	telegram.SendMessage(fmt.Sprintf("🔴 Stone Agent %s 已停止 - %s", Version, EscapeMarkdown(cfg.Server.Name)))
	db.Close()
	logInfo("Stone 已关闭")
}

func pollUpdates(ctx context.Context, telegram *Telegram, cmd *Command) {
	offset := 0
	startupTime := time.Now() // 记录启动时间，忽略旧回调

	for {
		select {
		case <-ctx.Done():
			logInfo("Telegram 轮询已停止")
			return
		default:
		}

		updates, err := telegram.GetUpdates(offset)
		if err != nil {
			logWarn("Telegram 连接断开")
			logInfo("正在重连...")
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1

			// 处理回调查询 - 忽略启动前的旧回调（防止重启循环）
			if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
				// 检查是否是重启相关的旧回调
				data := update.CallbackQuery.Data
				if strings.Contains(data, "confirm_reboot") || strings.Contains(data, "confirm_restart") {
					elapsed := time.Since(startupTime)
					if elapsed < 60*time.Second {
						logWarn("忽略旧回调: %s (启动后 %v)", data, elapsed.Round(time.Second))
						continue
					}
				}
				logInfo("回调: %s 来自 %d", data, update.CallbackQuery.From.ID)
				cmd.HandleCallbackQuery(update.CallbackQuery)
				continue
			}

			// 处理消息
			if update.Message != nil && update.Message.From != nil {
				logInfo("命令: %s 来自 %d", update.Message.Text, update.Message.From.ID)
				cmd.Handle(update.Message)
			}
		}
	}
}

func setupLogging() error {
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	mw := io.MultiWriter(os.Stderr, f)
	log.SetOutput(mw)
	log.SetFlags(0)
	return nil
}

func logInfo(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}

func logWarn(format string, v ...interface{}) {
	log.Printf("[WARN] "+format, v...)
}

func logError(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

func logFatal(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
	os.Exit(1)
}

func joinStrings(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}
