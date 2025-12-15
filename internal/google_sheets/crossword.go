package google_sheets

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

// CrosswordWord представляет слово для кроссворда
type CrosswordWord struct {
	Word        string
	Description string
}

// CrosswordProgress представляет прогресс пользователя в кроссворде
type CrosswordProgress struct {
	GuessedWords  []string
	CellLetters   map[string]string
	StartDate     string
	WrongAttempts []string
}

// GetCrosswordWords получает слова для кроссворда из листа 'Кроссворд'
func GetCrosswordWords(ctx context.Context, crosswordIndex int) ([]CrosswordWord, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Кроссворд"

	if err := EnsureRequiredSheets(ctx); err != nil {
		return nil, err
	}

	readRange := fmt.Sprintf("%s!A:Z", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	// Вычисляем индексы столбцов для данного кроссворда
	wordCol := crosswordIndex * 2
	descCol := crosswordIndex*2 + 1

	// Пропускаем заголовок (если есть)
	startRow := 0
	if len(resp.Values) > 0 {
		firstRow := resp.Values[0]
		if len(firstRow) > 0 {
			if val, ok := firstRow[0].(string); ok {
				firstCell := strings.ToLower(val)
				if firstCell == "слово" {
					startRow = 1
				}
			}
		}
	}

	var words []CrosswordWord
	for i := startRow; i < len(resp.Values); i++ {
		row := resp.Values[i]
		if len(row) > wordCol && len(row) > descCol {
			word := ""
			if val, ok := row[wordCol].(string); ok {
				word = strings.TrimSpace(strings.ToUpper(val))
			}

			description := ""
			if val, ok := row[descCol].(string); ok {
				description = strings.TrimSpace(val)
			}

			if word != "" && description != "" {
				words = append(words, CrosswordWord{
					Word:        word,
					Description: description,
				})
			}
		}
	}

	// Если слов нет для данного кроссворда и это первый кроссворд, добавляем дефолтные
	if len(words) == 0 && crosswordIndex == 0 {
		defaultWords := []CrosswordWord{
			{"СВАДЬБА", "Главное событие дня"},
			{"ТАМАДА", "Ведущий праздника"},
			{"ФАТА", "Головной убор невесты"},
			{"БУКЕТ", "Цветы в руках невесты"},
			{"КОЛЬЦО", "Символ брака"},
			{"ТОРТ", "Сладкое угощение"},
			{"ТОСТ", "Поздравление гостей"},
			{"ТАНЕЦ", "Развлечение на празднике"},
		}

		// Сохраняем дефолтные слова в таблицу
		defaultData := make([][]interface{}, len(defaultWords))
		for i, w := range defaultWords {
			defaultData[i] = []interface{}{w.Word, w.Description}
		}

		if err := AppendSheetValues(spreadsheetID, sheetName, defaultData); err != nil {
			return defaultWords, nil // Возвращаем дефолтные слова даже если не удалось сохранить
		}

		return defaultWords, nil
	}

	return words, nil
}

// GetCrosswordProgress получает прогресс пользователя в кроссворде
func GetCrosswordProgress(ctx context.Context, userID int, crosswordIndex int) (*CrosswordProgress, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Кроссворд_Прогресс"

	if err := EnsureRequiredSheets(ctx); err != nil {
		return nil, err
	}

	readRange := fmt.Sprintf("%s!A:D", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	userIDStr := fmt.Sprintf("%d", userID)
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
				progress := &CrosswordProgress{}

				startDate := ""
				if len(row) > 3 {
					if val, ok := row[3].(string); ok {
						startDate = strings.TrimSpace(val)
					}
				}
				progress.StartDate = startDate

				if len(row) > 2 {
					if val, ok := row[2].(string); ok {
						guessedWordsJSON := strings.TrimSpace(val)
						if guessedWordsJSON != "" {
							var guessedWordsMap map[string]interface{}
							if err := json.Unmarshal([]byte(guessedWordsJSON), &guessedWordsMap); err == nil {
								crosswordKey := strconv.Itoa(crosswordIndex)
								if words, ok := guessedWordsMap[crosswordKey].([]interface{}); ok {
									for _, w := range words {
										if word, ok := w.(string); ok {
											progress.GuessedWords = append(progress.GuessedWords, strings.TrimSpace(strings.ToUpper(word)))
										}
									}
								}

								// Получаем сохраненные буквы в клетках
								cellLettersKey := fmt.Sprintf("%s_cells", crosswordKey)
								if cells, ok := guessedWordsMap[cellLettersKey].(map[string]interface{}); ok {
									progress.CellLetters = make(map[string]string)
									for k, v := range cells {
										if val, ok := v.(string); ok {
											progress.CellLetters[k] = val
										}
									}
								}

								// Получаем историю неправильных попыток
								wrongAttemptsKey := fmt.Sprintf("%s_wrong_attempts", crosswordKey)
								if attempts, ok := guessedWordsMap[wrongAttemptsKey].([]interface{}); ok {
									for _, a := range attempts {
										if attempt, ok := a.(string); ok {
											progress.WrongAttempts = append(progress.WrongAttempts, attempt)
										}
									}
								}
							}
						}
					}
				}

				return progress, nil
			}
		}
	}

	return &CrosswordProgress{}, nil
}

