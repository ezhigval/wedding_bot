package cache

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"wedding-bot/internal/config"
	"wedding-bot/internal/google_sheets"
)

var db *sql.DB

// GameStats представляет статистику игрока
type GameStats struct {
	UserID        int
	FirstName     string
	LastName      string
	TotalScore    int
	DragonScore   int
	FlappyScore   int
	CrosswordScore int
	WordleScore   int
	Rank          string
	LastUpdated   string
}

// InitGameStatsCache инициализирует базу данных для кэша игровой статистики
func InitGameStatsCache() error {
	// Создаем директорию data если её нет
	dataDir := filepath.Dir(config.DBPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории data: %w", err)
	}

	dbPath := "data/game_stats_cache.db"
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("ошибка открытия базы данных: %w", err)
	}

	// Создаем таблицу
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS game_stats_cache (
			user_id INTEGER PRIMARY KEY,
			first_name TEXT,
			last_name TEXT,
			total_score INTEGER DEFAULT 0,
			dragon_score INTEGER DEFAULT 0,
			flappy_score INTEGER DEFAULT 0,
			crossword_score INTEGER DEFAULT 0,
			wordle_score INTEGER DEFAULT 0,
			rank TEXT DEFAULT 'Незнакомец',
			last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	// Создаем индексы
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_game_stats_user_id ON game_stats_cache(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_game_stats_updated ON game_stats_cache(last_updated DESC)",
	}

	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			return fmt.Errorf("ошибка создания индекса: %w", err)
		}
	}

	log.Println("Кэш игровой статистики инициализирован")
	return nil
}

// GetCachedStats получает статистику из кэша
func GetCachedStats(userID int) (*GameStats, error) {
	if db == nil {
		return nil, fmt.Errorf("база данных не инициализирована")
	}

	query := `
		SELECT user_id, first_name, last_name, total_score, dragon_score,
		       flappy_score, crossword_score, wordle_score, rank, last_updated
		FROM game_stats_cache
		WHERE user_id = ?
	`

	var stats GameStats
	err := db.QueryRow(query, userID).Scan(
		&stats.UserID,
		&stats.FirstName,
		&stats.LastName,
		&stats.TotalScore,
		&stats.DragonScore,
		&stats.FlappyScore,
		&stats.CrosswordScore,
		&stats.WordleScore,
		&stats.Rank,
		&stats.LastUpdated,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("ошибка получения статистики из кэша: %w", err)
	}

	return &stats, nil
}

