package bot

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"wedding-bot/internal/config"
)

const (
	broadcastPresetMiniApp           = "miniapp"
	broadcastPresetConfirmAttendance = "confirm_attendance"
	broadcastPresetCancelAttendance  = "cancel_attendance"
	broadcastPresetQuestion          = "question"

	broadcastGuestAttendanceConfirm = "guest:attendance:confirm"
	broadcastGuestAttendanceCancel  = "guest:attendance:cancel"

	broadcastQuestionFooter = "Можете задать любой вопрос в общем чате"
)

type broadcastPresetButton struct {
	Key           string
	AdminLabel    string
	UserLabel     string
	URL           string
	CallbackData  string
	AppendsFooter bool
}

func broadcastPresetButtons() []broadcastPresetButton {
	return []broadcastPresetButton{
		{
			Key:        broadcastPresetMiniApp,
			AdminLabel: "💒 Открыть приглашение",
			UserLabel:  "💒 Открыть приглашение",
			URL:        strings.TrimSpace(config.WebappURL),
		},
		{
			Key:          broadcastPresetConfirmAttendance,
			AdminLabel:   "✅ Подтвердить присутствие",
			UserLabel:    "✅ Подтвердить присутствие",
			CallbackData: broadcastGuestAttendanceConfirm,
		},
		{
			Key:          broadcastPresetCancelAttendance,
			AdminLabel:   "❌ Отменить присутствие",
			UserLabel:    "❌ Отменить присутствие",
			CallbackData: broadcastGuestAttendanceCancel,
		},
		{
			Key:           broadcastPresetQuestion,
			AdminLabel:    "💬 Задать вопрос",
			UserLabel:     "💬 Задать вопрос",
			URL:           broadcastGroupLinkURL(),
			AppendsFooter: true,
		},
	}
}

func broadcastPresetButtonByKey(key string) (broadcastPresetButton, error) {
	for _, preset := range broadcastPresetButtons() {
		if preset.Key == key {
			return preset, nil
		}
	}

	return broadcastPresetButton{}, fmt.Errorf("неизвестный тип кнопки")
}

func broadcastBuildPresetInlineButton(key string) (tgbotapi.InlineKeyboardButton, error) {
	preset, err := broadcastPresetButtonByKey(key)
	if err != nil {
		return tgbotapi.InlineKeyboardButton{}, err
	}

	if preset.URL != "" {
		if err := validateBroadcastButtonURL(preset.URL); err != nil {
			return tgbotapi.InlineKeyboardButton{}, err
		}
		return tgbotapi.NewInlineKeyboardButtonURL(preset.UserLabel, preset.URL), nil
	}

	if preset.CallbackData != "" {
		return tgbotapi.NewInlineKeyboardButtonData(preset.UserLabel, preset.CallbackData), nil
	}

	return tgbotapi.InlineKeyboardButton{}, fmt.Errorf("для кнопки %s не настроено действие", preset.AdminLabel)
}

func ensureBroadcastPresetSelectionMap(state *BroadcastState) {
	if state == nil || state.SelectedPresetButtons != nil {
		return
	}

	state.SelectedPresetButtons = make(map[string]bool)
}

func isBroadcastPresetSelected(state *BroadcastState, key string) bool {
	if state == nil {
		return false
	}

	if state.SelectedPresetButtons != nil && state.SelectedPresetButtons[key] {
		return true
	}

	preset, err := broadcastPresetButtonByKey(key)
	if err != nil {
		return false
	}

	for _, button := range state.Buttons {
		if button.Text != preset.UserLabel {
			continue
		}
		if preset.URL != "" && broadcastButtonURL(button) == preset.URL {
			return true
		}
		if preset.CallbackData != "" && broadcastButtonCallbackData(button) == preset.CallbackData {
			return true
		}
	}

	return false
}

func toggleBroadcastPresetButton(state *BroadcastState, key string) error {
	if state == nil {
		return fmt.Errorf("состояние рассылки не найдено")
	}

	preset, err := broadcastPresetButtonByKey(key)
	if err != nil {
		return err
	}

	button, err := broadcastBuildPresetInlineButton(key)
	if err != nil {
		return err
	}

	ensureBroadcastPresetSelectionMap(state)

	if isBroadcastPresetSelected(state, key) {
		state.Buttons = removeBroadcastInlineButton(state.Buttons, button)
		delete(state.SelectedPresetButtons, key)
		return nil
	}

	state.Buttons = addBroadcastInlineButton(state.Buttons, button)
	state.SelectedPresetButtons[preset.Key] = true
	return nil
}

func addBroadcastInlineButton(buttons []tgbotapi.InlineKeyboardButton, button tgbotapi.InlineKeyboardButton) []tgbotapi.InlineKeyboardButton {
	for _, existing := range buttons {
		if broadcastButtonsEqual(existing, button) {
			return buttons
		}
	}

	return append(buttons, button)
}

func removeBroadcastInlineButton(buttons []tgbotapi.InlineKeyboardButton, target tgbotapi.InlineKeyboardButton) []tgbotapi.InlineKeyboardButton {
	if len(buttons) == 0 {
		return nil
	}

	result := make([]tgbotapi.InlineKeyboardButton, 0, len(buttons))
	for _, button := range buttons {
		if broadcastButtonsEqual(button, target) {
			continue
		}
		result = append(result, button)
	}

	return result
}

func broadcastButtonsEqual(left tgbotapi.InlineKeyboardButton, right tgbotapi.InlineKeyboardButton) bool {
	return left.Text == right.Text &&
		broadcastButtonURL(left) == broadcastButtonURL(right) &&
		broadcastButtonCallbackData(left) == broadcastButtonCallbackData(right)
}

func broadcastButtonCallbackData(button tgbotapi.InlineKeyboardButton) string {
	if button.CallbackData == nil {
		return ""
	}

	return *button.CallbackData
}

func broadcastGroupLinkURL() string {
	link := strings.TrimSpace(config.GroupLink)
	if link != "" {
		return link
	}

	groupID := strings.TrimSpace(config.GroupID)
	if groupID == "" {
		return ""
	}

	return "https://t.me/" + strings.TrimPrefix(groupID, "@")
}

func broadcastRenderedText(state *BroadcastState) string {
	if state == nil {
		return ""
	}

	text := strings.TrimSpace(state.Text)
	if !broadcastHasPresetButton(state, broadcastPresetQuestion) {
		return text
	}
	if strings.Contains(text, broadcastQuestionFooter) {
		return text
	}
	if text == "" {
		return broadcastQuestionFooter
	}

	return text + "\n\n" + broadcastQuestionFooter
}

func broadcastHasPresetButton(state *BroadcastState, key string) bool {
	return isBroadcastPresetSelected(state, key)
}

func broadcastAdminPresetButtonText(state *BroadcastState, key string) string {
	preset, err := broadcastPresetButtonByKey(key)
	if err != nil {
		return "Неизвестная кнопка"
	}

	prefix := "➕"
	if isBroadcastPresetSelected(state, key) {
		prefix = "✅"
	}

	return fmt.Sprintf("%s %s", prefix, preset.AdminLabel)
}
