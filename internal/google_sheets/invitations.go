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
	Name       string
	TelegramID string
	UserID     string
	IsSent     bool
}

// NormalizeTelegramUsername приводит Telegram username/ссылку к каноничному виду (lowercase, без @ и t.me/)
func NormalizeTelegramUsername(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	s = strings.ToLower(s)
	s = strings.TrimPrefix(s, "https://t.me/")
	s = strings.TrimPrefix(s, "http://t.me/")
	s = strings.TrimPrefix(s, "t.me/")
	s = strings.TrimPrefix(s, "@")
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// FindInvitationByTelegramUsername ищет приглашение по Telegram username (столбец B на листе приглашений).
func FindInvitationByTelegramUsername(ctx context.Context, username string) (*InvitationInfo, error) {
	u := NormalizeTelegramUsername(username)
	if u == "" {
		return nil, nil
	}

	invitations, err := GetInvitationsList(ctx)
	if err != nil {
		return nil, err
	}

	for _, inv := range invitations {
		if NormalizeTelegramUsername(inv.TelegramID) == u {
			// копия в heap
			c := inv
			return &c, nil
		}
	}

	return nil, nil
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

		name := cellToString(row[0])
		if name == "" {
			continue
		}

		telegramID := ""
		if len(row) > 1 {
			telegramID = NormalizeTelegramID(cellToString(row[1]))
		}

		status := ""
		if len(row) > 2 {
			status = cellToString(row[2])
		}

		userID := ""
		if len(row) > 3 {
			userID = cellToString(row[3])
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
func invitationDataStartRow(values [][]interface{}) int {
	if len(values) == 0 {
		return 0
	}

	firstRow := values[0]
	if len(firstRow) == 0 {
		return 0
	}

	firstCell := strings.ToLower(cellToString(firstRow[0]))
	if strings.Contains(firstCell, "имя") || strings.Contains(firstCell, "name") || strings.Contains(firstCell, "гость") {
		return 1
	}

	return 0
}

func findInvitationRow(values [][]interface{}, guestName string, username *string) (int, string) {
	startRow := invitationDataStartRow(values)
	normalizedGuestName := strings.ToLower(strings.TrimSpace(guestName))

	if normalizedGuestName != "" {
		for i := startRow; i < len(values); i++ {
			row := values[i]
			if len(row) == 0 {
				continue
			}

			if strings.ToLower(cellToString(row[0])) == normalizedGuestName {
				return i + 1, "name"
			}
		}
	}

	if username != nil {
		normalizedUsername := NormalizeTelegramUsername(*username)
		if normalizedUsername != "" {
			for i := startRow; i < len(values); i++ {
				row := values[i]
				if len(row) < 2 {
					continue
				}

				if NormalizeTelegramUsername(cellToString(row[1])) == normalizedUsername {
					return i + 1, "username"
				}
			}
		}
	}

	return -1, ""
}

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

	foundRow, foundBy := findInvitationRow(resp.Values, guestName, username)
	if foundRow <= 0 {
		if username != nil && NormalizeTelegramUsername(*username) != "" {
			return fmt.Errorf("гость %s / @%s не найден в таблице приглашений", guestName, NormalizeTelegramUsername(*username))
		}
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

	log.Printf("Обновлены username и user_id для %s в строке %d (поиск: %s)", guestName, foundRow, foundBy)
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
			name := strings.TrimSpace(strings.ToLower(cellToString(row[0])))
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
