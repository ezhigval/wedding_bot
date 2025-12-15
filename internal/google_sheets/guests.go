package google_sheets

import (
	"context"
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/sheets/v4"

	"wedding-bot/internal/config"
)

// AddGuestToSheets добавляет гостя в Google Sheets
func AddGuestToSheets(ctx context.Context, firstName, lastName string, age *int, category, side *string, userID *int) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsSheetName

	// Получаем все значения
	readRange := fmt.Sprintf("%s!A:F", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения значений: %w", err)
	}

	fullName := strings.TrimSpace(fmt.Sprintf("%s %s", firstName, lastName))
	values := resp.Values

	// Ищем существующую строку
	foundRow := -1
	for i, row := range values {
		if len(row) > 0 {
			existingName := ""
			if val, ok := row[0].(string); ok {
				existingName = strings.TrimSpace(val)
			}
			if strings.EqualFold(existingName, fullName) {
				foundRow = i + 1 // +1 потому что индексация с 1
				break
			}
		}
	}

	if foundRow > 0 {
		// Обновляем существующую строку
		updates := []*sheets.ValueRange{
			{
				Range:  fmt.Sprintf("%s!C%d", sheetName, foundRow),
				Values: [][]interface{}{{"ДА"}},
			},
		}

		if category != nil {
			updates = append(updates, &sheets.ValueRange{
				Range:  fmt.Sprintf("%s!D%d", sheetName, foundRow),
				Values: [][]interface{}{{*category}},
			})
		}

		if side != nil {
			updates = append(updates, &sheets.ValueRange{
				Range:  fmt.Sprintf("%s!E%d", sheetName, foundRow),
				Values: [][]interface{}{{*side}},
			})
		}

		if userID != nil {
			updates = append(updates, &sheets.ValueRange{
				Range:  fmt.Sprintf("%s!F%d", sheetName, foundRow),
				Values: [][]interface{}{{fmt.Sprintf("%d", *userID)}},
			})
		}

		batchUpdate := &sheets.BatchUpdateValuesRequest{
			ValueInputOption: "USER_ENTERED",
			Data:             updates,
		}

		_, err = service.Spreadsheets.Values.BatchUpdate(spreadsheetID, batchUpdate).Do()
		if err != nil {
			return fmt.Errorf("ошибка обновления строки: %w", err)
		}

		log.Printf("Гость %s найден в строке %d, обновлено подтверждение", fullName, foundRow)
		return nil
	}

	// Ищем первую пустую строку
	emptyRow := len(values) + 1
	for i, row := range values {
		if len(row) == 0 || (len(row) > 0 && strings.TrimSpace(fmt.Sprintf("%v", row[0])) == "") {
			emptyRow = i + 1
			break
		}
	}

	// Формируем данные для вставки
	rowData := []interface{}{fullName}
	if age != nil {
		rowData = append(rowData, fmt.Sprintf("%d", *age))
	} else {
		rowData = append(rowData, "")
	}
	rowData = append(rowData, "ДА") // Подтверждение
	if category != nil {
		rowData = append(rowData, *category)
	} else {
		rowData = append(rowData, "")
	}
	if side != nil {
		rowData = append(rowData, *side)
	} else {
		rowData = append(rowData, "")
	}
	if userID != nil {
		rowData = append(rowData, fmt.Sprintf("%d", *userID))
	} else {
		rowData = append(rowData, "")
	}

	// Вставляем данные
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{rowData},
	}

	range_ := fmt.Sprintf("%s!A%d:F%d", sheetName, emptyRow, emptyRow)
	_, err = service.Spreadsheets.Values.Update(
		spreadsheetID,
		range_,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("ошибка добавления гостя: %w", err)
	}

	log.Printf("Гость %s добавлен в Google Sheets в строку %d", fullName, emptyRow)
	return nil
}

// CancelInvitation отменяет приглашение - обновляет столбец C на "НЕТ"
func CancelInvitation(ctx context.Context, firstName, lastName string) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsSheetName

	fullName := strings.TrimSpace(fmt.Sprintf("%s %s", firstName, lastName))

	// Получаем все значения
	readRange := fmt.Sprintf("%s!A:C", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения значений: %w", err)
	}

	// Ищем строку
	foundRow := -1
	for i, row := range resp.Values {
		if len(row) > 0 {
			existingName := ""
			if val, ok := row[0].(string); ok {
				existingName = strings.TrimSpace(val)
			}
			if strings.EqualFold(existingName, fullName) {
				foundRow = i + 1
				break
			}
		}
	}

	if foundRow <= 0 {
		return fmt.Errorf("гость %s не найден", fullName)
	}

	// Обновляем столбец C
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{"НЕТ"}},
	}

	range_ := fmt.Sprintf("%s!C%d", sheetName, foundRow)
	_, err = service.Spreadsheets.Values.Update(
		spreadsheetID,
		range_,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("ошибка обновления: %w", err)
	}

	log.Printf("Приглашение для %s отменено (строка %d)", fullName, foundRow)
	return nil
}

