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
	clean := strings.TrimSpace(value)
	if clean == "" {
		return false
	}

	switch strings.ToLower(clean) {
	case "да", "true", "1", "yes", "y", "истина", "ok":
		return true
	case "нет", "false", "0", "no", "n", "ложь":
		return false
	}

	upper := strings.ToUpper(clean)
	return upper == "TRUE" || upper == "ДА" || upper == "✓" || upper == "✔"
}

func buildGuestFullName(firstName, lastName string) string {
	return strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(firstName),
		strings.TrimSpace(lastName),
	}, " "))
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}

	clean := strings.TrimSpace(*value)
	return &clean
}

// CompanionGuest описывает дополнительного гостя,
// которого основной пользователь добавляет к своей регистрации.
type CompanionGuest struct {
	FirstName string
	LastName  string
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

	fullName := buildGuestFullName(firstName, lastName)
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

	// Сначала ищем по идентификатору (column F): user_id или username.
	foundRow := -1
	foundBy := ""
	if authUserID > 0 || authUsername != "" {
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

	// Если по идентификатору не нашли, пробуем по имени.
	if foundRow <= 0 && fullName != "" {
		for i, row := range values {
			if len(row) == 0 {
				continue
			}

			existingName := cellToString(row[0])
			if strings.EqualFold(existingName, fullName) {
				foundRow = i + 1
				foundBy = "name"
				break
			}
		}
	}

	if foundRow > 0 {
		// Обновляем существующую строку
		updates := []*sheets.ValueRange{
			{
				Range:  fmt.Sprintf("%s!C%d", sheetName, foundRow),
				Values: [][]interface{}{{true}},
			},
		}

		existingName := ""
		if foundRow-1 >= 0 && foundRow-1 < len(values) && len(values[foundRow-1]) > 0 {
			existingName = cellToString(values[foundRow-1][0])
		}
		if fullName != "" && !strings.EqualFold(existingName, fullName) {
			updates = append(updates, &sheets.ValueRange{
				Range:  fmt.Sprintf("%s!A%d", sheetName, foundRow),
				Values: [][]interface{}{{fullName}},
			})
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
	rowData = append(rowData, true) // Подтверждение (checkbox)
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

func resolveCompanionMetadata(ctx context.Context, category, side *string, userID *int, username *string) (*string, *string, error) {
	effectiveCategory := cloneStringPointer(category)
	effectiveSide := cloneStringPointer(side)

	authUserID := 0
	if userID != nil {
		authUserID = *userID
	}

	authUsername := ""
	if username != nil {
		authUsername = *username
	}

	if effectiveCategory != nil && effectiveSide != nil {
		return effectiveCategory, effectiveSide, nil
	}

	existingGuest, err := FindGuestByIdentifier(ctx, authUserID, authUsername)
	if err != nil {
		return nil, nil, err
	}
	if existingGuest == nil {
		return effectiveCategory, effectiveSide, nil
	}

	if effectiveCategory == nil && strings.TrimSpace(existingGuest.Category) != "" {
		categoryCopy := strings.TrimSpace(existingGuest.Category)
		effectiveCategory = &categoryCopy
	}

	if effectiveSide == nil && strings.TrimSpace(existingGuest.Side) != "" {
		sideCopy := strings.TrimSpace(existingGuest.Side)
		effectiveSide = &sideCopy
	}

	return effectiveCategory, effectiveSide, nil
}

// AddGuestGroupToSheets сохраняет основного гостя и дополнительных гостей одной регистрационной группой.
func AddGuestGroupToSheets(ctx context.Context, firstName, lastName string, age *int, category, side *string, userID *int, username *string, companions []CompanionGuest) error {
	hasPrimaryGuest := buildGuestFullName(firstName, lastName) != ""

	authUserID := 0
	if userID != nil {
		authUserID = *userID
	}

	authUsername := ""
	if username != nil {
		authUsername = *username
	}

	existingOwner, err := FindGuestByIdentifier(ctx, authUserID, authUsername)
	if err != nil {
		return err
	}

	if !hasPrimaryGuest && len(companions) > 0 && existingOwner == nil {
		return fmt.Errorf("основной гость не найден, сначала завершите регистрацию для себя")
	}

	effectiveCategory, effectiveSide, err := resolveCompanionMetadata(ctx, category, side, userID, username)
	if err != nil {
		return err
	}

	if hasPrimaryGuest {
		if err := AddGuestToSheets(ctx, firstName, lastName, age, category, side, userID, username); err != nil {
			return err
		}
	}

	for _, companion := range companions {
		if err := AddAdditionalGuestToSheets(ctx, companion.FirstName, companion.LastName, effectiveCategory, effectiveSide, userID, username); err != nil {
			return err
		}
	}

	return nil
}

// AddAdditionalGuestToSheets добавляет или обновляет дополнительного гостя, сохраняя связь с основным владельцем регистрации.
func AddAdditionalGuestToSheets(ctx context.Context, firstName, lastName string, category, side *string, userID *int, username *string) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsSheetName

	fullName := buildGuestFullName(firstName, lastName)
	if fullName == "" {
		return nil
	}

	authUserID := 0
	if userID != nil {
		authUserID = *userID
	}

	authUsername := ""
	if username != nil {
		authUsername = *username
	}

	authIdentifier := buildAuthIdentifierForStorage(authUserID, authUsername)
	if authIdentifier == "" {
		return fmt.Errorf("идентификатор владельца регистрации не передан")
	}

	readRange := fmt.Sprintf("%s!A:F", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения значений: %w", err)
	}

	values := resp.Values
	foundRow := -1

	for i, row := range values {
		if len(row) == 0 {
			continue
		}

		existingName := cellToString(row[0])
		if !strings.EqualFold(existingName, fullName) {
			continue
		}

		existingIdentifier := ""
		if len(row) > 5 {
			existingIdentifier = cellToString(row[5])
		}

		if existingIdentifier == "" || guestIdentifierMatches(existingIdentifier, authUserID, authUsername) {
			foundRow = i + 1
			break
		}
	}

	if foundRow > 0 {
		updates := []*sheets.ValueRange{
			{
				Range:  fmt.Sprintf("%s!A%d", sheetName, foundRow),
				Values: [][]interface{}{{fullName}},
			},
			{
				Range:  fmt.Sprintf("%s!C%d", sheetName, foundRow),
				Values: [][]interface{}{{true}},
			},
		}

		if category != nil {
			updates = append(updates, &sheets.ValueRange{
				Range:  fmt.Sprintf("%s!D%d", sheetName, foundRow),
				Values: [][]interface{}{{strings.TrimSpace(*category)}},
			})
		}

		if side != nil {
			updates = append(updates, &sheets.ValueRange{
				Range:  fmt.Sprintf("%s!E%d", sheetName, foundRow),
				Values: [][]interface{}{{strings.TrimSpace(*side)}},
			})
		}

		existingIdentifier := ""
		if foundRow-1 >= 0 && foundRow-1 < len(values) && len(values[foundRow-1]) > 5 {
			existingIdentifier = cellToString(values[foundRow-1][5])
		}

		if authIdentifier != "" && existingIdentifier != authIdentifier {
			updates = append(updates, &sheets.ValueRange{
				Range:  fmt.Sprintf("%s!F%d", sheetName, foundRow),
				Values: [][]interface{}{{authIdentifier}},
			})
		}

		batchUpdate := &sheets.BatchUpdateValuesRequest{
			ValueInputOption: "USER_ENTERED",
			Data:             updates,
		}

		if _, err := service.Spreadsheets.Values.BatchUpdate(spreadsheetID, batchUpdate).Do(); err != nil {
			return fmt.Errorf("ошибка обновления дополнительного гостя: %w", err)
		}

		log.Printf("Дополнительный гость %s обновлён в строке %d", fullName, foundRow)
		return nil
	}

	emptyRow := len(values) + 1
	for i, row := range values {
		if len(row) == 0 || cellToString(row[0]) == "" {
			emptyRow = i + 1
			break
		}
	}

	rowData := []interface{}{fullName, "", true}
	if category != nil {
		rowData = append(rowData, strings.TrimSpace(*category))
	} else {
		rowData = append(rowData, "")
	}
	if side != nil {
		rowData = append(rowData, strings.TrimSpace(*side))
	} else {
		rowData = append(rowData, "")
	}
	rowData = append(rowData, authIdentifier)

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{rowData},
	}

	range_ := fmt.Sprintf("%s!A%d:F%d", sheetName, emptyRow, emptyRow)
	if _, err := service.Spreadsheets.Values.Update(
		spreadsheetID,
		range_,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do(); err != nil {
		return fmt.Errorf("ошибка добавления дополнительного гостя: %w", err)
	}

	log.Printf("Дополнительный гость %s добавлен в Google Sheets в строку %d", fullName, emptyRow)
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

	foundMatch := false
	for i, row := range resp.Values {
		if len(row) <= 5 {
			continue
		}

		identifier := cellToString(row[5])
		if !guestIdentifierMatches(identifier, userID, username) {
			continue
		}
		foundMatch = true

		confirmation := ""
		if len(row) > 2 {
			confirmation = cellToString(row[2])
		}

		registered := isConfirmedValue(confirmation)
		log.Printf("Найдена строка гостя по идентификатору (row=%d, registered=%v, id=%s)", i+1, registered, identifier)
		if registered {
			return true, nil
		}
		// Не выходим сразу при false: ниже может быть другая строка этого же пользователя с подтверждением.
	}

	if foundMatch {
		log.Printf("Найдены строки по идентификатору, но без подтверждения участия (user_id=%d, username=%s)", userID, NormalizeTelegramUsername(username))
		return false, nil
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

	// Ищем все строки по user_id/username в колонке F.
	foundRows := make([]int, 0, 2)
	for i, row := range resp.Values {
		if len(row) <= 5 {
			continue
		}
		if guestIdentifierMatches(cellToString(row[5]), userID, username) {
			foundRows = append(foundRows, i+1)
		}
	}

	if len(foundRows) == 0 {
		if userID > 0 {
			return fmt.Errorf("гость с user_id=%d не найден", userID)
		}
		return fmt.Errorf("гость с username=%s не найден", NormalizeTelegramUsername(username))
	}

	updates := make([]*sheets.ValueRange, 0, len(foundRows))
	for _, row := range foundRows {
		updates = append(updates, &sheets.ValueRange{
			Range:  fmt.Sprintf("%s!C%d", sheetName, row),
			Values: [][]interface{}{{false}},
		})
	}

	batchUpdate := &sheets.BatchUpdateValuesRequest{
		ValueInputOption: "USER_ENTERED",
		Data:             updates,
	}

	if _, err := service.Spreadsheets.Values.BatchUpdate(spreadsheetID, batchUpdate).Do(); err != nil {
		return fmt.Errorf("ошибка обновления: %w", err)
	}

	if userID > 0 {
		log.Printf("Регистрация гостя с user_id=%d отменена (строк: %d)", userID, len(foundRows))
	} else {
		log.Printf("Регистрация гостя с username=%s отменена (строк: %d)", NormalizeTelegramUsername(username), len(foundRows))
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
