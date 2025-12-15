package google_sheets

import (
	"context"
	"fmt"
	"log"
	"strings"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

// InvitationInfo представляет информацию о приглашении
type InvitationInfo struct {
	Name      string
	TelegramID string
	UserID    string
	IsSent    bool
}

// GetInvitationsList получает список приглашений из Google Sheets
func GetInvitationsList(ctx context.Context) ([]InvitationInfo, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsInvitationsSheetName

	// Проверяем существование листа
	if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
		log.Printf("Ошибка проверки листа %s: %v", sheetName, err)
		return nil, err
	}

	readRange := fmt.Sprintf("%s!A:D", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	var invitations []InvitationInfo
	startRow := 0

	// Проверяем, есть ли заголовок
	if len(resp.Values) > 0 {
		firstRow := resp.Values[0]
		if len(firstRow) > 0 {
			firstCell := strings.ToLower(fmt.Sprintf("%v", firstRow[0]))
			if strings.Contains(firstCell, "имя") || strings.Contains(firstCell, "name") || strings.Contains(firstCell, "гость") {
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

		name := ""
		if val, ok := row[0].(string); ok {
			name = strings.TrimSpace(val)
		}
		if name == "" {
			continue
		}

		telegramID := ""
		if len(row) > 1 {
			if val, ok := row[1].(string); ok {
				telegramID = NormalizeTelegramID(val)
			}
		}

		status := ""
		if len(row) > 2 {
			if val, ok := row[2].(string); ok {
				status = strings.TrimSpace(val)
			}
		}

		userID := ""
		if len(row) > 3 {
			if val, ok := row[3].(string); ok {
				userID = strings.TrimSpace(val)
			}
		}

		isSent := strings.ToUpper(status) == "ДА"

		invitations = append(invitations, InvitationInfo{
			Name:       name,
			TelegramID: telegramID,
			UserID:     userID,
			IsSent:     isSent,
		})
	}

	log.Printf("Получено %d приглашений из Google Sheets", len(invitations))
	return invitations, nil
}

// UpdateInvitationUserID обновляет user_id и username в таблице приглашений
func UpdateInvitationUserID(ctx context.Context, guestName string, userID int, username *string) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsInvitationsSheetName

	readRange := fmt.Sprintf("%s!A:D", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения значений: %w", err)
	}

	startRow := 0
	if len(resp.Values) > 0 {
		firstRow := resp.Values[0]
		if len(firstRow) > 0 {
			firstCell := strings.ToLower(fmt.Sprintf("%v", firstRow[0]))
			if strings.Contains(firstCell, "имя") || strings.Contains(firstCell, "name") || strings.Contains(firstCell, "гость") {
				startRow = 1
			}
		}
	}

	// Ищем строку с именем гостя
	foundRow := -1
	for i := startRow; i < len(resp.Values); i++ {
		row := resp.Values[i]
		if len(row) > 0 {
			name := ""
			if val, ok := row[0].(string); ok {
				name = strings.TrimSpace(strings.ToLower(val))
			}
			if name == strings.ToLower(guestName) {
				foundRow = i + 1
				break
			}
		}
	}

	if foundRow <= 0 {
		return fmt.Errorf("гость %s не найден в таблице приглашений", guestName)
	}

	updates := []*sheets.ValueRange{}

	// Обновляем username (столбец B)
	if username != nil {
		usernameClean := strings.TrimPrefix(*username, "@")
		updates = append(updates, &sheets.ValueRange{
			Range:  fmt.Sprintf("%s!B%d", sheetName, foundRow),
			Values: [][]interface{}{{usernameClean}},
		})
	}

	// Обновляем user_id (столбец D)
	updates = append(updates, &sheets.ValueRange{
		Range:  fmt.Sprintf("%s!D%d", sheetName, foundRow),
		Values: [][]interface{}{{fmt.Sprintf("%d", userID)}},
	})

	batchUpdate := &sheets.BatchUpdateValuesRequest{
		ValueInputOption: "USER_ENTERED",
		Data:             updates,
	}

	_, err = service.Spreadsheets.Values.BatchUpdate(spreadsheetID, batchUpdate).Do()
	if err != nil {
		return fmt.Errorf("ошибка обновления: %w", err)
	}

	log.Printf("Обновлены username и user_id для %s в строке %d", guestName, foundRow)
	return nil
}

// MarkInvitationAsSent отмечает приглашение как отправленное
func MarkInvitationAsSent(ctx context.Context, guestName string) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsInvitationsSheetName

	readRange := fmt.Sprintf("%s!A:C", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения значений: %w", err)
	}

	startRow := 0
	if len(resp.Values) > 0 {
		firstRow := resp.Values[0]
		if len(firstRow) > 0 {
			firstCell := strings.ToLower(fmt.Sprintf("%v", firstRow[0]))
			if strings.Contains(firstCell, "имя") || strings.Contains(firstCell, "name") || strings.Contains(firstCell, "гость") {
				startRow = 1
			}
		}
	}

	// Ищем строку с именем гостя
	foundRow := -1
	for i := startRow; i < len(resp.Values); i++ {
		row := resp.Values[i]
		if len(row) > 0 {
			name := ""
			if val, ok := row[0].(string); ok {
				name = strings.TrimSpace(strings.ToLower(val))
			}
			if name == strings.ToLower(guestName) {
				foundRow = i + 1
				break
			}
		}
	}

	if foundRow <= 0 {
		return fmt.Errorf("гость %s не найден в таблице приглашений", guestName)
	}

	// Обновляем столбец C на "ДА"
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{"ДА"}},
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

	log.Printf("Приглашение для %s отмечено как отправленное (строка %d)", guestName, foundRow)
	return nil
}