// CheckGuestRegistration проверяет, зарегистрирован ли гость
func CheckGuestRegistration(ctx context.Context, userID int) (bool, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return false, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsSheetName

	// Получаем все значения столбца F (user_id)
	readRange := fmt.Sprintf("%s!F:F", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return false, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	userIDStr := fmt.Sprintf("%d", userID)
	for _, row := range resp.Values {
		if len(row) > 0 {
			if val, ok := row[0].(string); ok {
				if strings.TrimSpace(val) == userIDStr {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// GuestInfo представляет информацию о госте
type GuestInfo struct {
	FirstName string
	LastName  string
	Username  string
	UserID    string
	Category  string
	Side      string
}

// GetAllGuestsFromSheets получает список всех зарегистрированных гостей
func GetAllGuestsFromSheets(ctx context.Context) ([]GuestInfo, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsSheetName

	readRange := fmt.Sprintf("%s!A:F", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	var guests []GuestInfo
	for _, row := range resp.Values {
		if len(row) == 0 {
			continue
		}

		fullName := ""
		if val, ok := row[0].(string); ok {
			fullName = strings.TrimSpace(val)
		}
		if fullName == "" {
			continue
		}

		confirmation := ""
		if len(row) > 2 {
			if val, ok := row[2].(string); ok {
				confirmation = strings.TrimSpace(val)
			}
		}

		// Берем только тех, кто подтвердил участие
		if strings.ToUpper(confirmation) != "ДА" {
			continue
		}

		// Парсим имя и фамилию
		nameParts := strings.SplitN(fullName, " ", 2)
		firstName := nameParts[0]
		lastName := ""
		if len(nameParts) > 1 {
			lastName = nameParts[1]
		}

		category := ""
		if len(row) > 3 {
			if val, ok := row[3].(string); ok {
				category = strings.TrimSpace(val)
			}
		}

		side := ""
		if len(row) > 4 {
			if val, ok := row[4].(string); ok {
				side = strings.TrimSpace(val)
			}
		}

		userID := ""
		if len(row) > 5 {
			if val, ok := row[5].(string); ok {
				userID = strings.TrimSpace(val)
			}
		}

		guests = append(guests, GuestInfo{
			FirstName: firstName,
			LastName:  lastName,
			Username:  "",
			UserID:    userID,
			Category:  category,
			Side:      side,
		})
	}

	return guests, nil
}

// GetGuestsCountFromSheets получает количество зарегистрированных гостей
func GetGuestsCountFromSheets(ctx context.Context) (int, error) {
	guests, err := GetAllGuestsFromSheets(ctx)
	if err != nil {
		return 0, err
	}
	return len(guests), nil
}

// CancelGuestRegistrationByUserID отменяет регистрацию гостя по user_id
func CancelGuestRegistrationByUserID(ctx context.Context, userID int) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsSheetName

	userIDStr := fmt.Sprintf("%d", userID)

	// Получаем все значения
	readRange := fmt.Sprintf("%s!A:F", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения значений: %w", err)
	}

	// Ищем строку по user_id
	foundRow := -1
	for i, row := range resp.Values {
		if len(row) > 5 {
			if val, ok := row[5].(string); ok {
				if strings.TrimSpace(val) == userIDStr {
					foundRow = i + 1
					break
				}
			}
		}
	}

	if foundRow <= 0 {
		return fmt.Errorf("гость с user_id=%d не найден", userID)
	}

	// Обновляем столбец C на "НЕТ"
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{"НЕТ"}},
	}

	range_ := fmt.Sprintf("%s!C%d", sheetName, foundRow)
	_, err = service.Spreadsheets.Values.Update(
		spreadsheetID,
		range_,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("ошибка обновления: %w", err)
	}

	log.Printf("Регистрация гостя с user_id=%d отменена (строка %d)", userID, foundRow)
	return nil
}

// DeleteGuestFromSheets удаляет гостя из таблицы по user_id
func DeleteGuestFromSheets(ctx context.Context, userID int) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsID

	userIDStr := fmt.Sprintf("%d", userID)

	// Получаем все значения
	readRange := fmt.Sprintf("%s!A:F", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения значений: %w", err)
	}

	// Ищем строку по user_id
	foundRow := -1
	for i, row := range resp.Values {
		if len(row) > 5 {
			if val, ok := row[5].(string); ok {
				if strings.TrimSpace(val) == userIDStr {
					foundRow = i + 1
					break
				}
			}
		}
	}

	if foundRow <= 0 {
		return fmt.Errorf("гость с user_id=%d не найден", userID)
	}

	// Удаляем строку
	spreadsheet, err := GetSpreadsheet(spreadsheetID)
	if err != nil {
		return err
	}

	var sheetID int64
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == sheetName {
			sheetID = sheet.Properties.SheetId
			break
		}
	}

	if sheetID == 0 {
		return fmt.Errorf("лист %s не найден", sheetName)
	}

	deleteDimensionRequest := &sheets.DeleteDimensionRequest{
		Range: &sheets.DimensionRange{
			SheetId:    sheetID,
			Dimension:  "ROWS",
			StartIndex: int64(foundRow - 1),
			EndIndex:   int64(foundRow),
		},
	}

	batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteDimension: deleteDimensionRequest,
			},
		},
	}

	_, err = service.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdateRequest).Do()
	if err != nil {
		return fmt.Errorf("ошибка удаления строки: %w", err)
	}

	log.Printf("Гость с user_id=%d удален из Google Sheets (строка %d)", userID, foundRow)
	return nil
}

// FindGuestByName находит гостя по имени и фамилии
func FindGuestByName(ctx context.Context, firstName, lastName string) (*GuestInfo, error) {
	guests, err := GetAllGuestsFromSheets(ctx)
	if err != nil {
		return nil, err
	}

	fullName := strings.TrimSpace(strings.ToLower(fmt.Sprintf("%s %s", firstName, lastName)))
	for _, guest := range guests {
		guestFullName := strings.TrimSpace(strings.ToLower(fmt.Sprintf("%s %s", guest.FirstName, guest.LastName)))
		if guestFullName == fullName {
			return &guest, nil
		}
	}

	return nil, nil
}

// NormalizeTelegramID нормализует Telegram ID
func NormalizeTelegramID(telegramID string) string {
	// Убираем пробелы и кавычки
	id := strings.TrimSpace(telegramID)
	id = strings.Trim(id, `"'`)
	return id
}

