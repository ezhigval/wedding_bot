package google_sheets

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

// SeatingLockStatus представляет статус закрепления рассадки
type SeatingLockStatus struct {
	Locked   bool
	LockedAt string
	Reason   string
}

// GetSeatingLockStatus получает статус закрепления рассадки из листа Config
func GetSeatingLockStatus(ctx context.Context) (*SeatingLockStatus, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Config"

	// Проверяем существование листа
	if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
		return &SeatingLockStatus{
			Locked:   false,
			LockedAt: "",
			Reason:   "no_config_sheet",
		}, nil
	}

	readRange := fmt.Sprintf("%s!A:B", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	status := &SeatingLockStatus{
		Locked:   false,
		LockedAt: "",
		Reason:   "not_locked",
	}

	// Ищем SEATING_LOCKED и SEATING_LOCKED_AT
	for _, row := range resp.Values {
		if len(row) > 0 {
			key := ""
			if val, ok := row[0].(string); ok {
				key = strings.TrimSpace(val)
			}

			if key == "SEATING_LOCKED" && len(row) > 1 {
				if val, ok := row[1].(string); ok {
					if strings.TrimSpace(val) == "1" {
						status.Locked = true
					}
				}
			}

			if key == "SEATING_LOCKED_AT" && len(row) > 1 {
				if val, ok := row[1].(string); ok {
					status.LockedAt = strings.TrimSpace(val)
				}
			}
		}
	}

	return status, nil
}

// LockSeating закрепляет текущую рассадку
func LockSeating(ctx context.Context) (*SeatingLockStatus, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID

	// Проверяем текущий статус
	status, err := GetSeatingLockStatus(ctx)
	if err != nil {
		return nil, err
	}

	if status.Locked {
		return &SeatingLockStatus{
			Locked:   true,
			LockedAt: status.LockedAt,
			Reason:   "already_locked",
		}, nil
	}

	// Читаем текущую рассадку
	seating, err := GetSeatingFromSheets(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения рассадки: %w", err)
	}

	// Создаём/очищаем лист 'Рассадка_фикс'
	fixedSheetName := "Рассадка_фикс"
	if err := EnsureSheetExists(spreadsheetID, fixedSheetName); err != nil {
		return nil, fmt.Errorf("ошибка создания листа Рассадка_фикс: %w", err)
	}

	// Очищаем лист (удаляем все значения)
	clearRange := fmt.Sprintf("%s!A:Z", fixedSheetName)
	clearRequest := &sheets.ClearValuesRequest{}
	_, err = service.Spreadsheets.Values.Clear(spreadsheetID, clearRange, clearRequest).Do()
	if err != nil {
		log.Printf("Ошибка очистки листа Рассадка_фикс: %v", err)
	}

	// Записываем рассадку в Рассадка_фикс
	// Формируем данные: первая строка - заголовки (столы), остальные - гости
	maxGuests := 0
	for _, table := range seating {
		if len(table.Guests) > maxGuests {
			maxGuests = len(table.Guests)
		}
	}

	// Создаем матрицу данных
	values := make([][]interface{}, maxGuests+1)
	
	// Заголовок (первая строка)
	header := []interface{}{""} // Пустая ячейка в столбце A
	for _, table := range seating {
		header = append(header, table.Table)
	}
	values[0] = header

	// Данные (гости)
	for i := 0; i < maxGuests; i++ {
		row := []interface{}{""} // Пустая ячейка в столбце A
		for _, table := range seating {
			if i < len(table.Guests) {
				row = append(row, table.Guests[i])
			} else {
				row = append(row, "")
			}
		}
		values[i+1] = row
	}

	// Записываем данные
	if err := UpdateSheetValues(spreadsheetID, fixedSheetName, "A1", values); err != nil {
		return nil, fmt.Errorf("ошибка записи в Рассадка_фикс: %w", err)
	}

	// Обновляем Config
	configSheetName := "Config"
	if err := EnsureSheetExists(spreadsheetID, configSheetName); err != nil {
		return nil, fmt.Errorf("ошибка создания листа Config: %w", err)
	}

	nowStr := time.Now().Format("2006-01-02 15:04:05")
	configValues := [][]interface{}{
		{"SEATING_LOCKED", "1"},
		{"SEATING_LOCKED_AT", nowStr},
	}

	if err := UpdateSheetValues(spreadsheetID, configSheetName, "A1", configValues); err != nil {
		return nil, fmt.Errorf("ошибка обновления Config: %w", err)
	}

	log.Printf("Рассадка закреплена в листе 'Рассадка_фикс' в %s", nowStr)
	return &SeatingLockStatus{
		Locked:   true,
		LockedAt: nowStr,
		Reason:   "ok",
	}, nil
}

