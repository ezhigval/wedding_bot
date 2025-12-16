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

// AddWordleWord добавляет новое слово в конец списка Wordle
func AddWordleWord(ctx context.Context, word string) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Wordle"

	if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
		return err
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{strings.ToUpper(strings.TrimSpace(word))}},
	}

	readRange := fmt.Sprintf("%s!A:A", sheetName)
	_, err = service.Spreadsheets.Values.Append(spreadsheetID, readRange, valueRange).
		ValueInputOption("USER_ENTERED").
		Do()
	if err != nil {
		return fmt.Errorf("ошибка добавления слова: %w", err)
	}

	return nil
}

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

	words := []string{}
	for i := 1; i < len(resp.Values); i++ {
		if len(resp.Values[i]) > 0 {
			if val, ok := resp.Values[i][0].(string); ok && strings.TrimSpace(val) != "" {
				words = append(words, strings.TrimSpace(strings.ToUpper(val)))
			}
		}
	}
	if len(words) == 0 {
		return fmt.Errorf("нет слов для переключения")
	}

	currentIndex := 0
	if val, err := getConfigValue(ctx, "WORDLE_INDEX"); err == nil && val != "" {
		if idx, err := strconv.Atoi(val); err == nil && idx >= 0 {
			currentIndex = idx % len(words)
		}
	}

	nextIndex := (currentIndex + 1) % len(words)

	// Сбрасываем прогресс всех пользователей
	if err := resetWordleProgress(ctx); err != nil {
		log.Printf("Ошибка сброса прогресса Wordle: %v", err)
	}

	// Сохраняем новый индекс
	_ = upsertConfigEntries(ctx, map[string]string{
		"WORDLE_INDEX":       fmt.Sprintf("%d", nextIndex),
		"WORDLE_LAST_SWITCH": time.Now().Format("2006-01-02"),
		"WORDLE_LAST_WORD":   words[nextIndex],
	})

	log.Printf("Слово Wordle переключено с индекса %d на %d", currentIndex, nextIndex)
	return nil
}

func resetWordleProgress(ctx context.Context) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID

	// Чистим прогресс
	clearReq := &sheets.ClearValuesRequest{}
	if _, err := service.Spreadsheets.Values.Clear(spreadsheetID, "Wordle_Прогресс!A:Z", clearReq).Do(); err != nil {
		return err
	}
	if _, err := service.Spreadsheets.Values.Clear(spreadsheetID, "Wordle_Состояние!A:Z", clearReq).Do(); err != nil {
		return err
	}

	// Восстанавливаем заголовки
	_ = UpdateSheetValues(spreadsheetID, "Wordle_Прогресс", "A1", [][]interface{}{
		{"user_id", "отгаданные_слова"},
	})
	_ = UpdateSheetValues(spreadsheetID, "Wordle_Состояние", "A1", [][]interface{}{
		{"user_id", "current_word", "attempts", "current_guess", "last_word_date"},
	})

	return nil
}
