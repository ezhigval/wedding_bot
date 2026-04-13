package google_sheets

import (
	"context"
	"fmt"
	"log"
	"strings"

	"wedding-bot/internal/config"
)

// SeatingTable представляет стол с гостями
type SeatingTable struct {
	Table  string
	Guests []string
}

func readSeatingFromSheet(ctx context.Context, sheetName string) ([]SeatingTable, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	readRange := fmt.Sprintf("%s!A:Z", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	if len(resp.Values) == 0 {
		return []SeatingTable{}, nil
	}

	headerRow := resp.Values[0]
	if len(headerRow) < 2 {
		return []SeatingTable{}, nil
	}

	var tables []SeatingTable
	for idx := 1; idx < len(headerRow); idx++ {
		tableName := ""
		if val, ok := headerRow[idx].(string); ok {
			tableName = strings.TrimSpace(val)
		}
		if tableName == "" {
			continue
		}

		var guests []string
		for r := 1; r < len(resp.Values); r++ {
			row := resp.Values[r]
			if idx >= len(row) {
				continue
			}

			guestName := ""
			if val, ok := row[idx].(string); ok {
				guestName = strings.TrimSpace(val)
			}
			if guestName != "" {
				guests = append(guests, guestName)
			}
		}

		tables = append(tables, SeatingTable{
			Table:  tableName,
			Guests: guests,
		})
	}

	log.Printf("Прочитана рассадка из листа '%s': %d столов (%d гостей)", sheetName, len(tables), func() int {
		total := 0
		for _, t := range tables {
			total += len(t.Guests)
		}
		return total
	}())

	return tables, nil
}

// GetSeatingFromSheets получает текущую рассадку из листа "Рассадка"
func GetSeatingFromSheets(ctx context.Context) ([]SeatingTable, error) {
	return readSeatingFromSheet(ctx, "Рассадка")
}

// GetPublishedSeatingFromSheets получает опубликованную рассадку из листа "Рассадка_фикс".
func GetPublishedSeatingFromSheets(ctx context.Context) ([]SeatingTable, error) {
	if err := EnsureSheetExists(config.GoogleSheetsID, "Рассадка_фикс"); err != nil {
		return nil, err
	}

	return readSeatingFromSheet(ctx, "Рассадка_фикс")
}

// GuestTableInfo представляет информацию о столе гостя
type GuestTableInfo struct {
	FullName  string
	Table     string
	Neighbors []string
}

func normalizeComparableGuestName(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

// GetGuestTableAndNeighbors находит для гостя по user_id стол и соседей
func GetGuestTableAndNeighbors(ctx context.Context, userID int) (*GuestTableInfo, error) {
	return GetGuestTableAndNeighborsByIdentifier(ctx, userID, "")
}

// GetGuestTableAndNeighborsByIdentifier находит для гостя по user_id и/или username стол и соседей.
func GetGuestTableAndNeighborsByIdentifier(ctx context.Context, userID int, username string) (*GuestTableInfo, error) {
	guest, err := FindGuestByIdentifier(ctx, userID, username)
	if err != nil {
		return nil, err
	}
	if guest == nil {
		log.Printf("Гость с user_id=%d username=%s не найден в 'Список гостей'", userID, NormalizeTelegramUsername(username))
		return nil, nil
	}

	fullName := strings.TrimSpace(strings.Join([]string{guest.FirstName, guest.LastName}, " "))
	if fullName == "" {
		return nil, nil
	}

	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	targetName := normalizeComparableGuestName(fullName)

	// Ищем имя гостя в зафиксированной рассадке ('Рассадка_фикс')
	seatingSheetName := "Рассадка_фикс"
	readRange := fmt.Sprintf("%s!A:Z", seatingSheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения рассадки: %w", err)
	}

	if len(resp.Values) < 2 {
		return nil, nil
	}

	headerRow := resp.Values[0]
	var targetTable string
	var neighbors []string

	for colIdx := 1; colIdx < len(headerRow); colIdx++ {
		tableName := ""
		if val, ok := headerRow[colIdx].(string); ok {
			tableName = strings.TrimSpace(val)
		}

		if tableName == "" {
			continue
		}

		var columnNames []string
		for r := 1; r < len(resp.Values); r++ {
			row := resp.Values[r]
			if colIdx >= len(row) {
				continue
			}

			name := ""
			if val, ok := row[colIdx].(string); ok {
				name = strings.TrimSpace(val)
			}

			if name != "" {
				columnNames = append(columnNames, name)
			}
		}

		// Ищем полное совпадение имени в этом столе
		found := false
		for _, n := range columnNames {
			if normalizeComparableGuestName(n) == targetName {
				targetTable = tableName
				// Соседи - все остальные имена в этом столбце
				for _, neighbor := range columnNames {
					if normalizeComparableGuestName(neighbor) != targetName {
						neighbors = append(neighbors, neighbor)
					}
				}
				found = true
				break
			}
		}

		if found {
			break
		}
	}

	if targetTable == "" {
		log.Printf("Гость '%s' (user_id=%d, username=%s) не найден в зафиксированной рассадке", fullName, userID, NormalizeTelegramUsername(username))
		return nil, nil
	}

	return &GuestTableInfo{
		FullName:  fullName,
		Table:     targetTable,
		Neighbors: neighbors,
	}, nil
}
