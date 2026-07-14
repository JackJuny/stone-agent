package monitor

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// GetNetworkStats 获取网卡流量统计
func GetNetworkStats(iface string) (rx, tx uint64, err error) {
	rxPath := fmt.Sprintf("/sys/class/net/%s/statistics/rx_bytes", iface)
	txPath := fmt.Sprintf("/sys/class/net/%s/statistics/tx_bytes", iface)

	rxData, err := os.ReadFile(rxPath)
	if err != nil {
		return 0, 0, fmt.Errorf("read rx: %w", err)
	}
	txData, err := os.ReadFile(txPath)
	if err != nil {
		return 0, 0, fmt.Errorf("read tx: %w", err)
	}

	rx, err = strconv.ParseUint(strings.TrimSpace(string(rxData)), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse rx: %w", err)
	}
	tx, err = strconv.ParseUint(strings.TrimSpace(string(txData)), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse tx: %w", err)
	}

	return rx, tx, nil
}

// GetLocalIP 获取本机IP地址
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return "unknown"
}
