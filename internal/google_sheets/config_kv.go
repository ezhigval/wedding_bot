package google_sheets

import (
	"context"
	"fmt"
	"strings"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

// getConfigEntries читает лист Config как key/value
func getConfigEntries(ctx context.Context) (map[string]string, map[string]int, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Config"

	if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
		return nil, nil, err
	}

	readRange := fmt.Sprintf("%s!A:B", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка чтения Config: %w", err)
	}

	values := resp.Values
	entries := make(map[string]string)
	rows := make(map[string]int) // row index (1-based) for each key

	for i, row := range values {
		if len(row) == 0 {
			continue
		}
		key := ""
		if v, ok := row[0].(string); ok {
			key = strings.TrimSpace(v)
		}
		if key == "" {
			continue
		}
		val := ""
		if len(row) > 1 {
			if v, ok := row[1].(string); ok {
				val = strings.TrimSpace(v)
			}
		}
		entries[key] = val
		rows[key] = i + 1
	}

	return entries, rows, nil
}

// upsertConfigEntries обновляет/добавляет ключи в листе Config
func upsertConfigEntries(ctx context.Context, data map[string]string) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Config"

	if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
		return err
	}

	entries, rows, err := getConfigEntries(ctx)
	if err != nil {
		return err
	}

	// Подготавливаем batch update
	var updates []*sheets.ValueRange
	appendRows := [][]interface{}{}

	for key, val := range data {
		if rowIdx, ok := rows[key]; ok {
			// Обновляем существующую строку (колонка B)
			updates = append(updates, &sheets.ValueRange{
				Range:  fmt.Sprintf("%s!B%d", sheetName, rowIdx),
				Values: [][]interface{}{{val}},
			})
		} else {
			// Добавляем новую строку
			appendRows = append(appendRows, []interface{}{key, val})
			entries[key] = val
		}
	}

	if len(updates) > 0 {
		batch := &sheets.BatchUpdateValuesRequest{
			ValueInputOption: "USER_ENTERED",
			Data:             updates,
		}
		if _, err := service.Spreadsheets.Values.BatchUpdate(spreadsheetID, batch).Do(); err != nil {
			return fmt.Errorf("ошибка обновления Config: %w", err)
		}
	}

	if len(appendRows) > 0 {
		valueRange := &sheets.ValueRange{
			Values: appendRows,
		}
		readRange := fmt.Sprintf("%s!A:B", sheetName)
		if _, err := service.Spreadsheets.Values.Append(
			spreadsheetID,
			readRange,
			valueRange,
		).ValueInputOption("USER_ENTERED").Do(); err != nil {
			return fmt.Errorf("ошибка добавления в Config: %w", err)
		}
	}

	return nil
}

func getConfigValue(ctx context.Context, key string) (string, error) {
	entries, _, err := getConfigEntries(ctx)
	if err != nil {
		return "", err
	}
	return entries[key], nil
}
