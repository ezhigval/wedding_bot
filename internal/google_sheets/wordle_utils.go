package google_sheets

import (
	"context"
	"fmt"
	"log"
	"strings"

	"wedding-bot/internal/config"
)

// SwitchWordleWordForAll переключает слово Wordle для всех пользователей
func SwitchWordleWordForAll(ctx context.Context) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Wordle"

	// Получаем все слова
	readRange := fmt.Sprintf("%s!A:A", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения слов: %w", err)
	}

	if len(resp.Values) <= 1 {
		return fmt.Errorf("нет слов для переключения")
	}

	// Находим текущее активное слово (последнее непустое)
	currentIndex := -1
	for i := len(resp.Values) - 1; i > 0; i-- {
		row := resp.Values[i]
		if len(row) > 0 {
			if word, ok := row[0].(string); ok && strings.TrimSpace(word) != "" {
				currentIndex = i
				break
			}
		}
	}

	if currentIndex < 0 {
		return fmt.Errorf("не найдено активное слово")
	}

	// Переключаем на следующее слово (если есть)
	nextIndex := currentIndex + 1
	if nextIndex >= len(resp.Values) {
		// Если следующего слова нет, берем первое (после заголовка)
		nextIndex = 1
	}

	// Обновляем прогресс всех пользователей - сбрасываем для нового слова
	// TODO: Реализовать сброс прогресса пользователей

	log.Printf("Слово Wordle переключено с индекса %d на %d", currentIndex, nextIndex)
	return nil
}

