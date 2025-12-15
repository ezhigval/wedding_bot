package config

import (
	"os"
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
	GoogleSheetsID          string
	GoogleSheetsCredentials string
	GoogleSheetsSheetName   string
	GoogleSheetsInvitationsSheetName string
	GoogleSheetsAdminsSheetName      string
	GoogleSheetsTimelineSheetName    string
	GoogleSheetsRulesSheetName       string

	// SeatingAPIToken - токен для защищённых вызовов рассадки
	SeatingAPIToken string
)

// LoadConfig загружает конфигурацию из переменных окружения
func LoadConfig() error {
	// Загружаем .env файл если он существует
	_ = godotenv.Load()

	// Токен бота
	BotToken = strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	// Дополнительная очистка от пробелов и кавычек
	BotToken = strings.Trim(BotToken, `"'`)

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
		WeddingAddress = "Санкт-Петербург"
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

	// Токен для рассадки
	SeatingAPIToken = strings.TrimSpace(os.Getenv("SEATING_API_TOKEN"))
	SeatingAPIToken = strings.Trim(SeatingAPIToken, `"'`)

	return nil
}

