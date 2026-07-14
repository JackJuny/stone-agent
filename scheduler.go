package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/user/stone/alerts"
	"github.com/user/stone/monitor"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	telegram  *Telegram
	config    *Config
	db        *DB
	checker   monitor.ServiceChecker
	cmd       *Command
	alertMgr  *alerts.Manager
}

// NewScheduler 创建调度器
func NewScheduler(telegram *Telegram, config *Config, db *DB, checker monitor.ServiceChecker, cmd *Command, alertMgr *alerts.Manager) *Scheduler {
	return &Scheduler{
		telegram: telegram,
		config:   config,
		db:       db,
		checker:  checker,
		cmd:      cmd,
		alertMgr: alertMgr,
	}
}

// Start 启动调度器
func (s *Scheduler) Start(ctx context.Context) {
	go s.dailyReportLoop(ctx)
	go s.monitorLoop(ctx)
	go s.dbCleanLoop(ctx)
}

func (s *Scheduler) dailyReportLoop(ctx context.Context) {
	loc := s.config.Report.Location()

	for {
		select {
		case <-ctx.Done():
			logInfo("日报调度已停止")
			return
		default:
		}

		now := time.Now().In(loc)
		target := s.parseTargetTime(loc)
		if target == nil {
			logError("日报时间格式错误: %s，1分钟后重试", s.config.Report.DailyTime)
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Minute):
			}
			continue
		}

		next := time.Date(now.Year(), now.Month(), now.Day(), target.Hour(), target.Minute(), 0, 0, loc)
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}

		logInfo("daily report scheduled for %s", next.Format("2006-01-02 15:04:05"))

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(next)):
		}

		s.sendDailyReportWithRetry()
	}
}

func (s *Scheduler) parseTargetTime(loc *time.Location) *time.Time {
	t, err := time.ParseInLocation("15:04", s.config.Report.DailyTime, loc)
	if err != nil {
		return nil
	}
	return &t
}

func (s *Scheduler) sendDailyReportWithRetry() {
	logInfo("sending daily report...")

	status, err := monitor.Collect(s.config.GetNetworkIF())
	if err != nil {
		logError("failed to get status: %v", err)
		return
	}

	msg := FormatReport(s.config, status, s.db)
	if err := s.telegram.SendMessage(msg); err != nil {
		logWarn("daily report failed: %v", err)
		logInfo("retry in 5 minutes...")
		time.Sleep(5 * time.Minute)
		if err := s.telegram.SendMessage(msg); err != nil {
			logError("daily report retry failed: %v", err)
		} else {
			logInfo("retry success")
		}
		return
	}

	logInfo("daily report sent")
}

func (s *Scheduler) monitorLoop(ctx context.Context) {
	s.collectAndStore()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logInfo("监控调度已停止")
			return
		case <-ticker.C:
			status := s.collectAndStore()
			if status != nil {
				s.checkAlerts(status)
			}
			s.checkServices()
		}
	}
}

// collectAndStore 采集并存储系统状态
func (s *Scheduler) collectAndStore() *monitor.SystemStatus {
	status, err := monitor.Collect(s.config.GetNetworkIF())
	if err != nil {
		logError("collect: failed to get status: %v", err)
		return nil
	}

	if err := s.db.InsertStats(status, status.NetRX, status.NetTX, status.UptimeSec); err != nil {
		logError("collect: failed to insert stats: %v", err)
		return nil
	}

	// 更新最后采集时间
	if s.cmd != nil {
		s.cmd.UpdateLastCollect()
	}

	logInfo("stats collected: cpu=%.0f%% mem=%.0f%% disk=%.0f%%", status.CPU, status.MemoryPercent, status.DiskPercent)
	return status
}

// dbCleanLoop 定期清理过期数据
func (s *Scheduler) dbCleanLoop(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logInfo("数据库清理已停止")
			return
		case <-ticker.C:
			retentionDays := s.config.GetRetentionDays()
			if err := s.db.CleanOldRecords(retentionDays); err != nil {
				logError("db clean failed: %v", err)
			} else {
				logInfo("db cleaned: removed records older than %d days", retentionDays)
			}
		}
	}
}

func (s *Scheduler) checkAlerts(status *monitor.SystemStatus) {
	var alerts []string

	if status.CPU > float64(s.config.Monitor.CPUThreshold) {
		alerts = append(alerts, fmt.Sprintf("CPU %.0f%%", status.CPU))
	}

	if status.MemoryPercent > float64(s.config.Monitor.MemoryThreshold) {
		alerts = append(alerts, fmt.Sprintf("RAM %.0f%%", status.MemoryPercent))
	}

	if status.DiskPercent > float64(s.config.Monitor.DiskThreshold) {
		alerts = append(alerts, fmt.Sprintf("Disk %.0f%%", status.DiskPercent))
	}

	for _, alert := range alerts {
		msg := FormatAlert(s.config, "系统", alert)

		if err := s.telegram.SendMessage(msg); err != nil {
			logError("monitor: failed to send alert: %v", err)
		} else {
			logInfo("alert sent")
		}
	}
}

func (s *Scheduler) checkServices() {
	for _, svc := range s.config.Services {
		// 检查服务是否存在
		status := s.checker.Status(svc)
		installed := status != "stopped" || s.isServiceInstalled(svc)
		running := s.checker.IsRunning(svc)

		// 使用 Alert Engine 检查状态
		if s.alertMgr != nil {
			s.alertMgr.CheckService(svc, installed, running)
		}
	}
}

// isServiceInstalled 检查服务是否安装
func (s *Scheduler) isServiceInstalled(service string) bool {
	cmd := exec.Command("systemctl", "list-unit-files", service+".service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), service+".service")
}
