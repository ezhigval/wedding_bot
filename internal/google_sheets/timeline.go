package google_sheets

import (
	"context"
	"fmt"
	"log"
	"strings"

	"wedding-bot/internal/config"
)

// TimelineItem представляет элемент тайминга
type TimelineItem struct {
	Time  string
	Event string
}

// GetTimeline получает тайминг мероприятия из Google Sheets
func GetTimeline(ctx context.Context) ([]TimelineItem, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := config.GoogleSheetsTimelineSheetName

	// Проверяем существование листа
	if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
		log.Printf("Вкладка '%s' не найдена", sheetName)
		return []TimelineItem{}, nil
	}

	// Получаем данные ТОЛЬКО из первых двух столбцов (A: время, B: событие)
	readRange := fmt.Sprintf("%s!A:B", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения значений: %w", err)
	}

	var timeline []TimelineItem
	for _, row := range resp.Values {
		// Гарантируем, что у нас всегда есть как минимум 2 ячейки
		timeCell := ""
		eventCell := ""

		if len(row) > 0 {
			if val, ok := row[0].(string); ok {
				timeCell = strings.TrimSpace(val)
			}
		}

		if len(row) > 1 {
			if val, ok := row[1].(string); ok {
				eventCell = strings.TrimSpace(val)
			}
		}

		// Пропускаем полностью пустые строки и строки без события
		if eventCell == "" {
			continue
		}

		timeline = append(timeline, TimelineItem{
			Time:  timeCell,
			Event: eventCell,
		})
	}

	return timeline, nil
}

