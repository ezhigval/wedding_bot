package google_sheets

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

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

// GetRankByScore определяет звание игрока по общему счету
func GetRankByScore(totalScore int) string {
	if totalScore < 50 {
		return "Незнакомец"
	} else if totalScore < 100 {
		return "Ты хто?"
	} else if totalScore < 150 {
		return "Люся"
	} else if totalScore < 200 {
		return "Бедный родственник"
	} else if totalScore < 300 {
		return "Братуха"
	} else if totalScore < 400 {
		return "Батя в здании"
	} else {
		return "Монстр"
	}
}

// GetGuestNameByUserID получает имя и фамилию гостя по user_id
func GetGuestNameByUserID(ctx context.Context, userID int) (string, string, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return "", "", err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsSheetName

	readRange := fmt.Sprintf("%s!A:F", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return "", "", fmt.Errorf("ошибка чтения значений: %w", err)
	}

	userIDStr := fmt.Sprintf("%d", userID)
	for _, row := range resp.Values {
		if len(row) > 5 {
			if val, ok := row[5].(string); ok {
				if strings.TrimSpace(val) == userIDStr {
					// Нашли user_id в столбце F
					fullName := ""
					if val, ok := row[0].(string); ok {
						fullName = strings.TrimSpace(val)
					}
					if fullName != "" {
						nameParts := strings.SplitN(fullName, " ", 2)
						firstName := nameParts[0]
						lastName := ""
						if len(nameParts) > 1 {
							lastName = nameParts[1]
						}
						return firstName, lastName, nil
					}
				}
			}
		}
	}

	return "", "", nil
}

// GetGameStats получает статистику игрока по user_id
func GetGameStats(ctx context.Context, userID int) (*GameStats, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Игры"

	// Проверяем существование листа
	if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
		// Пытаемся создать через EnsureRequiredSheets
		if err := EnsureRequiredSheets(ctx); err != nil {
			return nil, fmt.Errorf("ошибка создания листа: %w", err)
		}
		// Проверяем еще раз
		if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
			return nil, nil
		}
	}

	readRange := fmt.Sprintf("%s!A:J", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	userIDStr := fmt.Sprintf("%d", userID)
	for i, row := range resp.Values {
		if i == 0 {
			continue // Пропускаем заголовок
		}

		if len(row) == 0 {
			continue
		}

		rowUserID := ""
		if val, ok := row[0].(string); ok {
			rowUserID = strings.TrimSpace(val)
		}

		if rowUserID != userIDStr {
			continue
		}

		// Нашли строку пользователя
		stats := &GameStats{
			UserID: userID,
		}

		if len(row) > 1 {
			if val, ok := row[1].(string); ok {
				stats.FirstName = strings.TrimSpace(val)
			}
		}

		if len(row) > 2 {
			if val, ok := row[2].(string); ok {
				stats.LastName = strings.TrimSpace(val)
			}
		}

		if len(row) > 3 {
			if val, ok := row[3].(string); ok {
				if score, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
					stats.TotalScore = score
				}
			}
		}

		if len(row) > 4 {
			if val, ok := row[4].(string); ok {
				if score, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
					stats.DragonScore = score
				}
			}
		}

		if len(row) > 5 {
			if val, ok := row[5].(string); ok {
				if score, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
					stats.FlappyScore = score
				}
			}
		}

		if len(row) > 6 {
			if val, ok := row[6].(string); ok {
				if score, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
					stats.CrosswordScore = score
				}
			}
		}

		if len(row) > 7 {
			if val, ok := row[7].(string); ok {
				if score, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
					stats.WordleScore = score
				}
			}
		}

		if len(row) > 8 {
			if val, ok := row[8].(string); ok {
				stats.Rank = strings.TrimSpace(val)
			}
		}
		if stats.Rank == "" {
			stats.Rank = "Незнакомец"
		}

		if len(row) > 9 {
			if val, ok := row[9].(string); ok {
				stats.LastUpdated = strings.TrimSpace(val)
			}
		}

		return stats, nil
	}

	return nil, nil
}