// SaveCrosswordProgress сохраняет прогресс пользователя в кроссворде
func SaveCrosswordProgress(ctx context.Context, userID int, guessedWords []string, crosswordIndex int, cellLetters map[string]string, wrongAttempts []string, startDate string) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Кроссворд_Прогресс"

	if err := EnsureRequiredSheets(ctx); err != nil {
		return err
	}

	readRange := fmt.Sprintf("%s!A:D", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения значений: %w", err)
	}

	userIDStr := fmt.Sprintf("%d", userID)
	foundRow := -1
	existingGuessedWordsJSON := make(map[string]interface{})
	existingCrosswordIndex := crosswordIndex
	existingStartDate := ""

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

				if len(row) > 1 {
					if val, ok := row[1].(string); ok {
						if idx, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
							existingCrosswordIndex = idx
						}
					}
				}

				if len(row) > 2 {
					if val, ok := row[2].(string); ok {
						jsonStr := strings.TrimSpace(val)
						if jsonStr != "" {
							json.Unmarshal([]byte(jsonStr), &existingGuessedWordsJSON)
						}
					}
				}

				if len(row) > 3 {
					if val, ok := row[3].(string); ok {
						existingStartDate = strings.TrimSpace(val)
					}
				}

				break
			}
		}
	}

	// Обновляем прогресс для данного кроссворда
	crosswordKey := strconv.Itoa(crosswordIndex)
	existingGuessedWordsJSON[crosswordKey] = guessedWords

	// Добавляем текущие буквы в клетках
	if cellLetters != nil {
		cellLettersKey := fmt.Sprintf("%s_cells", crosswordKey)
		existingGuessedWordsJSON[cellLettersKey] = cellLetters
	}

	// Добавляем историю неправильных попыток
	if wrongAttempts != nil {
		wrongAttemptsKey := fmt.Sprintf("%s_wrong_attempts", crosswordKey)
		existingGuessedWordsJSON[wrongAttemptsKey] = wrongAttempts
	}

	guessedWordsJSONStr, err := json.Marshal(existingGuessedWordsJSON)
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	// Определяем дату начала кроссворда
	if startDate == "" {
		if existingStartDate != "" {
			startDate = existingStartDate
		} else {
			startDate = time.Now().Format("2006-01-02")
		}
	}

	if foundRow > 0 {
		// Обновляем существующую строку
		// Обновляем индекс только если он больше текущего
		if crosswordIndex > existingCrosswordIndex {
			valueRange := &sheets.ValueRange{
				Values: [][]interface{}{{crosswordIndex}},
			}
			range_ := fmt.Sprintf("%s!B%d", sheetName, foundRow)
			_, err = service.Spreadsheets.Values.Update(
				spreadsheetID,
				range_,
				valueRange,
			).ValueInputOption("USER_ENTERED").Do()
			if err != nil {
				return fmt.Errorf("ошибка обновления индекса: %w", err)
			}
		}

		// Обновляем JSON и дату
		rowData := []interface{}{
			string(guessedWordsJSONStr),
			startDate,
		}

		valueRange := &sheets.ValueRange{
			Values: [][]interface{}{rowData},
		}

		range_ := fmt.Sprintf("%s!C%d:D%d", sheetName, foundRow, foundRow)
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
		rowData := []interface{}{
			userIDStr,
			crosswordIndex,
			string(guessedWordsJSONStr),
			startDate,
		}

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

	return nil
}

// GetCrosswordState получает состояние кроссворда для пользователя
func GetCrosswordState(ctx context.Context, userID int) (int, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return 0, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Кроссворд_Прогресс"

	if err := EnsureRequiredSheets(ctx); err != nil {
		return 0, err
	}

	readRange := fmt.Sprintf("%s!A:B", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return 0, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	userIDStr := fmt.Sprintf("%d", userID)
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
				if len(row) > 1 {
					if val, ok := row[1].(string); ok {
						if idx, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
							return idx, nil
						}
					}
				}
				return 0, nil
			}
		}
	}

	return 0, nil
}

// SetCrosswordIndex устанавливает индекс текущего кроссворда для пользователя
func SetCrosswordIndex(ctx context.Context, userID int, crosswordIndex int) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Кроссворд_Прогресс"

	if err := EnsureRequiredSheets(ctx); err != nil {
		return err
	}

	readRange := fmt.Sprintf("%s!A:B", sheetName)
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

	if foundRow > 0 {
		// Обновляем существующую строку
		valueRange := &sheets.ValueRange{
			Values: [][]interface{}{{crosswordIndex}},
		}

		range_ := fmt.Sprintf("%s!B%d", sheetName, foundRow)
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
		rowData := []interface{}{
			userIDStr,
			crosswordIndex,
			"{}",
			time.Now().Format("2006-01-02"),
		}

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

	return nil
}

