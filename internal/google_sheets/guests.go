package google_sheets

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"google.golang.org/api/sheets/v4"

	"wedding-bot/internal/config"
)

func cellToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(val, 'f', -1, 64))
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", val))
	}
}

func normalizeGuestIdentifier(value string) string {
	clean := strings.TrimSpace(strings.Trim(value, `"'`))
	if clean == "" {
		return ""
	}

	if n, ok := normalizeNumericIdentifier(clean); ok {
		return n
	}

	return NormalizeTelegramUsername(clean)
}

func normalizeNumericIdentifier(value string) (string, bool) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return "", false
	}

	if intValue, err := strconv.ParseInt(clean, 10, 64); err == nil && intValue > 0 {
		return strconv.FormatInt(intValue, 10), true
	}

	if floatValue, err := strconv.ParseFloat(clean, 64); err == nil {
		intValue := int64(floatValue)
		if floatValue == float64(intValue) && intValue > 0 {
			return strconv.FormatInt(intValue, 10), true
		}
	}

	return "", false
}

func guestIdentifierMatches(sheetValue string, userID int, username string) bool {
	normalizedSheet := normalizeGuestIdentifier(sheetValue)
	if normalizedSheet == "" {
		return false
	}

	if userID > 0 && normalizedSheet == strconv.Itoa(userID) {
		return true
	}

	normalizedUsername := NormalizeTelegramUsername(username)
	return normalizedUsername != "" && normalizedSheet == normalizedUsername
}

func buildAuthIdentifierForStorage(userID int, username string) string {
	if userID > 0 {
		return strconv.Itoa(userID)
	}
	normalizedUsername := NormalizeTelegramUsername(username)
	if normalizedUsername != "" {
		return normalizedUsername
	}
	return ""
}

func isConfirmedValue(value string) bool {
	return strings.ToUpper(strings.TrimSpace(value)) == "ДА"
}

