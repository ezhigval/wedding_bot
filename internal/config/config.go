package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var (
	// BotToken - токен Telegram бота
	BotToken string

	// WeddingDate - дата свадьбы
	WeddingDate time.Time

	// GroomName - имя жениха
	GroomName string

	// BrideName - имя невесты
	BrideName string

	// WeddingAddress - адрес свадьбы
	WeddingAddress string

	// WebappURL - URL Mini App
	WebappURL string

	// AdminUserID - ID администратора
	AdminUserID string

	// DBPath - путь к базе данных
	DBPath string

	// PhotoPath - путь к фотографии для бота
	PhotoPath string

	// WebappPhotoPath - путь к фотографии для Mini App
	WebappPhotoPath string

	// WebappPath - путь к веб-приложению
	WebappPath string

	// GroomTelegram - телеграм-аккаунт жениха
	GroomTelegram string

	// BrideTelegram - телеграм-аккаунт невесты
	BrideTelegram string

	// GroupLink - ссылка на группу для гостей
	GroupLink string

	// GroupID - ID группы
	GroupID string

	// AdminsFile - файл с админами
	AdminsFile string

	// AdminsList - список админов из переменной окружения
	AdminsList []string

	// Google Sheets настройки
	GoogleSheetsID                   string
	GoogleSheetsCredentials          string
	GoogleSheetsCredentialsBase64    string
	GoogleSheetsSheetName            string
	GoogleSheetsInvitationsSheetName string
	GoogleSheetsAdminsSheetName      string
	GoogleSheetsTimelineSheetName    string
	GoogleSheetsRulesSheetName       string
	GoogleDriveFolderID              string
	GoogleDriveOAuthClientID         string
	GoogleDriveOAuthClientSecret     string
	GoogleDriveOAuthRefreshToken     string

	// SeatingAPIToken - токен для защищённых вызовов рассадки
	SeatingAPIToken string

	// WordleDictionaryPath - путь к локальному словарю для проверки слов
	WordleDictionaryPath string

	// Debug - флаг debug-режима
	Debug bool
)

