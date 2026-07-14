#!/bin/bash

# Stone Agent 一键部署脚本
# 从 GitHub 拉取代码，编译，安装
# 用法: curl -sL https://raw.githubusercontent.com/JackJuny/stone-agent/master/install.sh | bash

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 配置
REPO_URL="https://github.com/JackJuny/stone-agent.git"
INSTALL_DIR="/tmp/stone-build"
BINARY="/usr/local/bin/stone"
CONFIG_DIR="/etc/stone"
CONFIG_FILE="/etc/stone/config.yaml"
SERVICE_FILE="/etc/systemd/system/stone.service"
LOG_DIR="/var/log/stone"
DATA_DIR="/var/lib/stone"
GO_BIN="/usr/local/go/bin"

# 确保 Go 在 PATH 中
export PATH=$PATH:$GO_BIN

# 检查 root 权限
if [ "$EUID" -ne 0 ]; then
    log_error "请以 root 身份运行: sudo bash install.sh"
    exit 1
fi

echo "🪨 Stone Agent 一键部署"
echo "========================"
echo ""

# 1. 检查依赖
log_info "检查依赖..."

# 检查 git
if ! command -v git &> /dev/null; then
    log_warn "git 未安装，正在安装..."
    if command -v apt-get &> /dev/null; then
        apt-get update -qq && apt-get install -y -qq git
    elif command -v yum &> /dev/null; then
        yum install -y -q git
    else
        log_error "请先安装 git"
        exit 1
    fi
fi

# 安装字体库（支持 emoji 显示）
log_info "检查字体库..."
if command -v apt-get &> /dev/null; then
    # Debian/Ubuntu
    if ! dpkg -l | grep -q fonts-noto-color-emoji 2>/dev/null; then
        log_info "安装 emoji 字体..."
        apt-get install -y -qq fonts-noto-color-emoji fonts-noto-cjk 2>/dev/null || true
    fi
elif command -v yum &> /dev/null; then
    # CentOS/RHEL
    if ! rpm -q google-noto-emoji-color-fonts 2>/dev/null; then
        log_info "安装 emoji 字体..."
        yum install -y -q google-noto-emoji-color-fonts google-noto-sans-cjk-fonts 2>/dev/null || true
    fi
fi
log_info "字体库检查完成"

# 检查 Go
if ! command -v go &> /dev/null && [ ! -f "$GO_BIN/go" ]; then
    log_warn "Go 未安装，正在安装..."
    
    # 检测架构
    ARCH=$(uname -m)
    if [ "$ARCH" = "x86_64" ]; then
        GO_ARCH="amd64"
    elif [ "$ARCH" = "aarch64" ]; then
        GO_ARCH="arm64"
    else
        log_error "不支持的架构: $ARCH"
        exit 1
    fi
    
    GO_VERSION="1.21.13"
    GO_URL="https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    
    cd /tmp
    wget -q "$GO_URL" -O go.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go.tar.gz
    rm go.tar.gz
    
    # 添加到 PATH（永久生效）
    if ! grep -q "/usr/local/go/bin" /etc/profile.d/go.sh 2>/dev/null; then
        echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
    fi
    
    log_info "Go ${GO_VERSION} 安装完成"
else
    log_info "Go 已安装: $(go version 2>/dev/null || $GO_BIN/go version 2>/dev/null)"
fi

# 2. 克隆代码
log_info "拉取代码..."
rm -rf "$INSTALL_DIR"
git clone --depth 1 "$REPO_URL" "$INSTALL_DIR"

# 3. 编译
log_info "编译中..."
cd "$INSTALL_DIR"

# 获取版本
VERSION=$(grep -oP 'Version\s*=\s*"\K[^"]+' main.go 2>/dev/null || echo "v0.6.2")

# 下载依赖
$GO_BIN/go mod tidy

CGO_ENABLED=0 $GO_BIN/go build -ldflags="-s -w -X main.Version=${VERSION}" -o stone

if [ ! -f stone ]; then
    log_error "编译失败"
    exit 1
fi

log_info "编译完成: $(ls -lh stone | awk '{print $5}')"

# 4. 停止旧服务
if systemctl is-active --quiet stone 2>/dev/null; then
    log_info "停止旧服务..."
    systemctl stop stone
fi

# 5. 安装二进制
log_info "安装二进制..."
cp stone "$BINARY"
chmod 755 "$BINARY"

# 6. 创建目录
mkdir -p "$CONFIG_DIR"
chmod 700 "$CONFIG_DIR"

mkdir -p "$LOG_DIR"
chmod 755 "$LOG_DIR"

mkdir -p "$DATA_DIR"
chmod 755 "$DATA_DIR"

# 7. 安装配置
if [ ! -f "$CONFIG_FILE" ]; then
    cp config.yaml.example "$CONFIG_FILE"
    chmod 600 "$CONFIG_FILE"
    log_info "配置文件已创建: $CONFIG_FILE"
else
    log_info "配置文件已存在: $CONFIG_FILE"
fi

# 8. 安装 systemd 服务
cp stone.service "$SERVICE_FILE"
systemctl daemon-reload
systemctl enable stone

# 9. 启动服务
log_info "启动服务..."
systemctl start stone

# 10. 检查状态
sleep 2
if systemctl is-active --quiet stone; then
    log_info "Stone Agent 启动成功!"
else
    log_error "Stone Agent 启动失败"
    systemctl status stone --no-pager
    exit 1
fi

# 清理
rm -rf "$INSTALL_DIR"

echo ""
echo "========================"
echo "🪨 Stone Agent 安装完成"
echo "========================"
echo ""
echo "版本:     ${VERSION}"
echo "二进制:   ${BINARY}"
echo "配置:     ${CONFIG_FILE}"
echo "日志:     ${LOG_DIR}/stone.log"
echo "数据库:   ${DATA_DIR}/stone.db"
echo ""
echo "命令:"
echo "  systemctl status stone    # 查看状态"
echo "  systemctl restart stone   # 重启服务"
echo "  stone version             # 查看版本"
echo "  stone status              # 查看状态"
echo "  stone check               # 检查配置"
echo "  stone security            # 安全状态"
echo "  stone alerts              # 告警状态"
echo ""
echo "下一步:"
echo "  1. 编辑配置: sudo vim ${CONFIG_FILE}"
echo "  2. 配置 Telegram Bot Token 和 Chat ID"
echo "  3. 重启服务: sudo systemctl restart stone"
