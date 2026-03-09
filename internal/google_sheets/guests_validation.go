package google_sheets

import (
	"context"
	"fmt"
	"log"
	"strings"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

type guestHeaderSpec struct {
	displayName string
	aliases     []string
	required    bool
}

func normalizeHeaderForCompare(value string) string {
	clean := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		" ", "",
		"_", "",
		"-", "",
		"—", "",
		"\t", "",
		".", "",
		",", "",
		":", "",
		";", "",
		"(", "",
		")", "",
		"/", "",
		"\\", "",
		"'", "",
		"\"", "",
	)
	return replacer.Replace(clean)
}

func headerMatchesAnyAlias(header string, aliases []string) bool {
	normalizedHeader := normalizeHeaderForCompare(header)
	if normalizedHeader == "" {
		return false
	}

	for _, alias := range aliases {
		normalizedAlias := normalizeHeaderForCompare(alias)
		if normalizedAlias == "" {
			continue
		}
		if strings.Contains(normalizedHeader, normalizedAlias) || strings.Contains(normalizedAlias, normalizedHeader) {
			return true
		}
	}

	return false
}

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

	// Ожидаемые заголовки и совместимые варианты.
	// Базовая рабочая схема:
	// A = имя и фамилия, C = присутствие (checkbox true/false), F = user_id/username
	expectedHeaders := map[int]guestHeaderSpec{
		0: {
			displayName: "Имя и фамилия",
			aliases:     []string{"Имя и фамилия", "ФИО", "Имя", "Full name"},
			required:    true,
		}, // A
		1: {
			displayName: "Возраст",
			aliases:     []string{"Возраст", "Фамилия", "Age"},
			required:    false,
		}, // B
		2: {
			displayName: "Присутствие на свадьбе",
			aliases:     []string{"Присутствие на свадьбе", "Присутствие", "Подтверждение", "Attendance"},
			required:    true,
		}, // C
		3: {
			displayName: "Родство",
			aliases:     []string{"Родство", "Категория", "Category"},
			required:    false,
		}, // D
		4: {
			displayName: "Сторона",
			aliases:     []string{"Сторона", "Side"},
			required:    false,
		}, // E
		5: {
			displayName: "Телеграм Юсер_айди",
			aliases: []string{
				"Телеграм Юсер_айди",
				"Telegram user id",
				"Telegram ID",
				"user_id",
				"userid",
				"username",
			},
			required: true,
		}, // F
		6: {
			displayName: "Стол",
			aliases:     []string{"Стол", "Table"},
			required:    false,
		}, // G
	}

	// Проверяем наличие заголовков
	missingHeaders := []string{}
	needsUpdate := false

	for col, spec := range expectedHeaders {
		if col >= len(headers) || strings.TrimSpace(headers[col]) == "" {
			if !spec.required {
				continue
			}
			needsUpdate = true
			missingHeaders = append(missingHeaders, fmt.Sprintf("%s (колонка %s)", spec.displayName, getColumnLetter(col+1)))
			continue
		}

		if !headerMatchesAnyAlias(headers[col], spec.aliases) {
			// Заголовок есть, но отличается от знакомых вариантов — не критично, только лог.
			log.Printf("⚠️ В колонке %s заголовок '%s' не распознан (ожидалось что-то из: %s)", getColumnLetter(col+1), headers[col], strings.Join(spec.aliases, ", "))
		}
	}

	if needsUpdate {
		log.Printf("⚠️ Обнаружены отсутствующие заголовки в листе '%s': %v", sheetName, missingHeaders)
		log.Printf("📝 Создаю/обновляю заголовки...")

		// Создаем массив заголовков
		newHeaders := make([]interface{}, 7)
		newHeaders[0] = "Имя и фамилия"
		newHeaders[1] = "Возраст"
		newHeaders[2] = "Присутствие на свадьбе"
		newHeaders[3] = "Родство"
		newHeaders[4] = "Сторона"
		newHeaders[5] = "Телеграм Юсер_айди"
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