// LoadConfig загружает конфигурацию из переменных окружения
func LoadConfig() error {
	// Загружаем .env файл если он существует (сначала .env.local, потом .env)
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load()

	// Токен бота
	BotToken = strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	// Дополнительная очистка от пробелов и кавычек
	BotToken = strings.Trim(BotToken, `"'`)

	Debug = parseBoolEnv(os.Getenv("DEBUG"))

	// Данные о свадьбе
	weddingDateStr := os.Getenv("WEDDING_DATE")
	if weddingDateStr == "" {
		weddingDateStr = "2026-06-05"
	}
	var err error
	WeddingDate, err = time.Parse("2006-01-02", weddingDateStr)
	if err != nil {
		// Если не удалось распарсить, используем дефолтную дату
		WeddingDate, _ = time.Parse("2006-01-02", "2026-06-05")
	}

	GroomName = os.Getenv("GROOM_NAME")
	if GroomName == "" {
		GroomName = "Валентин"
	}

	BrideName = os.Getenv("BRIDE_NAME")
	if BrideName == "" {
		BrideName = "Мария"
	}

	WeddingAddress = os.Getenv("WEDDING_ADDRESS")
	if WeddingAddress == "" {
		WeddingAddress = "Ресторан Марсала, Большой проспект Петроградской стороны, 84, Санкт-Петербург"
	}

	// URL Mini App
	WebappURL = os.Getenv("WEBAPP_URL")
	if WebappURL == "" {
		WebappURL = "https://your-webapp-url.com"
	}

	// ID администратора
	AdminUserID = os.Getenv("ADMIN_USER_ID")

	// Путь к базе данных
	DBPath = os.Getenv("DB_PATH")
	if DBPath == "" {
		DBPath = "data/wedding.db"
	}

	// Путь к фотографии
	PhotoPath = os.Getenv("PHOTO_PATH")
	if PhotoPath == "" {
		PhotoPath = "res/welcome_photo.jpeg"
	}

	WebappPhotoPath = os.Getenv("WEBAPP_PHOTO_PATH")
	if WebappPhotoPath == "" {
		WebappPhotoPath = "res/welcome_photo.jpeg"
	}

	// Путь к веб-приложению
	WebappPath = os.Getenv("WEBAPP_PATH")
	if WebappPath == "" {
		WebappPath = "webapp"
	}

	// Телеграм-аккаунты
	GroomTelegram = os.Getenv("GROOM_TELEGRAM")
	if GroomTelegram == "" {
		GroomTelegram = "ezhigval"
	}

	BrideTelegram = os.Getenv("BRIDE_TELEGRAM")
	if BrideTelegram == "" {
		BrideTelegram = "mrfilmpro"
	}

	// Ссылка на группу
	GroupLink = os.Getenv("GROUP_LINK")
	if GroupLink == "" {
		GroupLink = "https://t.me/+ow7ttcFCmoUzYzRi"
	}

	GroupID = os.Getenv("GROUP_ID")

	// Файл с админами
	AdminsFile = os.Getenv("ADMINS_FILE")
	if AdminsFile == "" {
		AdminsFile = "admins.json"
	}

	// Список админов из переменной окружения
	adminsEnv := os.Getenv("ADMINS")
	if adminsEnv == "" {
		adminsEnv = "@ezhigval, @mrfilmpro"
	}
	adminsParts := strings.Split(adminsEnv, ",")
	AdminsList = make([]string, 0, len(adminsParts))
	for _, admin := range adminsParts {
		admin = strings.TrimSpace(admin)
		admin = strings.TrimPrefix(admin, "@")
		if admin != "" {
			AdminsList = append(AdminsList, admin)
		}
	}

	// Google Sheets настройки
	GoogleSheetsID = os.Getenv("GOOGLE_SHEETS_ID")
	if GoogleSheetsID == "" {
		GoogleSheetsID = "15-S90u4kI97Kp1NRNhyyA_cuFriUwWAgmGEa80zZ5EI"
	}

	GoogleSheetsCredentials = os.Getenv("GOOGLE_SHEETS_CREDENTIALS")
	GoogleSheetsCredentialsBase64 = os.Getenv("GOOGLE_SHEETS_CREDENTIALS_BASE64")

	GoogleSheetsSheetName = os.Getenv("GOOGLE_SHEETS_SHEET_NAME")
	if GoogleSheetsSheetName == "" {
		GoogleSheetsSheetName = "Список гостей"
	}

	GoogleSheetsInvitationsSheetName = os.Getenv("GOOGLE_SHEETS_INVITATIONS_SHEET_NAME")
	if GoogleSheetsInvitationsSheetName == "" {
		GoogleSheetsInvitationsSheetName = "Пригласительные"
	}

	GoogleSheetsAdminsSheetName = os.Getenv("GOOGLE_SHEETS_ADMINS_SHEET_NAME")
	if GoogleSheetsAdminsSheetName == "" {
		GoogleSheetsAdminsSheetName = "Админ бота"
	}

	GoogleSheetsTimelineSheetName = os.Getenv("GOOGLE_SHEETS_TIMELINE_SHEET_NAME")
	if GoogleSheetsTimelineSheetName == "" {
		GoogleSheetsTimelineSheetName = "Публичная План-сетка"
	}

	GoogleSheetsRulesSheetName = os.Getenv("GOOGLE_SHEETS_RULES_SHEET_NAME")
	if GoogleSheetsRulesSheetName == "" {
		GoogleSheetsRulesSheetName = "Правила ИИ"
	}

	GoogleDriveFolderID = strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_FOLDER_ID"))
	GoogleDriveFolderID = strings.Trim(GoogleDriveFolderID, `"'`)

	GoogleDriveOAuthClientID = strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_OAUTH_CLIENT_ID"))
	GoogleDriveOAuthClientID = strings.Trim(GoogleDriveOAuthClientID, `"'`)

	GoogleDriveOAuthClientSecret = strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_OAUTH_CLIENT_SECRET"))
	GoogleDriveOAuthClientSecret = strings.Trim(GoogleDriveOAuthClientSecret, `"'`)

	GoogleDriveOAuthRefreshToken = strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_OAUTH_REFRESH_TOKEN"))
	GoogleDriveOAuthRefreshToken = strings.Trim(GoogleDriveOAuthRefreshToken, `"'`)

	// Токен для рассадки
	SeatingAPIToken = strings.TrimSpace(os.Getenv("SEATING_API_TOKEN"))
	SeatingAPIToken = strings.Trim(SeatingAPIToken, `"'`)

	// Словарь для Wordle
	WordleDictionaryPath = os.Getenv("WORDLE_DICTIONARY_PATH")
	if WordleDictionaryPath == "" {
		WordleDictionaryPath = "res/wordle_dictionary.txt"
	}

	return nil
}

func parseBoolEnv(raw string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false
	}

	return parsed
}

// IsDebug возвращает актуальное состояние debug-режима.
func IsDebug() bool {
	if Debug {
		return true
	}

	return parseBoolEnv(os.Getenv("DEBUG"))
}
