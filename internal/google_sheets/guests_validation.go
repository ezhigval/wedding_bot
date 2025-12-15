package google_sheets

import (
	"context"
	"fmt"
	"log"
	"strings"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

// ValidateGuestSheetStructure проверяет структуру листа "Список гостей"
// и создает заголовки, если их нет
func ValidateGuestSheetStructure(ctx context.Context) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsSheetName

	// Проверяем существование листа
	spreadsheet, err := GetSpreadsheet(spreadsheetID)
	if err != nil {
		return fmt.Errorf("ошибка получения таблицы: %w", err)
	}

	sheetExists := false
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == sheetName {
			sheetExists = true
			break
		}
	}

	if !sheetExists {
		return fmt.Errorf("лист '%s' не найден в таблице", sheetName)
	}

	// Читаем первую строку (заголовки)
	readRange := fmt.Sprintf("%s!A1:Z1", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения заголовков: %w", err)
	}

	var headers []string
	if len(resp.Values) > 0 {
		for _, val := range resp.Values[0] {
			if str, ok := val.(string); ok {
				headers = append(headers, strings.TrimSpace(str))
			} else {
				headers = append(headers, "")
			}
		}
	}

	// Ожидаемые заголовки (минимальный набор)
	expectedHeaders := map[int]string{
		0: "Имя",           // A
		1: "Фамилия",       // B (опционально, может быть объединено с A)
		2: "Подтверждение", // C
		3: "Категория",     // D
		4: "Сторона",       // E
		5: "user_id",       // F
		6: "Стол",          // G
	}

	// Проверяем наличие обязательных заголовков
	missingHeaders := []string{}
	needsUpdate := false

	// Проверяем, есть ли хотя бы заголовок "Имя" в колонке A
	if len(headers) == 0 || headers[0] == "" {
		needsUpdate = true
		missingHeaders = append(missingHeaders, "Имя (колонка A)")
	}

	// Проверяем наличие других важных заголовков
	for col, expectedHeader := range expectedHeaders {
		if col >= len(headers) || strings.TrimSpace(headers[col]) == "" {
			// Если это не обязательный заголовок (B - Фамилия может быть объединено с A), пропускаем
			if col == 1 {
				continue
			}
			needsUpdate = true
			missingHeaders = append(missingHeaders, fmt.Sprintf("%s (колонка %s)", expectedHeader, getColumnLetter(col+1)))
		} else {
			// Проверяем, что заголовок правильный (не строго, но логично)
			currentHeader := strings.TrimSpace(strings.ToLower(headers[col]))
			expectedLower := strings.ToLower(expectedHeader)
			if !strings.Contains(currentHeader, expectedLower) && !strings.Contains(expectedLower, currentHeader) {
				// Заголовок есть, но не совпадает - это не критично, просто логируем
				log.Printf("⚠️ В колонке %s заголовок '%s' не совпадает с ожидаемым '%s'", getColumnLetter(col+1), headers[col], expectedHeader)
			}
		}
	}

	if needsUpdate {
		log.Printf("⚠️ Обнаружены отсутствующие заголовки в листе '%s': %v", sheetName, missingHeaders)
		log.Printf("📝 Создаю/обновляю заголовки...")

		// Создаем массив заголовков
		newHeaders := make([]interface{}, 7)
		newHeaders[0] = "Имя"
		newHeaders[1] = "Фамилия"
		newHeaders[2] = "Подтверждение"
		newHeaders[3] = "Категория"
		newHeaders[4] = "Сторона"
		newHeaders[5] = "user_id"
		newHeaders[6] = "Стол"

		// Обновляем только если первая строка пустая или неполная
		if len(headers) == 0 || headers[0] == "" {
			valueRange := &sheets.ValueRange{
				Values: [][]interface{}{newHeaders},
			}

			_, err = service.Spreadsheets.Values.Update(
				spreadsheetID,
				fmt.Sprintf("%s!A1:G1", sheetName),
				valueRange,
			).ValueInputOption("USER_ENTERED").Do()

			if err != nil {
				return fmt.Errorf("ошибка обновления заголовков: %w", err)
			}

			log.Printf("✅ Заголовки созданы/обновлены в листе '%s'", sheetName)
		} else {
			log.Printf("ℹ️ Заголовки уже существуют, но некоторые могут отсутствовать. Проверьте структуру таблицы вручную.")
		}
	} else {
		log.Printf("✅ Структура листа '%s' проверена: все необходимые заголовки присутствуют", sheetName)
	}

	// Дополнительная проверка: валидация данных (опционально)
	// Проверяем, что есть хотя бы одна строка данных
	readDataRange := fmt.Sprintf("%s!A2:F100", sheetName)
	dataResp, err := service.Spreadsheets.Values.Get(spreadsheetID, readDataRange).Do()
	if err == nil && len(dataResp.Values) > 0 {
		log.Printf("✅ Найдено %d строк данных в листе '%s'", len(dataResp.Values), sheetName)
	} else {
		log.Printf("ℹ️ В листе '%s' пока нет данных (это нормально для нового проекта)", sheetName)
	}

	return nil
}

// getColumnLetter преобразует номер колонки (1-based) в букву (A, B, C, ...)
func getColumnLetter(colNum int) string {
	if colNum <= 0 {
		return ""
	}

	result := ""
	colNum-- // Переводим в 0-based

	for colNum >= 0 {
		result = string(rune('A'+colNum%26)) + result
		colNum = colNum/26 - 1
	}

	return result
}

