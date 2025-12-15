package google_sheets

import (
	"context"
	"fmt"
	"strings"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

// ListConfirmedGuests возвращает список подтвержденных гостей для исправления имени/фамилии
func ListConfirmedGuests(ctx context.Context) ([]map[string]interface{}, error) {
	guests, err := GetAllGuestsFromSheets(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(guests))
	for i, guest := range guests {
		fullName := fmt.Sprintf("%s %s", guest.FirstName, guest.LastName)
		result = append(result, map[string]interface{}{
			"row":       i + 2, // +2 потому что первая строка - заголовок, индексация с 1
			"full_name": strings.TrimSpace(fullName),
		})
	}

	return result, nil
}

// SwapGuestNameOrder меняет местами имя и фамилию гостя в указанной строке
func SwapGuestNameOrder(ctx context.Context, row int) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsSheetName

	// Читаем текущие значения
	readRange := fmt.Sprintf("%s!A:B", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения значений: %w", err)
	}

	if row < 1 || row > len(resp.Values) {
		return fmt.Errorf("неверный номер строки: %d", row)
	}

	rowIndex := row - 1 // Индексация с 0
	currentRow := resp.Values[rowIndex]

	if len(currentRow) < 2 {
		return fmt.Errorf("недостаточно данных в строке %d", row)
	}

	firstName := ""
	lastName := ""

	if val, ok := currentRow[0].(string); ok {
		firstName = strings.TrimSpace(val)
	}
	if len(currentRow) > 1 {
		if val, ok := currentRow[1].(string); ok {
			lastName = strings.TrimSpace(val)
		}
	}

	// Если в столбце A полное имя, а в B пусто - парсим
	if lastName == "" && firstName != "" {
		parts := strings.SplitN(firstName, " ", 2)
		if len(parts) == 2 {
			firstName = parts[0]
			lastName = parts[1]
		}
	}

	// Меняем местами
	newFirstName := lastName
	newLastName := firstName

	// Обновляем в таблице
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{
			{newFirstName, newLastName},
		},
	}

	range_ := fmt.Sprintf("%s!A%d:B%d", sheetName, row, row)
	_, err = service.Spreadsheets.Values.Update(
		spreadsheetID,
		range_,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("ошибка обновления: %w", err)
	}

	return nil
}

