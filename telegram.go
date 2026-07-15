package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Telegram Telegram Bot客户端
type Telegram struct {
	BotToken  string
	ChatID    string
	client    *http.Client
	mu        sync.RWMutex
	connected bool
}

// TelegramUpdate Telegram更新
type TelegramUpdate struct {
	UpdateID int              `json:"update_id"`
	Message  *TelegramMessage `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

// TelegramMessage Telegram消息
type TelegramMessage struct {
	MessageID int           `json:"message_id"`
	From      *TelegramUser `json:"from"`
	Chat      *TelegramChat `json:"chat"`
	Text      string        `json:"text"`
}

// CallbackQuery 回调查询
type CallbackQuery struct {
	ID   string        `json:"id"`
	From *TelegramUser `json:"from"`
	Data string        `json:"data"`
	Message *TelegramMessage `json:"message"`
}

// TelegramUser Telegram用户
type TelegramUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
}

// TelegramChat Telegram聊天
type TelegramChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// TelegramResponse Telegram API响应
type TelegramResponse struct {
	OK     bool               `json:"ok"`
	Result []TelegramUpdate   `json:"result"`
}

// SendMessageResponse 发送消息响应
type SendMessageResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	Result      *TelegramMessage `json:"result"`
}

// InlineKeyboard Inline键盘
type InlineKeyboard struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton Inline键盘按钮
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

// NewTelegram 创建Telegram客户端
func NewTelegram(botToken, chatID string) *Telegram {
	return &Telegram{
		BotToken: botToken,
		ChatID:   chatID,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		connected: false,
	}
}

// SendMessage 发送消息（带重试）
func (t *Telegram) SendMessage(text string) error {
	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		err := t.doSendMessage(text, nil)
		if err == nil {
			t.mu.Lock()
			t.connected = true
			t.mu.Unlock()
			return nil
		}
		lastErr = err
		logWarn("SendMessage retry %d/%d: %v", i+1, maxRetries, err)
		time.Sleep(time.Duration(i+1) * 2 * time.Second)
	}

	t.mu.Lock()
	t.connected = false
	t.mu.Unlock()
	return lastErr
}

// SendMessageWithKeyboard 发送带键盘的消息
func (t *Telegram) SendMessageWithKeyboard(text string, keyboard *InlineKeyboard) error {
	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		err := t.doSendMessage(text, keyboard)
		if err == nil {
			t.mu.Lock()
			t.connected = true
			t.mu.Unlock()
			return nil
		}
		lastErr = err
		logWarn("SendMessageWithKeyboard retry %d/%d: %v", i+1, maxRetries, err)
		time.Sleep(time.Duration(i+1) * 2 * time.Second)
	}

	t.mu.Lock()
	t.connected = false
	t.mu.Unlock()
	return lastErr
}

// EditMessage 编辑消息
func (t *Telegram) EditMessage(chatID int64, messageID int64, text string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", t.BotToken)

	data := url.Values{}
	data.Set("chat_id", fmt.Sprintf("%d", chatID))
	data.Set("message_id", fmt.Sprintf("%d", messageID))
	data.Set("text", text)
	data.Set("parse_mode", "Markdown")

	resp, err := t.client.PostForm(apiURL, data)
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var result SendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram error: %s", result.Description)
	}

	return nil
}

// EditMessageWithKeyboard 编辑消息（带键盘）
func (t *Telegram) EditMessageWithKeyboard(chatID int64, messageID int64, text string, keyboard *InlineKeyboard) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", t.BotToken)

	data := url.Values{}
	data.Set("chat_id", fmt.Sprintf("%d", chatID))
	data.Set("message_id", fmt.Sprintf("%d", messageID))
	data.Set("text", text)
	data.Set("parse_mode", "Markdown")

	if keyboard != nil {
		keyboardJSON, _ := json.Marshal(keyboard)
		data.Set("reply_markup", string(keyboardJSON))
	}

	resp, err := t.client.PostForm(apiURL, data)
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result SendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram error: %s", result.Description)
	}

	return nil
}

// AnswerCallbackQuery 回复回调查询
func (t *Telegram) AnswerCallbackQuery(callbackID, text string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", t.BotToken)

	data := url.Values{}
	data.Set("callback_query_id", callbackID)
	data.Set("text", text)

	resp, err := t.client.PostForm(apiURL, data)
	if err != nil {
		return fmt.Errorf("answer callback: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func (t *Telegram) doSendMessage(text string, keyboard *InlineKeyboard) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.BotToken)

	data := url.Values{}
	data.Set("chat_id", t.ChatID)
	data.Set("text", text)
	data.Set("parse_mode", "Markdown")

	if keyboard != nil {
		keyboardJSON, err := json.Marshal(keyboard)
		if err != nil {
			return fmt.Errorf("marshal keyboard: %w", err)
		}
		data.Set("reply_markup", string(keyboardJSON))
	}

	resp, err := t.client.PostForm(apiURL, data)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result SendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram error: %s", result.Description)
	}

	return nil
}

// GetUpdates 获取更新（带重连）
func (t *Telegram) GetUpdates(offset int) ([]TelegramUpdate, error) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", t.BotToken)

	data := url.Values{}
	data.Set("offset", fmt.Sprintf("%d", offset))
	data.Set("timeout", "30")

	resp, err := t.client.PostForm(apiURL, data)
	if err != nil {
		t.mu.Lock()
		t.connected = false
		t.mu.Unlock()
		return nil, fmt.Errorf("get updates: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result TelegramResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram error")
	}

	t.mu.Lock()
	t.connected = true
	t.mu.Unlock()

	return result.Result, nil
}

// VerifyChatID 验证Chat ID
func (t *Telegram) VerifyChatID(messageChatID int64) bool {
	return fmt.Sprintf("%d", messageChatID) == t.ChatID
}

// IsConnected 检查连接状态
func (t *Telegram) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected
}

// EscapeMarkdown 转义Markdown特殊字符（Telegram Markdown格式）
func EscapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"`", "\\`",
	)
	return replacer.Replace(text)
}