// AddGuestToSheets добавляет гостя в Google Sheets
func AddGuestToSheets(ctx context.Context, firstName, lastName string, age *int, category, side *string, userID *int, username *string) error {
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
	authUserID := 0
	if userID != nil {
		authUserID = *userID
	}
	authUsername := ""
	if username != nil {
		authUsername = *username
	}
	authIdentifier := buildAuthIdentifierForStorage(authUserID, authUsername)
	values := resp.Values

	// Ищем существующую строку
	foundRow := -1
	foundBy := ""
	for i, row := range values {
		if len(row) > 0 {
			existingName := cellToString(row[0])
			if fullName != "" && strings.EqualFold(existingName, fullName) {
				foundRow = i + 1 // +1 потому что индексация с 1
				foundBy = "name"
				break
			}
		}
	}

	// Дополнительно ищем по идентификатору (column F): user_id или username.
	if foundRow <= 0 && (authUserID > 0 || authUsername != "") {
		for i, row := range values {
			if len(row) <= 5 {
				continue
			}
			if guestIdentifierMatches(cellToString(row[5]), authUserID, authUsername) {
				foundRow = i + 1
				foundBy = "identifier"
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

		existingIdentifier := ""
		if foundRow-1 >= 0 && foundRow-1 < len(values) && len(values[foundRow-1]) > 5 {
			existingIdentifier = cellToString(values[foundRow-1][5])
		}

		// Обновляем идентификатор в колонке F:
		// 1) всегда заполняем пустой идентификатор,
		// 2) если пришел user_id, а в ячейке username — заменяем на user_id.
		identifierToWrite := ""
		shouldWriteIdentifier := false
		if authUserID > 0 {
			identifierToWrite = strconv.Itoa(authUserID)
			if existingIdentifier == "" {
				shouldWriteIdentifier = true
			} else if _, isNumeric := normalizeNumericIdentifier(existingIdentifier); !isNumeric {
				shouldWriteIdentifier = true
			}
		} else if authIdentifier != "" && existingIdentifier == "" {
			identifierToWrite = authIdentifier
			shouldWriteIdentifier = true
		}

		if shouldWriteIdentifier {
			updates = append(updates, &sheets.ValueRange{
				Range:  fmt.Sprintf("%s!F%d", sheetName, foundRow),
				Values: [][]interface{}{{identifierToWrite}},
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

		if fullName == "" {
			fullName = "без имени"
		}
		log.Printf("Гость %s найден в строке %d (по %s), обновлено подтверждение", fullName, foundRow, foundBy)
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
	rowData = append(rowData, authIdentifier)

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

// CheckGuestRegistration проверяет, зарегистрирован ли гость
func CheckGuestRegistration(ctx context.Context, userID int) (bool, error) {
	return CheckGuestRegistrationByIdentifier(ctx, userID, "")
}

// CheckGuestRegistrationByUsername проверяет регистрацию по username (column F).
func CheckGuestRegistrationByUsername(ctx context.Context, username string) (bool, error) {
	return CheckGuestRegistrationByIdentifier(ctx, 0, username)
}

// CheckGuestRegistrationByIdentifier проверяет регистрацию по user_id и/или username.
func CheckGuestRegistrationByIdentifier(ctx context.Context, userID int, username string) (bool, error) {
	if userID <= 0 && NormalizeTelegramUsername(username) == "" {
		return false, nil
	}

	service, err := GetGoogleSheetsClient()
	if err != nil {
		return false, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsSheetName

	// Column A: full name, C: confirmation, F: auth identifier (user_id or username)
	readRange := fmt.Sprintf("%s!A:F", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return false, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	for i, row := range resp.Values {
		if len(row) <= 5 {
			continue
		}

		identifier := cellToString(row[5])
		if !guestIdentifierMatches(identifier, userID, username) {
			continue
		}

		confirmation := ""
		if len(row) > 2 {
			confirmation = cellToString(row[2])
		}

		registered := isConfirmedValue(confirmation)
		log.Printf("Найдена строка гостя по идентификатору (row=%d, registered=%v, id=%s)", i+1, registered, identifier)
		return registered, nil
	}

	if userID > 0 {
		log.Printf("user_id %d не найден среди зарегистрированных гостей", userID)
	} else {
		log.Printf("username %s не найден среди зарегистрированных гостей", NormalizeTelegramUsername(username))
	}

	return false, nil
}

// FindGuestByIdentifier находит гостя по user_id/username в колонке F независимо от подтверждения.
func FindGuestByIdentifier(ctx context.Context, userID int, username string) (*GuestInfo, error) {
	if userID <= 0 && NormalizeTelegramUsername(username) == "" {
		return nil, nil
	}

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

	for _, row := range resp.Values {
		if len(row) <= 5 {
			continue
		}

		identifier := cellToString(row[5])
		if !guestIdentifierMatches(identifier, userID, username) {
			continue
		}

		fullName := ""
		if len(row) > 0 {
			fullName = cellToString(row[0])
		}
		if fullName == "" {
			continue
		}

		nameParts := strings.SplitN(fullName, " ", 2)
		firstName := nameParts[0]
		lastName := ""
		if len(nameParts) > 1 {
			lastName = nameParts[1]
		}

		category := ""
		if len(row) > 3 {
			category = cellToString(row[3])
		}

		side := ""
		if len(row) > 4 {
			side = cellToString(row[4])
		}

		guest := &GuestInfo{
			FirstName: firstName,
			LastName:  lastName,
			UserID:    identifier,
			Category:  category,
			Side:      side,
		}

		if _, ok := normalizeNumericIdentifier(identifier); !ok {
			guest.Username = NormalizeTelegramUsername(identifier)
		}

		return guest, nil
	}

	return nil, nil
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

		fullName := cellToString(row[0])
		if fullName == "" {
			continue
		}

		confirmation := ""
		if len(row) > 2 {
			confirmation = cellToString(row[2])
		}

		// Берем только тех, кто подтвердил участие
		if !isConfirmedValue(confirmation) {
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
			category = cellToString(row[3])
		}

		side := ""
		if len(row) > 4 {
			side = cellToString(row[4])
		}

		userID := ""
		if len(row) > 5 {
			userID = cellToString(row[5])
		}

		username := ""
		if _, isNumeric := normalizeNumericIdentifier(userID); !isNumeric {
			username = NormalizeTelegramUsername(userID)
		}

		guests = append(guests, GuestInfo{
			FirstName: firstName,
			LastName:  lastName,
			Username:  username,
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
	return CancelGuestRegistrationByIdentifier(ctx, userID, "")
}

// CancelGuestRegistrationByUsername отменяет регистрацию по username (column F).
func CancelGuestRegistrationByUsername(ctx context.Context, username string) error {
	return CancelGuestRegistrationByIdentifier(ctx, 0, username)
}

// CancelGuestRegistrationByIdentifier отменяет регистрацию по user_id и/или username.
func CancelGuestRegistrationByIdentifier(ctx context.Context, userID int, username string) error {
	if userID <= 0 && NormalizeTelegramUsername(username) == "" {
		return fmt.Errorf("идентификатор гостя не передан")
	}

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

	// Ищем строку по user_id/username в колонке F
	foundRow := -1
	for i, row := range resp.Values {
		if len(row) <= 5 {
			continue
		}
		if guestIdentifierMatches(cellToString(row[5]), userID, username) {
			foundRow = i + 1
			break
		}
	}

	if foundRow <= 0 {
		if userID > 0 {
			return fmt.Errorf("гость с user_id=%d не найден", userID)
		}
		return fmt.Errorf("гость с username=%s не найден", NormalizeTelegramUsername(username))
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

	if userID > 0 {
		log.Printf("Регистрация гостя с user_id=%d отменена (строка %d)", userID, foundRow)
	} else {
		log.Printf("Регистрация гостя с username=%s отменена (строка %d)", NormalizeTelegramUsername(username), foundRow)
	}
	return nil
}

// DeleteGuestFromSheets удаляет гостя из таблицы по user_id
func DeleteGuestFromSheets(ctx context.Context, userID int) error {
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

// NormalizeTelegramID нормализует Telegram ID
func NormalizeTelegramID(telegramID string) string {
	// Убираем пробелы и кавычки
	id := strings.TrimSpace(telegramID)
	id = strings.Trim(id, `"'`)
	return id
}
