package bot

import (
	"sync"
)

// PhotoModeUsers хранит множество user_id с включенным фоторежимом
var PhotoModeUsers = struct {
	sync.RWMutex
	users map[int64]bool
}{
	users: make(map[int64]bool),
}

// IsPhotoModeEnabled проверяет, включен ли фоторежим для пользователя
func IsPhotoModeEnabled(userID int64) bool {
	PhotoModeUsers.RLock()
	defer PhotoModeUsers.RUnlock()
	return PhotoModeUsers.users[userID]
}

// SetPhotoModeEnabled устанавливает фоторежим для пользователя
func SetPhotoModeEnabled(userID int64, enabled bool) {
	PhotoModeUsers.Lock()
	defer PhotoModeUsers.Unlock()
	if enabled {
		PhotoModeUsers.users[userID] = true
	} else {
		delete(PhotoModeUsers.users, userID)
	}
}

