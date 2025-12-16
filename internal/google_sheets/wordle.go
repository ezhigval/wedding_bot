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

// WordleState представляет состояние игры Wordle
type WordleState struct {
	CurrentWord  string
	Attempts     [][]map[string]interface{}
	CurrentGuess string
	LastWordDate string
}

// GetWordleWord получает последнее (актуальное) слово из вкладки "Wordle"
func GetWordleWord(ctx context.Context) (string, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return "", err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Wordle"

	if err := EnsureRequiredSheets(ctx); err != nil {
		return "", err
	}

	readRange := fmt.Sprintf("%s!A:A", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return "", fmt.Errorf("ошибка чтения значений: %w", err)
	}

	if len(resp.Values) == 0 {
		return "", nil
	}

	// Пропускаем заголовок (первую строку)
	var words []string
	for i := 1; i < len(resp.Values); i++ {
		row := resp.Values[i]
		if len(row) > 0 {
			if val, ok := row[0].(string); ok {
				word := strings.TrimSpace(strings.ToUpper(val))
				if word != "" {
					words = append(words, word)
				}
			}
		}
	}

	// Возвращаем последнее слово (актуальное)
	if len(words) > 0 {
		return words[len(words)-1], nil
	}

	return "", nil
}

// GetWordleGuessedWords получает список отгаданных слов пользователя
func GetWordleGuessedWords(ctx context.Context, userID int) ([]string, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Wordle_Прогресс"

	if err := EnsureRequiredSheets(ctx); err != nil {
		return nil, err
	}

	readRange := fmt.Sprintf("%s!A:B", sheetName)
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
				// Нашли пользователя
				if len(row) > 1 {
					if val, ok := row[1].(string); ok {
						wordsStr := strings.TrimSpace(val)
						if wordsStr != "" {
							words := strings.Split(wordsStr, ",")
							var result []string
							for _, w := range words {
								w = strings.TrimSpace(strings.ToUpper(w))
								if w != "" {
									result = append(result, w)
								}
							}
							return result, nil
						}
					}
				}
				return []string{}, nil
			}
		}
	}

	return []string{}, nil
}

// SaveWordleProgress сохраняет прогресс пользователя в Wordle
func SaveWordleProgress(ctx context.Context, userID int, guessedWords []string) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Wordle_Прогресс"

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

	wordsStr := strings.Join(guessedWords, ",")

	if foundRow > 0 {
		// Обновляем существующую строку
		valueRange := &sheets.ValueRange{
			Values: [][]interface{}{{wordsStr}},
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
		valueRange := &sheets.ValueRange{
			Values: [][]interface{}{{userIDStr, wordsStr}},
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

// GetWordleState получает состояние игры Wordle для пользователя
func GetWordleState(ctx context.Context, userID int) (*WordleState, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Wordle_Состояние"

	if err := EnsureRequiredSheets(ctx); err != nil {
		return nil, err
	}

	readRange := fmt.Sprintf("%s!A:E", sheetName)
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
				state := &WordleState{}

				if len(row) > 1 {
					if val, ok := row[1].(string); ok {
						state.CurrentWord = strings.TrimSpace(val)
					}
				}

				if len(row) > 2 {
					if val, ok := row[2].(string); ok {
						attemptsJSON := strings.TrimSpace(val)
						if attemptsJSON != "" {
							var attempts [][]map[string]interface{}
							if err := json.Unmarshal([]byte(attemptsJSON), &attempts); err == nil {
								state.Attempts = attempts
							}
						}
					}
				}

				if len(row) > 3 {
					if val, ok := row[3].(string); ok {
						state.CurrentGuess = strings.TrimSpace(val)
					}
				}

				if len(row) > 4 {
					if val, ok := row[4].(string); ok {
						state.LastWordDate = strings.TrimSpace(val)
					}
				}

				return state, nil
			}
		}
	}

	return nil, nil
}

// SaveWordleState сохраняет состояние игры Wordle для пользователя
func SaveWordleState(ctx context.Context, userID int, currentWord string, attempts [][]map[string]interface{}, lastWordDate string, currentGuess string) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Wordle_Состояние"

	if err := EnsureRequiredSheets(ctx); err != nil {
		return err
	}

	readRange := fmt.Sprintf("%s!A:E", sheetName)
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

	attemptsJSON, err := json.Marshal(attempts)
	if err != nil {
		return fmt.Errorf("ошибка сериализации attempts: %w", err)
	}

	if foundRow > 0 {
		// Обновляем существующую строку
		rowData := []interface{}{
			currentWord,
			string(attemptsJSON),
			currentGuess,
			lastWordDate,
		}

		valueRange := &sheets.ValueRange{
			Values: [][]interface{}{rowData},
		}

		range_ := fmt.Sprintf("%s!B%d:E%d", sheetName, foundRow, foundRow)
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
			currentWord,
			string(attemptsJSON),
			currentGuess,
			lastWordDate,
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

// GetWordleWordForUser получает актуальное слово для пользователя с учетом автоматической смены раз в сутки
func GetWordleWordForUser(ctx context.Context, userID int) (string, error) {
	// Получаем все слова из таблицы Wordle
	words, err := getAllWordleWords(ctx)
	if err != nil {
		return "", err
	}

	if len(words) == 0 {
		return "", nil
	}

	// Получаем состояние пользователя
	state, err := GetWordleState(ctx, userID)
	if err != nil {
		return "", err
	}

	if state == nil || state.CurrentWord == "" || state.LastWordDate == "" {
		// Первый раз или нет сохраненного состояния - берем слово по глобальному индексу
		currentWord := words[0]
		if idx, err := getConfigValue(ctx, "WORDLE_INDEX"); err == nil && idx != "" {
			if parsed, err := strconv.Atoi(idx); err == nil && parsed >= 0 {
				currentWord = words[parsed%len(words)]
			}
		}
		today := time.Now().Format("2006-01-02")
		if err := SaveWordleState(ctx, userID, currentWord, [][]map[string]interface{}{}, today, ""); err != nil {
			return "", err
		}
		return currentWord, nil
	}

	// Проверяем, прошло ли больше суток
	lastDate := state.LastWordDate
	today := time.Now().Format("2006-01-02")

	if lastDate != today {
		// Нужно сменить слово на следующее
		currentWord := state.CurrentWord
		currentIndex := -1
		for i, w := range words {
			if w == currentWord {
				currentIndex = i
				break
			}
		}

		var nextWord string
		if currentIndex >= 0 {
			nextIndex := (currentIndex + 1) % len(words) // Циклически переходим к следующему
			nextWord = words[nextIndex]
		} else {
			// Текущее слово не найдено в списке, берем первое
			nextWord = words[0]
		}

		// Сохраняем новое слово и сбрасываем попытки
		if err := SaveWordleState(ctx, userID, nextWord, [][]map[string]interface{}{}, today, ""); err != nil {
			return "", err
		}
		return nextWord, nil
	}

	// Слово не менялось, возвращаем текущее
	return state.CurrentWord, nil
}

// getAllWordleWords получает все слова из таблицы Wordle
func getAllWordleWords(ctx context.Context) ([]string, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Wordle"

	readRange := fmt.Sprintf("%s!A:A", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	var words []string
	for i := 1; i < len(resp.Values); i++ { // Пропускаем заголовок
		row := resp.Values[i]
		if len(row) > 0 {
			if val, ok := row[0].(string); ok {
				word := strings.TrimSpace(strings.ToUpper(val))
				if word != "" {
					words = append(words, word)
				}
			}
		}
	}

	return words, nil
}
