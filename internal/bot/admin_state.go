package bot

import (
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AdminInputMode string

const (
	AdminInputModeNone           AdminInputMode = ""
	AdminInputModeWordleAdd      AdminInputMode = "wordle_add"
	AdminInputModeCrosswordAdd   AdminInputMode = "crossword_add"
	AdminInputModeGroupBroadcast AdminInputMode = "group_broadcast"
)

type GroupBroadcastState struct {
	Text    string
	PhotoID string
	Buttons []tgbotapi.InlineKeyboardButton
}

var (
	adminInputModesMu sync.RWMutex
	adminInputModes   = make(map[int64]AdminInputMode)

	groupBroadcastMu     sync.RWMutex
	groupBroadcastStates = make(map[int64]*GroupBroadcastState)
)

func SetAdminInputMode(userID int64, mode AdminInputMode) {
	adminInputModesMu.Lock()
	defer adminInputModesMu.Unlock()
	adminInputModes[userID] = mode
}

func GetAdminInputMode(userID int64) AdminInputMode {
	adminInputModesMu.RLock()
	defer adminInputModesMu.RUnlock()
	return adminInputModes[userID]
}

func ClearAdminInputMode(userID int64) {
	adminInputModesMu.Lock()
	defer adminInputModesMu.Unlock()
	delete(adminInputModes, userID)
}

func InitGroupBroadcastState(userID int64) {
	groupBroadcastMu.Lock()
	defer groupBroadcastMu.Unlock()
	groupBroadcastStates[userID] = &GroupBroadcastState{
		Buttons: make([]tgbotapi.InlineKeyboardButton, 0),
	}
}

func GetGroupBroadcastState(userID int64) *GroupBroadcastState {
	groupBroadcastMu.RLock()
	defer groupBroadcastMu.RUnlock()
	return groupBroadcastStates[userID]
}

func ClearGroupBroadcastState(userID int64) {
	groupBroadcastMu.Lock()
	defer groupBroadcastMu.Unlock()
	delete(groupBroadcastStates, userID)
}