// SaveCachedStats сохраняет статистику в кэш
func SaveCachedStats(stats *GameStats) error {
	if db == nil {
		return fmt.Errorf("база данных не инициализирована")
	}

	query := `
		INSERT OR REPLACE INTO game_stats_cache
		(user_id, first_name, last_name, total_score, dragon_score,
		 flappy_score, crossword_score, wordle_score, rank, last_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	lastUpdated := stats.LastUpdated
	if lastUpdated == "" {
		lastUpdated = time.Now().Format(time.RFC3339)
	}

	rank := stats.Rank
	if rank == "" {
		rank = "Незнакомец"
	}

	_, err := db.Exec(query,
		stats.UserID,
		stats.FirstName,
		stats.LastName,
		stats.TotalScore,
		stats.DragonScore,
		stats.FlappyScore,
		stats.CrosswordScore,
		stats.WordleScore,
		rank,
		lastUpdated,
	)

	if err != nil {
		return fmt.Errorf("ошибка сохранения статистики в кэш: %w", err)
	}

	return nil
}

// ParseDateTime парсит строку даты в time.Time
func ParseDateTime(dtStr string) (time.Time, error) {
	if dtStr == "" {
		return time.Time{}, fmt.Errorf("пустая строка даты")
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05.000000",
	}

	for _, fmt := range formats {
		if t, err := time.Parse(fmt, dtStr); err == nil {
			return t, nil
		}
	}

	// Пробуем ISO формат
	dtStr = strings.ReplaceAll(dtStr, "Z", "+00:00")
	return time.Parse(time.RFC3339, dtStr)
}

// SyncGameStats синхронизирует статистику между Google Sheets и кэшем
func SyncGameStats(ctx context.Context, userID int, sheetsStats *google_sheets.GameStats, cachedStats *GameStats) (*GameStats, error) {
	// Если нет данных ни в одном источнике
	if sheetsStats == nil && cachedStats == nil {
		return &GameStats{
			UserID:        userID,
			FirstName:     "",
			LastName:      "",
			TotalScore:    0,
			DragonScore:   0,
			FlappyScore:   0,
			CrosswordScore: 0,
			WordleScore:   0,
			Rank:          "Незнакомец",
			LastUpdated:   time.Now().Format(time.RFC3339),
		}, nil
	}

	// Если есть только один источник - используем его
	if sheetsStats == nil {
		return cachedStats, nil
	}

	if cachedStats == nil {
		// Добавляем текущую дату если её нет
		if sheetsStats.LastUpdated == "" {
			sheetsStats.LastUpdated = time.Now().Format(time.RFC3339)
		}
		// Сохраняем в кэш
		stats := &GameStats{
			UserID:        sheetsStats.UserID,
			FirstName:     sheetsStats.FirstName,
			LastName:      sheetsStats.LastName,
			TotalScore:    sheetsStats.TotalScore,
			DragonScore:   sheetsStats.DragonScore,
			FlappyScore:   sheetsStats.FlappyScore,
			CrosswordScore: sheetsStats.CrosswordScore,
			WordleScore:   sheetsStats.WordleScore,
			Rank:          sheetsStats.Rank,
			LastUpdated:   sheetsStats.LastUpdated,
		}
		if err := SaveCachedStats(stats); err != nil {
			log.Printf("Ошибка сохранения в кэш: %v", err)
		}
		return stats, nil
	}

	// Сравниваем даты
	sheetsDT, err1 := ParseDateTime(sheetsStats.LastUpdated)
	cachedDT, err2 := ParseDateTime(cachedStats.LastUpdated)

	// Если даты нет в одном из источников, считаем его устаревшим
	if err1 != nil && err2 != nil {
		// Оба без дат - используем тот, у которого больше счет
		if sheetsStats.TotalScore >= cachedStats.TotalScore {
			sheetsStats.LastUpdated = time.Now().Format(time.RFC3339)
			stats := &GameStats{
				UserID:        sheetsStats.UserID,
				FirstName:     sheetsStats.FirstName,
				LastName:      sheetsStats.LastName,
				TotalScore:    sheetsStats.TotalScore,
				DragonScore:   sheetsStats.DragonScore,
				FlappyScore:   sheetsStats.FlappyScore,
				CrosswordScore: sheetsStats.CrosswordScore,
				WordleScore:   sheetsStats.WordleScore,
				Rank:          sheetsStats.Rank,
				LastUpdated:   sheetsStats.LastUpdated,
			}
			if err := SaveCachedStats(stats); err != nil {
				log.Printf("Ошибка сохранения в кэш: %v", err)
			}
			return stats, nil
		}
		return cachedStats, nil
	}

	if err1 != nil {
		// В Sheets нет даты - используем кэш
		return cachedStats, nil
	}

	if err2 != nil {
		// В кэше нет даты - используем Sheets и обновляем кэш
		stats := &GameStats{
			UserID:        sheetsStats.UserID,
			FirstName:     sheetsStats.FirstName,
			LastName:      sheetsStats.LastName,
			TotalScore:    sheetsStats.TotalScore,
			DragonScore:   sheetsStats.DragonScore,
			FlappyScore:   sheetsStats.FlappyScore,
			CrosswordScore: sheetsStats.CrosswordScore,
			WordleScore:   sheetsStats.WordleScore,
			Rank:          sheetsStats.Rank,
			LastUpdated:   sheetsStats.LastUpdated,
		}
		if err := SaveCachedStats(stats); err != nil {
			log.Printf("Ошибка сохранения в кэш: %v", err)
		}
		return stats, nil
	}

	// Сравниваем даты
	if sheetsDT.After(cachedDT) {
		// Sheets новее - обновляем кэш
		stats := &GameStats{
			UserID:        sheetsStats.UserID,
			FirstName:     sheetsStats.FirstName,
			LastName:      sheetsStats.LastName,
			TotalScore:    sheetsStats.TotalScore,
			DragonScore:   sheetsStats.DragonScore,
			FlappyScore:   sheetsStats.FlappyScore,
			CrosswordScore: sheetsStats.CrosswordScore,
			WordleScore:   sheetsStats.WordleScore,
			Rank:          sheetsStats.Rank,
			LastUpdated:   sheetsStats.LastUpdated,
		}
		if err := SaveCachedStats(stats); err != nil {
			log.Printf("Ошибка сохранения в кэш: %v", err)
		}
		return stats, nil
	} else if cachedDT.After(sheetsDT) {
		// Кэш новее - возвращаем кэш (но не обновляем Sheets здесь, это делается отдельно)
		return cachedStats, nil
	} else {
		// Даты равны - используем Sheets (как основной источник)
		stats := &GameStats{
			UserID:        sheetsStats.UserID,
			FirstName:     sheetsStats.FirstName,
			LastName:      sheetsStats.LastName,
			TotalScore:    sheetsStats.TotalScore,
			DragonScore:   sheetsStats.DragonScore,
			FlappyScore:   sheetsStats.FlappyScore,
			CrosswordScore: sheetsStats.CrosswordScore,
			WordleScore:   sheetsStats.WordleScore,
			Rank:          sheetsStats.Rank,
			LastUpdated:   sheetsStats.LastUpdated,
		}
		if err := SaveCachedStats(stats); err != nil {
			log.Printf("Ошибка сохранения в кэш: %v", err)
		}
		return stats, nil
	}
}

