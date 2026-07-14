package server

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"
)

// Identity 服务器身份
type Identity struct {
	ServerID   string
	Name       string
	Location   string
	Role       string
	Version    string
	configPath string
}

// NewIdentity 创建服务器身份
func NewIdentity(name, location, role, configPath string) *Identity {
	return &Identity{
		Name:       name,
		Location:   location,
		Role:       role,
		configPath: configPath,
	}
}

// LoadOrGenerate 加载或生成server_id
func (i *Identity) LoadOrGenerate() string {
	// 尝试从配置文件读取
	if i.configPath != "" {
		data, err := os.ReadFile(i.configPath)
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "server_id:") {
					id := strings.TrimPrefix(line, "server_id:")
					id = strings.TrimSpace(id)
					id = strings.Trim(id, "\"")
					if id != "" && id != "auto" {
						i.ServerID = id
						return id
					}
				}
			}
		}
	}

	// 生成新的server_id
	i.ServerID = i.generateID()
	i.saveToFile()
	return i.ServerID
}

func (i *Identity) generateID() string {
	// 生成格式: us-la-01-xxxx
	locationCode := strings.ToLower(strings.ReplaceAll(i.Location, " ", ""))
	if len(locationCode) > 6 {
		locationCode = locationCode[:6]
	}

	b := make([]byte, 2)
	rand.Read(b)
	suffix := fmt.Sprintf("%x", b)

	return fmt.Sprintf("%s-%s", locationCode, suffix)
}

func (i *Identity) saveToFile() {
	if i.configPath == "" {
		return
	}

	data, err := os.ReadFile(i.configPath)
	if err != nil {
		return
	}

	content := string(data)

	// 检查是否已有server_id
	if strings.Contains(content, "server_id:") {
		// 替换现有的server_id
		lines := strings.Split(content, "\n")
		for idx, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "server_id:") {
				lines[idx] = fmt.Sprintf("  server_id: \"%s\"", i.ServerID)
				break
			}
		}
		os.WriteFile(i.configPath, []byte(strings.Join(lines, "\n")), 0644)
	} else {
		// 在文件开头添加server_id注释和配置
		header := fmt.Sprintf("# Server ID (auto-generated, do not modify)\nserver_id: \"%s\"\n\n", i.ServerID)
		os.WriteFile(i.configPath, []byte(header+content), 0644)
	}
}

// GetID 获取server_id
func (i *Identity) GetID() string {
	return i.ServerID
}

// GetName 获取服务器名称
func (i *Identity) GetName() string {
	return i.Name
}

// GetLocation 获取位置
func (i *Identity) GetLocation() string {
	return i.Location
}

// GetRole 获取角色
func (i *Identity) GetRole() string {
	return i.Role
}
