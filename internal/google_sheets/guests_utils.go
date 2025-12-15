package google_sheets

import (
	"context"
	"fmt"
	"strings"
	"time"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

// UpdateGuestUserID обновляет user_id гостя в указанной строке
func UpdateGuestUserID(ctx context.Context, row int, userID int) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsSheetName

	// Обновляем столбец F (user_id)
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{fmt.Sprintf("%d", userID)}},
	}

	range_ := fmt.Sprintf("%s!F%d", sheetName, row)
	_, err = service.Spreadsheets.Values.Update(
		spreadsheetID,
		range_,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("ошибка обновления user_id: %w", err)
	}

	return nil
}

// FindDuplicateGuests находит дубликаты гостей в таблице
func FindDuplicateGuests(ctx context.Context) (map[string][]GuestInfo, error) {
	guests, err := GetAllGuestsFromSheets(ctx)
	if err != nil {
		return nil, err
	}

	// Группируем по полному имени
	nameMap := make(map[string][]GuestInfo)
	for _, guest := range guests {
		fullName := strings.TrimSpace(strings.ToLower(fmt.Sprintf("%s %s", guest.FirstName, guest.LastName)))
		if fullName != "" {
			nameMap[fullName] = append(nameMap[fullName], guest)
		}
	}

	// Находим дубликаты (имена, которые встречаются более одного раза)
	duplicates := make(map[string][]GuestInfo)
	for name, guestList := range nameMap {
		if len(guestList) > 1 {
			duplicates[name] = guestList
		}
	}

	return duplicates, nil
}

