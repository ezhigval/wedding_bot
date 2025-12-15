package google_sheets

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"wedding-bot/internal/config"
)

// SeatingTable представляет стол с гостями
type SeatingTable struct {
	Table  string
	Guests []string
}

// GetSeatingFromSheets получает текущую рассадку из листа "Рассадка"
func GetSeatingFromSheets(ctx context.Context) ([]SeatingTable, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Рассадка"

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
	// Заголовки столов начинаются с колонки B (индекс 1)
	for idx := 1; idx < len(headerRow); idx++ {
		tableName := ""
		if val, ok := headerRow[idx].(string); ok {
			tableName = strings.TrimSpace(val)
		}

		if tableName == "" {
			continue
		}

		var guests []string
		// Строки 2..N (индексы 1..len(values)-1)
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

	log.Printf("Прочитана рассадка: %d столов (%d гостей)", len(tables), func() int {
		total := 0
		for _, t := range tables {
			total += len(t.Guests)
		}
		return total
	}())

	return tables, nil
}

// GuestTableInfo представляет информацию о столе гостя
type GuestTableInfo struct {
	FullName string
	Table    string
	Neighbors []string
}

// GetGuestTableAndNeighbors находит для гостя по user_id стол и соседей
func GetGuestTableAndNeighbors(ctx context.Context, userID int) (*GuestTableInfo, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsSheetName

	// 1. Находим полное имя гостя по user_id на листе "Список гостей"
	readRange := fmt.Sprintf("%s!A:F", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	userIDStr := strconv.Itoa(userID)
	fullName := ""

	for i, row := range resp.Values {
		if i == 0 {
			continue // Пропускаем заголовок
		}

		if len(row) > 5 {
			uidCell := ""
			if val, ok := row[5].(string); ok {
				uidCell = strings.TrimSpace(val)
			}

			if uidCell == userIDStr {
				if val, ok := row[0].(string); ok {
					fullName = strings.TrimSpace(val)
				}
				break
			}
		}
	}

	if fullName == "" {
		log.Printf("Гость с user_id=%d не найден в 'Список гостей'", userID)
		return nil, nil
	}

	// 2. Ищем это имя в зафиксированной рассадке ('Рассадка_фикс')
	seatingSheetName := "Рассадка_фикс"
	readRange = fmt.Sprintf("%s!A:Z", seatingSheetName)
	resp, err = service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
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
			if n == fullName {
				targetTable = tableName
				// Соседи - все остальные имена в этом столбце
				for _, neighbor := range columnNames {
					if neighbor != fullName {
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
		log.Printf("Гость '%s' (user_id=%d) не найден в зафиксированной рассадке", fullName, userID)
		return nil, nil
	}

	return &GuestTableInfo{
		FullName:  fullName,
		Table:     targetTable,
		Neighbors: neighbors,
	}, nil
}

