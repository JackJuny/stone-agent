package main

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 主配置结构
type Config struct {
	Profile    string         `yaml:"profile"`
	Server     ServerConfig   `yaml:"server"`
	Telegram   TelegramConfig `yaml:"telegram"`
	Report     ReportConfig   `yaml:"report"`
	Monitor    MonitorConfig  `yaml:"monitor"`
	Database   DatabaseConfig `yaml:"database"`
	Actions    ActionsConfig  `yaml:"actions"`
	Security   SecurityConfig `yaml:"security"`
	Alerts     AlertsConfig   `yaml:"alerts"`
	Services   []string       `yaml:"services"`
	Ports      []int          `yaml:"ports"`
	Containers []string       `yaml:"containers"`
}

// ServerConfig 服务器信息
type ServerConfig struct {
	ID       string `yaml:"id"`
	ServerID string `yaml:"server_id"`
	Name     string `yaml:"name"`
	Location string `yaml:"location"`
	Role     string `yaml:"role"`
}

// TelegramConfig Telegram配置
type TelegramConfig struct {
	BotToken     string       `yaml:"bot_token"`
	ChatID       string       `yaml:"chat_id"`
	AllowedUsers []int64      `yaml:"allowed_users"`
	Users        []UserConfig `yaml:"users"`
}

// UserConfig 用户配置
type UserConfig struct {
	ID      int64    `yaml:"id"`
	Servers []string `yaml:"servers"`
}

// ReportConfig 日报配置
type ReportConfig struct {
	Timezone  string `yaml:"timezone"`
	DailyTime string `yaml:"daily_time"`
}

// MonitorConfig 监控阈值
type MonitorConfig struct {
	CPUThreshold    int    `yaml:"cpu_threshold"`
	MemoryThreshold int    `yaml:"memory_threshold"`
	DiskThreshold   int    `yaml:"disk_threshold"`
	NetworkIF       string `yaml:"network_interface"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	RetentionDays int `yaml:"retention_days"`
}

// ActionsConfig 操作配置
type ActionsConfig struct {
	Enabled        bool                       `yaml:"enabled"`
	ConfirmTimeout int                        `yaml:"confirm_timeout"`
	Allow          AllowConfig                `yaml:"allow"`
	Services       map[string]ServiceAllowConfig `yaml:"services"`
	Cooldown       CooldownConfig             `yaml:"cooldown"`
}

// AllowConfig 系统操作权限
type AllowConfig struct {
	Reboot       bool `yaml:"reboot"`
	Shutdown     bool `yaml:"shutdown"`
	RestartAgent bool `yaml:"restart_agent"`
}

// ServiceAllowConfig 服务操作权限（细化版）
type ServiceAllowConfig struct {
	Allow []string `yaml:"allow"`
}

// CooldownConfig 冷却配置
type CooldownConfig struct {
	RestartAgent    int `yaml:"restart_agent"`
	Reboot          int `yaml:"reboot"`
	ServiceRestart  int `yaml:"service_restart"`
	ServiceStart    int `yaml:"service_start"`
	ServiceStop     int `yaml:"service_stop"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	Enabled bool           `yaml:"enabled"`
	UFW     UFWConfig      `yaml:"ufw"`
	Fail2ban Fail2banConfig `yaml:"fail2ban"`
	Notifications NotificationsConfig `yaml:"notifications"`
}

// UFWConfig UFW配置
type UFWConfig struct {
	Enabled bool `yaml:"enabled"`
}

// Fail2banConfig Fail2ban配置
type Fail2banConfig struct {
	Enabled bool `yaml:"enabled"`
}

// NotificationsConfig 通知配置
type NotificationsConfig struct {
	Security bool `yaml:"security"`
}

// AlertsConfig 告警配置
type AlertsConfig struct {
	Enabled       bool `yaml:"enabled"`
	NotifyRecovery bool `yaml:"notify_recovery"`
	Cooldown      int  `yaml:"cooldown"` // 分钟
}

var timeRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// LoadConfig 加载配置文件
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{
		Profile: "default",
		Report: ReportConfig{
			Timezone:  "Asia/Singapore",
			DailyTime: "22:00",
		},
		Monitor: MonitorConfig{
			CPUThreshold:    90,
			MemoryThreshold: 90,
			DiskThreshold:   85,
			NetworkIF:       "eth0",
		},
		Database: DatabaseConfig{
			RetentionDays: 90,
		},
		Actions: ActionsConfig{
			Enabled:        true,
			ConfirmTimeout: 60,
			Allow: AllowConfig{
				Reboot:       true,
				Shutdown:     false,
				RestartAgent: true,
			},
			Services: make(map[string]ServiceAllowConfig),
			Cooldown: CooldownConfig{
				RestartAgent:   30,
				Reboot:         300,
				ServiceRestart: 30,
				ServiceStart:   10,
				ServiceStop:    10,
			},
		},
		Security: SecurityConfig{
			Enabled: true,
			UFW: UFWConfig{
				Enabled: true,
			},
			Fail2ban: Fail2banConfig{
				Enabled: true,
			},
			Notifications: NotificationsConfig{
				Security: true,
			},
		},
		Alerts: AlertsConfig{
			Enabled:       true,
			NotifyRecovery: true,
			Cooldown:      30,
		},
		Ports:      []int{443, 52536},
		Containers: []string{},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Telegram.BotToken == "" {
		return fmt.Errorf("[ERROR] Telegram Token 未配置")
	}
	if c.Telegram.ChatID == "" {
		return fmt.Errorf("[ERROR] Telegram Chat ID 未配置")
	}
	if !timeRe.MatchString(c.Report.DailyTime) {
		return fmt.Errorf("[ERROR] 日报时间格式错误: %q (应为 HH:MM，如 22:00)", c.Report.DailyTime)
	}
	if _, err := time.LoadLocation(c.Report.Timezone); err != nil {
		return fmt.Errorf("[ERROR] 时区配置错误: %q (%v)", c.Report.Timezone, err)
	}
	if c.Monitor.NetworkIF == "" {
		c.Monitor.NetworkIF = "eth0"
	}
	if c.Database.RetentionDays <= 0 {
		c.Database.RetentionDays = 90
	}
	if c.Actions.ConfirmTimeout <= 0 {
		c.Actions.ConfirmTimeout = 60
	}
	return nil
}

// Location 返回配置的时区
func (c *ReportConfig) Location() *time.Location {
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// GetNetworkIF 获取网络接口名称
func (c *Config) GetNetworkIF() string {
	if c.Monitor.NetworkIF == "" {
		return "eth0"
	}
	return c.Monitor.NetworkIF
}

// GetRetentionDays 获取数据保留天数
func (c *Config) GetRetentionDays() int {
	if c.Database.RetentionDays <= 0 {
		return 90
	}
	return c.Database.RetentionDays
}

// IsUserAllowed 检查用户是否有控制权限
func (c *Config) IsUserAllowed(userID int64) bool {
	if len(c.Telegram.AllowedUsers) == 0 {
		return fmt.Sprintf("%d", userID) == c.Telegram.ChatID
	}

	for _, id := range c.Telegram.AllowedUsers {
		if id == userID {
			return true
		}
	}
	return false
}

// IsServiceActionAllowed 检查服务操作是否允许（细化版）
func (c *Config) IsServiceActionAllowed(service, action string) bool {
	svcConfig, ok := c.Actions.Services[service]
	if !ok {
		return false
	}

	for _, allowed := range svcConfig.Allow {
		if allowed == action {
			return true
		}
	}
	return false
}

// GetCooldown 获取冷却时间
func (c *Config) GetCooldown(action string) int {
	switch action {
	case "restart_agent":
		return c.Actions.Cooldown.RestartAgent
	case "reboot":
		return c.Actions.Cooldown.Reboot
	case "service_restart":
		return c.Actions.Cooldown.ServiceRestart
	case "service_start":
		return c.Actions.Cooldown.ServiceStart
	case "service_stop":
		return c.Actions.Cooldown.ServiceStop
	default:
		return 0
	}
}
