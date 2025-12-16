package google_sheets

import (
	"context"
	"fmt"
	"log"
	"regexp"
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
	// Порядок приоритетов: настроенное имя, затем дефолты на всякий случай
	sheetNames := []string{config.GoogleSheetsTimelineSheetName, "Публичная План-сетка", "Публичная план-сетка", "Публичная план сетка"}

	for _, sheetName := range sheetNames {
		if sheetName == "" {
			continue
		}

		// Проверяем существование листа
		if err := EnsureSheetExists(spreadsheetID, sheetName); err != nil {
			continue
		}

		// Получаем данные из первых двух столбцов (A: время, B: событие)
		readRange := fmt.Sprintf("%s!A:B", sheetName)
		resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
		if err != nil {
			log.Printf("Ошибка чтения план-сетки с листа %s: %v", sheetName, err)
			continue
		}

		timeline := parseTimelineRows(resp.Values)
		if len(timeline) > 0 {
			return timeline, nil
		}
	}

	return []TimelineItem{}, nil
}

func parseTimelineRows(rows [][]interface{}) []TimelineItem {
	var timeline []TimelineItem
	timeRe := regexp.MustCompile(`^\d{1,2}[:.]\d{2}`)

	for i, row := range rows {
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

		// Пропускаем полностью пустые строки
		if eventCell == "" && timeCell == "" {
			continue
		}

		// Пропускаем заголовок
		if i == 0 && (strings.Contains(strings.ToLower(eventCell), "событие") || strings.Contains(strings.ToLower(timeCell), "время")) {
			continue
		}

		// Определяем порядок столбцов (A:время/B:событие или наоборот)
		timeVal := timeCell
		eventVal := eventCell

		// Если событие пустое, пробуем определить, что время в столбце B, а событие в A
		if eventVal == "" && timeRe.MatchString(eventCell) {
			timeVal = eventCell
			eventVal = timeCell
		}

		// Если событие пустое, но столбец A похож на событие, а B на время
		if eventVal == "" && timeVal != "" && len(row) > 1 {
			if timeRe.MatchString(eventCell) {
				timeVal = eventCell
				eventVal = timeCell
			} else if timeRe.MatchString(timeCell) && eventCell != "" {
				timeVal = timeCell
				eventVal = eventCell
			}
		}

		// Если всё ещё пустое событие, пропускаем
		if eventVal == "" {
			continue
		}

		timeline = append(timeline, TimelineItem{
			Time:  timeVal,
			Event: eventVal,
		})
	}

	return timeline
}
