#!/bin/bash

set -e

BINARY="/usr/local/bin/stone"
CONFIG_DIR="/etc/stone"
CONFIG_FILE="/etc/stone/config.yaml"
SERVICE_FILE="/etc/systemd/system/stone.service"
LOG_DIR="/var/log/stone"
LOG_FILE="/var/log/stone/stone.log"
DATA_DIR="/var/lib/stone"

echo "Installing Stone Agent v0.3.0..."

if [ "$EUID" -ne 0 ]; then
    echo "Please run as root"
    exit 1
fi

if [ ! -f "./stone" ]; then
    echo "Binary not found. Please compile first."
    exit 1
fi

cp ./stone "$BINARY"
chmod 755 "$BINARY"
echo "Binary installed to $BINARY"

mkdir -p "$CONFIG_DIR"
chmod 700 "$CONFIG_DIR"

if [ ! -f "$CONFIG_FILE" ]; then
    cp ./config.yaml.example "$CONFIG_FILE"
    chmod 600 "$CONFIG_FILE"
    echo "Config created at $CONFIG_FILE"
else
    echo "Config already exists at $CONFIG_FILE"
fi

mkdir -p "$LOG_DIR"
touch "$LOG_FILE"
chmod 755 "$LOG_DIR"
chmod 644 "$LOG_FILE"

mkdir -p "$DATA_DIR"
chmod 755 "$DATA_DIR"

cp ./stone.service "$SERVICE_FILE"
systemctl daemon-reload
systemctl enable stone
systemctl start stone

echo ""
echo "Stone Agent v0.3.0 installed successfully!"
echo ""
echo "Config:   $CONFIG_FILE (mode 600)"
echo "Logs:     $LOG_FILE"
echo "Database: $DATA_DIR/stone.db"
echo ""
echo "Commands:"
echo "  systemctl status stone"
echo "  systemctl restart stone"
echo "  stone status"
echo "  stone version"
echo "  stone check"
echo "  stone db"
echo "  stone plugins"
echo "  stone services"