// UpdateGameScore обновляет счет игрока в конкретной игре
func UpdateGameScore(ctx context.Context, userID int, gameType string, score int) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Игры"

	// Получаем имя и фамилию пользователя
	firstName, lastName, _ := GetGuestNameByUserID(ctx, userID)

	// Проверяем существование листа
	if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
		// Пытаемся создать через EnsureRequiredSheets
		if err := EnsureRequiredSheets(ctx); err != nil {
			return fmt.Errorf("ошибка создания листа: %w", err)
		}
		// Проверяем еще раз
		if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
			return fmt.Errorf("лист все еще не существует: %w", err)
		}
	}

	// Получаем текущую статистику
	stats, err := GetGameStats(ctx, userID)
	if err != nil {
		return err
	}

	// Если статистики нет, создаем новую
	if stats == nil {
		stats = &GameStats{
			UserID:      userID,
			FirstName:   firstName,
			LastName:    lastName,
			TotalScore:  0,
			DragonScore: 0,
			FlappyScore: 0,
			CrosswordScore: 0,
			WordleScore: 0,
			Rank:        "Незнакомец",
		}
	} else {
		// Обновляем имя и фамилию если они изменились
		if firstName != "" {
			stats.FirstName = firstName
		}
		if lastName != "" {
			stats.LastName = lastName
		}
	}

	// Конвертируем игровые очки в рейтинговые по формулам
	var ratingPoints int
	switch gameType {
	case "dragon":
		ratingPoints = score / 200
		stats.DragonScore += ratingPoints
	case "flappy":
		ratingPoints = score / 2
		stats.FlappyScore += ratingPoints
	case "crossword":
		ratingPoints = score * 25
		stats.CrosswordScore += ratingPoints
	case "wordle":
		ratingPoints = score * 5
		stats.WordleScore += ratingPoints
	default:
		return fmt.Errorf("неизвестный тип игры: %s", gameType)
	}

	// Пересчитываем общий счет
	stats.TotalScore = stats.DragonScore + stats.FlappyScore + stats.CrosswordScore + stats.WordleScore

	// Определяем звание
	stats.Rank = GetRankByScore(stats.TotalScore)
	stats.LastUpdated = time.Now().Format(time.RFC3339)

	// Обновляем или создаем строку
	readRange := fmt.Sprintf("%s!A:J", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения значений: %w", err)
	}

	userIDStr := fmt.Sprintf("%d", userID)
	foundRow := -1
	for i, row := range resp.Values {
		if i == 0 {
			continue // Пропускаем заголовок
		}
		if len(row) > 0 {
			rowUserID := ""
			if val, ok := row[0].(string); ok {
				rowUserID = strings.TrimSpace(val)
			}
			if rowUserID == userIDStr {
				foundRow = i + 1
				break
			}
		}
	}

	rowData := []interface{}{
		fmt.Sprintf("%d", stats.UserID),
		stats.FirstName,
		stats.LastName,
		fmt.Sprintf("%d", stats.TotalScore),
		fmt.Sprintf("%d", stats.DragonScore),
		fmt.Sprintf("%d", stats.FlappyScore),
		fmt.Sprintf("%d", stats.CrosswordScore),
		fmt.Sprintf("%d", stats.WordleScore),
		stats.Rank,
		stats.LastUpdated,
	}

	if foundRow > 0 {
		// Обновляем существующую строку
		range_ := fmt.Sprintf("%s!A%d:J%d", sheetName, foundRow, foundRow)
		valueRange := &sheets.ValueRange{
			Values: [][]interface{}{rowData},
		}

		_, err = service.Spreadsheets.Values.Update(
			spreadsheetID,
			range_,
			valueRange,
		).ValueInputOption("USER_ENTERED").Do()

		if err != nil {
			return fmt.Errorf("ошибка обновления: %w", err)
		}
	} else {
		// Создаем новую строку
		valueRange := &sheets.ValueRange{
			Values: [][]interface{}{rowData},
		}

		readRange = fmt.Sprintf("%s!A:Z", sheetName)
		_, err = service.Spreadsheets.Values.Append(
			spreadsheetID,
			readRange,
			valueRange,
		).ValueInputOption("USER_ENTERED").Do()

		if err != nil {
			return fmt.Errorf("ошибка добавления: %w", err)
		}
	}

	log.Printf("Обновлен счет для user_id=%d (%s %s), игра=%s, счет=%d, звание=%s",
		stats.UserID, stats.FirstName, stats.LastName, gameType, score, stats.Rank)
	return nil
}

