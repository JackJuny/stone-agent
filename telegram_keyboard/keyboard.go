package keyboard

import (
	"fmt"
)

// Button 按钮
type Button struct {
	Text         string
	CallbackData string
}

// Keyboard Inline键盘
type Keyboard struct {
	Rows [][]Button
}

// NewKeyboard 创建键盘
func NewKeyboard() *Keyboard {
	return &Keyboard{}
}

// AddRow 添加一行按钮
func (k *Keyboard) AddRow(buttons ...Button) *Keyboard {
	k.Rows = append(k.Rows, buttons)
	return k
}

// AddButton 添加单个按钮（新行）
func (k *Keyboard) AddButton(text, callbackData string) *Keyboard {
	k.Rows = append(k.Rows, []Button{{Text: text, CallbackData: callbackData}})
	return k
}

// ToTelegram 转换为Telegram格式
func (k *Keyboard) ToTelegram() map[string]interface{} {
	keyboard := make([][]map[string]string, len(k.Rows))
	for i, row := range k.Rows {
		keyboard[i] = make([]map[string]string, len(row))
		for j, btn := range row {
			keyboard[i][j] = map[string]string{
				"text":          btn.Text,
				"callback_data": btn.CallbackData,
			}
		}
	}
	return map[string]interface{}{
		"inline_keyboard": keyboard,
	}
}

// BackButton 返回按钮
func BackButton(callbackData string) Button {
	return Button{Text: "🔙 返回", CallbackData: callbackData}
}

// ConfirmButton 确认按钮
func ConfirmButton(action string) Button {
	return Button{Text: "✅ 确认执行", CallbackData: fmt.Sprintf("confirm_%s", action)}
}

// CancelButton 取消按钮
func CancelButton() Button {
	return Button{Text: "❌ 取消", CallbackData: "cancel"}
}
