package google_sheets

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

// AdminInfo представляет информацию об админе
type AdminInfo struct {
	Username  string
	UserID    *int
	APIID     string
	APIHash   string
	Phone     string
	LoginCode string
	RowIndex  int
}

// GetAdminsList получает список админов из Google Sheets
func GetAdminsList(ctx context.Context) ([]AdminInfo, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsAdminsSheetName

	// Проверяем существование листа
	if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
		log.Printf("Ошибка проверки листа %s: %v", sheetName, err)
		return nil, err
	}

	readRange := fmt.Sprintf("%s!A:G", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	var admins []AdminInfo
	startRow := 0

	// Проверяем, есть ли заголовок
	if len(resp.Values) > 0 {
		firstRow := resp.Values[0]
		if len(firstRow) > 0 {
			firstCell := strings.ToLower(fmt.Sprintf("%v", firstRow[0]))
			if strings.Contains(firstCell, "username") || strings.Contains(firstCell, "админ") || strings.Contains(firstCell, "admin") {
				startRow = 1
			}
		}
	}

	// Обрабатываем данные
	for i := startRow; i < len(resp.Values); i++ {
		row := resp.Values[i]
		if len(row) == 0 {
			continue
		}

		username := ""
		if val, ok := row[0].(string); ok {
			username = strings.TrimSpace(val)
		}
		if username == "" {
			continue
		}

		usernameClean := strings.TrimPrefix(strings.ToLower(username), "@")

		admin := AdminInfo{
			Username: usernameClean,
			RowIndex: i + 1,
		}

		// User ID (столбец B)
		if len(row) > 1 {
			if val, ok := row[1].(string); ok {
				val = strings.TrimSpace(val)
				if val != "" {
					if userID, err := strconv.Atoi(val); err == nil {
						admin.UserID = &userID
					}
				}
			}
		}

		// API ID (столбец D)
		if len(row) > 3 {
			if val, ok := row[3].(string); ok {
				admin.APIID = strings.TrimSpace(val)
			}
		}

		// API Hash (столбец E)
		if len(row) > 4 {
			if val, ok := row[4].(string); ok {
				admin.APIHash = strings.TrimSpace(val)
			}
		}

		// Phone (столбец F)
		if len(row) > 5 {
			if val, ok := row[5].(string); ok {
				admin.Phone = strings.TrimSpace(val)
			}
		}

		// Login Code (столбец G)
		if len(row) > 6 {
			if val, ok := row[6].(string); ok {
				admin.LoginCode = strings.TrimSpace(val)
			}
		}

		admins = append(admins, admin)
	}

	log.Printf("Получено %d админов из Google Sheets", len(admins))
	return admins, nil
}

// GetAdminLoginCodeAndClear получает и очищает код авторизации админа
func GetAdminLoginCodeAndClear(ctx context.Context, adminUserID int) (string, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return "", err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsAdminsSheetName

	readRange := fmt.Sprintf("%s!A:G", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return "", fmt.Errorf("ошибка чтения значений: %w", err)
	}

	startRow := 0
	if len(resp.Values) > 0 {
		firstRow := resp.Values[0]
		if len(firstRow) > 0 {
			firstCell := strings.ToLower(fmt.Sprintf("%v", firstRow[0]))
			if strings.Contains(firstCell, "username") || strings.Contains(firstCell, "админ") || strings.Contains(firstCell, "admin") {
				startRow = 1
			}
		}
	}

	// Ищем строку с user_id
	for i := startRow; i < len(resp.Values); i++ {
		row := resp.Values[i]
		if len(row) < 2 {
			continue
		}

		userIDCell := ""
		if val, ok := row[1].(string); ok {
			userIDCell = strings.TrimSpace(val)
		}

		userIDValue, err := strconv.Atoi(userIDCell)
		if err != nil || userIDValue != adminUserID {
			continue
		}

		// Нашли строку текущего админа
		code := ""
		if len(row) > 6 {
			if val, ok := row[6].(string); ok {
				code = strings.TrimSpace(val)
			}
		}

		if code == "" {
			log.Printf("В столбце G для админа %d код не найден (строка %d)", adminUserID, i+1)
			return "", nil
		}

		// Очищаем ячейку с кодом (столбец G = 7)
		valueRange := &sheets.ValueRange{
			Values: [][]interface{}{{""}},
		}

		range_ := fmt.Sprintf("%s!G%d", sheetName, i+1)
		_, err = service.Spreadsheets.Values.Update(
			spreadsheetID,
			range_,
			valueRange,
		).ValueInputOption("USER_ENTERED").Do()

		if err != nil {
			return "", fmt.Errorf("ошибка очистки кода: %w", err)
		}

		log.Printf("Считан и очищен код авторизации для админа %d из строки %d", adminUserID, i+1)
		return code, nil
	}

	log.Printf("Строка для админа %d в листе 'Админ бота' не найдена", adminUserID)
	return "", nil
}

// SaveAdminToSheets сохраняет или обновляет админа в Google Sheets
func SaveAdminToSheets(ctx context.Context, username string, userID int) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsAdminsSheetName

	// Проверяем существование листа
	if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
		return fmt.Errorf("ошибка проверки листа: %w", err)
	}

	usernameLower := strings.TrimPrefix(strings.ToLower(username), "@")

	readRange := fmt.Sprintf("%s!A:B", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения значений: %w", err)
	}

	startRow := 0
	if len(resp.Values) > 0 {
		firstRow := resp.Values[0]
		if len(firstRow) > 0 {
			firstCell := strings.ToLower(fmt.Sprintf("%v", firstRow[0]))
			if strings.Contains(firstCell, "username") || strings.Contains(firstCell, "админ") || strings.Contains(firstCell, "admin") {
				startRow = 1
			}
		}
	}

	// Ищем существующую строку по username
	foundRow := -1
	for i := startRow; i < len(resp.Values); i++ {
		row := resp.Values[i]
		if len(row) > 0 {
			existingUsername := ""
			if val, ok := row[0].(string); ok {
				existingUsername = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(val)), "@")
			}
			if existingUsername == usernameLower {
				foundRow = i + 1
				break
			}
		}
	}

	if foundRow > 0 {
		// Обновляем существующую строку (столбец B = user_id)
		valueRange := &sheets.ValueRange{
			Values: [][]interface{}{{fmt.Sprintf("%d", userID)}},
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

		log.Printf("Админ %s обновлен в строке %d: user_id=%d", username, foundRow, userID)
		return nil
	}

	// Добавляем новую строку
	rowData := []interface{}{
		usernameLower,
		fmt.Sprintf("%d", userID),
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

	log.Printf("Админ %s добавлен в Google Sheets: user_id=%d", username, userID)
	return nil
}

