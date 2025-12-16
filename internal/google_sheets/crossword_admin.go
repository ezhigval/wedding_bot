package google_sheets

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

// AddCrossword добавляет новый кроссворд в новые столбцы и возвращает его индекс
func AddCrossword(ctx context.Context, words []CrosswordWord) (int, error) {
	spreadsheetID := config.GoogleSheetsID
	sheetName := "Кроссворд"

	if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
		return 0, err
	}

	// Получаем таблицу, чтобы понять количество столбцов
	spreadsheet, err := GetSpreadsheet(spreadsheetID)
	if err != nil {
		return 0, err
	}

	var sheetID int64
	var columnCount int64
	for _, sh := range spreadsheet.Sheets {
		if sh.Properties.Title == sheetName {
			sheetID = sh.Properties.SheetId
			columnCount = sh.Properties.GridProperties.ColumnCount
			break
		}
	}

	if sheetID == 0 {
		return 0, fmt.Errorf("лист %s не найден", sheetName)
	}

	index := int(columnCount / 2) // текущих кроссвордов = столбцов/2

	// Готовим данные (заголовок + слова)
	rows := make([][]interface{}, len(words)+1)
	rows[0] = []interface{}{"слово", "описание"}
	for i, w := range words {
		rows[i+1] = []interface{}{strings.ToUpper(w.Word), w.Description}
	}

	// Стартовый столбец
	colWord := columnIndexToLetter(index*2 + 1)
	rangeA1 := fmt.Sprintf("%s1:%s", colWord, columnIndexToLetter(index*2+2))

	if err := UpdateSheetValues(spreadsheetID, sheetName, rangeA1, rows); err != nil {
		return index, err
	}

	return index, nil
}

// SwitchCrosswordForAll переключает кроссворд на следующий, циклически
func SwitchCrosswordForAll(ctx context.Context) (int, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return 0, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Кроссворд"

	// Получаем количество столбцов
	spreadsheet, err := GetSpreadsheet(spreadsheetID)
	if err != nil {
		return 0, err
	}

	var columnCount int64
	for _, sh := range spreadsheet.Sheets {
		if sh.Properties.Title == sheetName {
			columnCount = sh.Properties.GridProperties.ColumnCount
			break
		}
	}
	total := int(columnCount / 2)
	if total == 0 {
		return 0, fmt.Errorf("нет кроссвордов для переключения")
	}

	current := 0
	if val, err := getConfigValue(ctx, "CROSSWORD_INDEX"); err == nil && val != "" {
		if idx, err := strconv.Atoi(val); err == nil {
			current = idx % total
		}
	}
	next := (current + 1) % total

	// Чистим прогресс
	clearReq := &sheets.ClearValuesRequest{}
	if _, err := service.Spreadsheets.Values.Clear(spreadsheetID, "Кроссворд_Прогресс!A:Z", clearReq).Do(); err != nil {
		return next, err
	}
	// Восстанавливаем заголовки
	_ = UpdateSheetValues(spreadsheetID, "Кроссворд_Прогресс", "A1", [][]interface{}{
		{"user_id", "current_crossword_index", "guessed_words_json", "crossword_start_date"},
	})

	// Сохраняем индекс и дату
	_ = upsertConfigEntries(ctx, map[string]string{
		"CROSSWORD_INDEX":       fmt.Sprintf("%d", next),
		"CROSSWORD_LAST_SWITCH": time.Now().Format("2006-01-02"),
	})

	return next, nil
}

// GetCurrentCrosswordIndex возвращает глобальный индекс кроссворда (default 0)
func GetCurrentCrosswordIndex(ctx context.Context) int {
	if val, err := getConfigValue(ctx, "CROSSWORD_INDEX"); err == nil && val != "" {
		if idx, err := strconv.Atoi(val); err == nil && idx >= 0 {
			return idx
		}
	}
	return 0
}

func columnIndexToLetter(col int) string {
	if col <= 0 {
		return ""
	}
	result := ""
	for col > 0 {
		col--
		result = string(rune('A'+col%26)) + result
		col /= 26
	}
	return result
}
