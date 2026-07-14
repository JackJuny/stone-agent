# Stone Agent - Personal VPS Operations Assistant

轻量远程运维 Agent，通过 Telegram 管理多台 Linux VPS。

## 特点

- 单文件二进制，无依赖
- 无 Docker，无 Web 面板
- 低资源占用，systemd 常驻
- Telegram Bot 管理
- 多服务器支持
- SQLite 历史数据存储
- 模块化监控架构
- 插件系统
- Action 操作框架
- 安全权限控制
- 操作冷却机制
- 健康检查

## 编译

```bash
CGO_ENABLED=0 go build -ldflags="-s -w -X main.Version=v0.4.1" -o stone
```

## 安装

```bash
chmod +x install.sh
sudo ./install.sh
```

## 配置

编辑 `/etc/stone/config.yaml`:

```yaml
profile: default

server:
  id: sg01
  name: 新加坡VPS
  location: Singapore

telegram:
  bot_token: "YOUR_BOT_TOKEN"
  chat_id: "YOUR_CHAT_ID"
  allowed_users:
    - 123456789

report:
  timezone: Asia/Singapore
  daily_time: "22:00"

monitor:
  cpu_threshold: 90
  memory_threshold: 90
  disk_threshold: 85
  network_interface: eth0

database:
  retention_days: 90

actions:
  enabled: true
  confirm_timeout: 60
  allow:
    reboot: true
    shutdown: false
    restart_agent: true
  services:
    sing-box:
      allow:
        - status
        - start
        - stop
        - restart
    docker:
      allow:
        - status
        - restart
    caddy:
      allow:
        - status
  cooldown:
    restart_agent: 30
    reboot: 300
    service_restart: 30
    service_start: 10
    service_stop: 10

services:
  - sing-box
  - nginx
  - docker

ports:
  - 443
  - 52536

containers:
  - nginx
  - mysql
  - redis
```

## Telegram 命令

### 查看命令
| 命令 | 说明 |
|------|------|
| `/status` | 查看当前服务器状态 |
| `/health` | 健康检查 |
| `/info` | 查看系统详细信息 |
| `/hostname` | 查看主机信息 |
| `/network` | 查看网络流量统计 |
| `/disk` | 查看磁盘使用详情 |
| `/services` | 显示服务状态 |
| `/plugins` | 显示插件列表 |
| `/actions` | 显示可用操作 |
| `/logs` | 显示操作日志 |
| `/report` | 立即发送日报 |

### 控制命令
| 命令 | 说明 |
|------|------|
| `/service <名> <动作>` | 服务控制 (restart/start/stop/status) |
| `/restart` | 重启 Stone Agent |
| `/reboot` | 重启服务器 |

## CLI 命令

```bash
stone status    # 显示当前状态
stone report    # 立即发送日报
stone check     # 检查配置、数据库、Telegram连接
stone health    # 健康检查
stone db        # 显示数据库信息
stone plugins   # 显示可用插件
stone services  # 显示服务状态
stone actions   # 显示可用操作
stone logs [N]  # 显示最近N条操作日志
stone version   # 显示版本
stone uninstall # 卸载
```

## 安全控制

### 用户白名单
```yaml
telegram:
  allowed_users:
    - 123456789
```

### 服务权限（细化版）
```yaml
actions:
  services:
    sing-box:
      allow:
        - status
        - start
        - stop
        - restart
```

### 操作冷却
```yaml
actions:
  cooldown:
    restart_agent: 30
    reboot: 300
    service_restart: 30
```

## 架构

```
vps-agent/
├── main.go
├── config.go
├── database.go
├── telegram.go
├── command.go
├── scheduler.go
├── report.go
├── actions/
│   ├── manager.go
│   ├── service.go
│   ├── system.go
│   └── confirm.go
├── monitor/
│   ├── types.go
│   ├── cpu.go
│   ├── memory.go
│   ├── disk.go
│   ├── network.go
│   └── service.go
├── plugins/
│   ├── manager.go
│   ├── system/
│   ├── singbox/
│   └── docker/
├── config.yaml.example
├── stone.service
├── install.sh
├── go.mod
└── README.md
```

## 操作日志

所有控制操作记录到 SQLite:

```bash
stone logs      # 最近20条
stone logs 10   # 最近10条
```

## 卸载

```bash
stone uninstall
```

## 回滚方案

```bash
systemctl stop stone
cp /path/to/stone-v0.4.0 /usr/local/bin/stone
systemctl start stone
```

## 安全说明

- 无 Web 服务，无开放端口
- 仅指定 Chat ID 可操作
- 控制命令需要用户白名单授权
- 高危操作（reboot/restart）需要二次确认（Inline Keyboard）
- 操作冷却机制防止误操作
- 所有操作记录到数据库
- 服务名白名单验证
- 配置文件权限 600
