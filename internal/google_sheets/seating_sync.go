package google_sheets

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

const (
	editableSeatingSheetName       = "Рассадка"
	editableSeatingReadRange       = "A:AZ"
	editableSeatingUnassignedLabel = "Без стола"
)

// SeatingSyncReport описывает результат синхронизации между листами гостей и рассадки.
type SeatingSyncReport struct {
	GuestRows        int  `json:"guest_rows"`
	SeatingTables    int  `json:"seating_tables"`
	UnassignedGuests int  `json:"unassigned_guests"`
	UpdatedGuestRows int  `json:"updated_guest_rows,omitempty"`
	RewrittenSeating bool `json:"rewritten_seating,omitempty"`
}

type guestSeatRow struct {
	Row            int
	FullName       string
	ComparableName string
	Table          string
}

type editableSeatingSnapshot struct {
	RawValues                  [][]interface{}
	UnassignedHeader           string
	TableOrder                 []string
	OrderedComparableByTable   map[string][]string
	AssignmentByComparableName map[string]string
	AmbiguousComparableNames   map[string]struct{}
}

// SyncSeatingFromGuestList перестраивает лист "Рассадка" по колонке G листа "Список гостей".
func SyncSeatingFromGuestList(ctx context.Context) (*SeatingSyncReport, error) {
	if err := EnsureSheetExists(config.GoogleSheetsID, editableSeatingSheetName); err != nil {
		return nil, err
	}

	guestRows, _, err := readGuestSeatRows(ctx)
	if err != nil {
		return nil, err
	}

	seatingSnapshot, err := readEditableSeatingSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	matrix := buildEditableSeatingMatrix(guestRows, seatingSnapshot)
	rewritten := false
	if !sheetMatrixEquals(seatingSnapshot.RawValues, matrix) {
		if err := rewriteEditableSeatingSheet(ctx, matrix); err != nil {
			return nil, err
		}
		rewritten = true
	}

	report := &SeatingSyncReport{
		GuestRows:        len(guestRows),
		SeatingTables:    len(mergeEditableTableOrder(seatingSnapshot.TableOrder, guestRows)),
		UnassignedGuests: countUnassignedGuestRows(guestRows),
		RewrittenSeating: rewritten,
	}

	log.Printf(
		"Синхронизация Список гостей -> Рассадка завершена: guest_rows=%d seating_tables=%d rewritten=%v",
		report.GuestRows,
		report.SeatingTables,
		report.RewrittenSeating,
	)

	return report, nil
}

// SyncGuestListTablesFromSeating обновляет колонку G листа "Список гостей" по текущему состоянию листа "Рассадка".
func SyncGuestListTablesFromSeating(ctx context.Context) (*SeatingSyncReport, error) {
	if err := EnsureSheetExists(config.GoogleSheetsID, editableSeatingSheetName); err != nil {
		return nil, err
	}

	guestRows, guestAmbiguous, err := readGuestSeatRows(ctx)
	if err != nil {
		return nil, err
	}

	seatingSnapshot, err := readEditableSeatingSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	updates := make([]*sheets.ValueRange, 0)
	unassignedCount := 0

	for _, guestRow := range guestRows {
		if hasComparableGuestName(guestAmbiguous, guestRow.ComparableName) || hasComparableGuestName(seatingSnapshot.AmbiguousComparableNames, guestRow.ComparableName) {
			continue
		}

		targetTable := normalizeSeatTable(seatingSnapshot.AssignmentByComparableName[guestRow.ComparableName])
		if targetTable == "" {
			unassignedCount++
		}

		if targetTable == guestRow.Table {
			continue
		}

		updates = append(updates, &sheets.ValueRange{
			Range:  fmt.Sprintf("%s!G%d", config.GoogleSheetsSheetName, guestRow.Row),
			Values: [][]interface{}{{targetTable}},
		})
	}

	if len(updates) > 0 {
		batchUpdate := &sheets.BatchUpdateValuesRequest{
			ValueInputOption: "USER_ENTERED",
			Data:             updates,
		}

		if _, err := service.Spreadsheets.Values.BatchUpdate(config.GoogleSheetsID, batchUpdate).Do(); err != nil {
			return nil, fmt.Errorf("ошибка обновления столов в списке гостей: %w", err)
		}
	}

	report := &SeatingSyncReport{
		GuestRows:        len(guestRows),
		SeatingTables:    len(seatingSnapshot.TableOrder),
		UnassignedGuests: unassignedCount,
		UpdatedGuestRows: len(updates),
	}

	log.Printf(
		"Синхронизация Рассадка -> Список гостей завершена: guest_rows=%d seating_tables=%d updated_rows=%d",
		report.GuestRows,
		report.SeatingTables,
		report.UpdatedGuestRows,
	)

	return report, nil
}

func readGuestSeatRows(ctx context.Context) ([]guestSeatRow, map[string]struct{}, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, nil, err
	}

	readRange := fmt.Sprintf("%s!A:G", config.GoogleSheetsSheetName)
	resp, err := service.Spreadsheets.Values.Get(config.GoogleSheetsID, readRange).Do()
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка чтения списка гостей: %w", err)
	}

	startRow := 0
	if len(resp.Values) > 0 && looksLikeGuestHeaderRow(resp.Values[0]) {
		startRow = 1
	}

	rows := make([]guestSeatRow, 0, len(resp.Values))
	nameSeen := make(map[string]int)
	ambiguous := make(map[string]struct{})

	for rowIdx := startRow; rowIdx < len(resp.Values); rowIdx++ {
		row := resp.Values[rowIdx]
		fullName := cellToStringAt(row, 0)
		if fullName == "" {
			continue
		}

		comparableName := normalizeComparableGuestName(fullName)
		if comparableName == "" {
			continue
		}

		nameSeen[comparableName]++
		if nameSeen[comparableName] > 1 {
			ambiguous[comparableName] = struct{}{}
		}

		rows = append(rows, guestSeatRow{
			Row:            rowIdx + 1,
			FullName:       fullName,
			ComparableName: comparableName,
			Table:          normalizeSeatTable(cellToStringAt(row, 6)),
		})
	}

	return rows, ambiguous, nil
}

func readEditableSeatingSnapshot(ctx context.Context) (*editableSeatingSnapshot, error) {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	readRange := fmt.Sprintf("%s!%s", editableSeatingSheetName, editableSeatingReadRange)
	resp, err := service.Spreadsheets.Values.Get(config.GoogleSheetsID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения листа рассадки: %w", err)
	}

	snapshot := &editableSeatingSnapshot{
		RawValues:                  resp.Values,
		UnassignedHeader:           editableSeatingUnassignedLabel,
		TableOrder:                 []string{},
		OrderedComparableByTable:   make(map[string][]string),
		AssignmentByComparableName: make(map[string]string),
		AmbiguousComparableNames:   make(map[string]struct{}),
	}

	if len(resp.Values) == 0 {
		return snapshot, nil
	}

	headerRow := resp.Values[0]
	if unassignedHeader := cellToStringAt(headerRow, 0); unassignedHeader != "" {
		snapshot.UnassignedHeader = unassignedHeader
	}

	for colIdx := 1; colIdx < len(headerRow); colIdx++ {
		tableName := normalizeSeatTable(cellToStringAt(headerRow, colIdx))
		if tableName == "" {
			continue
		}
		if containsString(snapshot.TableOrder, tableName) {
			continue
		}
		snapshot.TableOrder = append(snapshot.TableOrder, tableName)
	}

	for rowIdx := 1; rowIdx < len(resp.Values); rowIdx++ {
		row := resp.Values[rowIdx]

		unassignedName := cellToStringAt(row, 0)
		if unassignedName != "" {
			markEditableSeatingGuest(snapshot, "", unassignedName)
		}

		for colIdx, tableName := range snapshot.TableOrder {
			actualCol := colIdx + 1
			guestName := cellToStringAt(row, actualCol)
			if guestName == "" {
				continue
			}
			markEditableSeatingGuest(snapshot, tableName, guestName)
		}
	}

	return snapshot, nil
}

func markEditableSeatingGuest(snapshot *editableSeatingSnapshot, tableName, guestName string) {
	comparableName := normalizeComparableGuestName(guestName)
	if comparableName == "" {
		return
	}

	snapshot.OrderedComparableByTable[tableName] = append(snapshot.OrderedComparableByTable[tableName], comparableName)

	if currentTable, exists := snapshot.AssignmentByComparableName[comparableName]; exists {
		if currentTable != tableName {
			snapshot.AmbiguousComparableNames[comparableName] = struct{}{}
		}
		return
	}

	snapshot.AssignmentByComparableName[comparableName] = tableName
}

func buildEditableSeatingMatrix(guestRows []guestSeatRow, seatingSnapshot *editableSeatingSnapshot) [][]interface{} {
	tableOrder := mergeEditableTableOrder(seatingSnapshot.TableOrder, guestRows)
	rowsByComparableName, guestAmbiguous := buildGuestRowsByComparableName(guestRows)
	guestsByTable := make(map[string][]string, len(tableOrder)+1)
	addedRows := make(map[int]struct{}, len(guestRows))

	tables := make([]string, 0, len(tableOrder)+1)
	tables = append(tables, "")
	tables = append(tables, tableOrder...)

	for _, tableName := range tables {
		for _, comparableName := range seatingSnapshot.OrderedComparableByTable[tableName] {
			if hasComparableGuestName(guestAmbiguous, comparableName) {
				continue
			}

			guestRow, exists := rowsByComparableName[comparableName]
			if !exists || guestRow.Table != tableName {
				continue
			}
			if _, alreadyAdded := addedRows[guestRow.Row]; alreadyAdded {
				continue
			}

			guestsByTable[tableName] = append(guestsByTable[tableName], guestRow.FullName)
			addedRows[guestRow.Row] = struct{}{}
		}

		for _, guestRow := range guestRows {
			if guestRow.Table != tableName {
				continue
			}
			if _, alreadyAdded := addedRows[guestRow.Row]; alreadyAdded {
				continue
			}

			guestsByTable[tableName] = append(guestsByTable[tableName], guestRow.FullName)
			addedRows[guestRow.Row] = struct{}{}
		}
	}

	maxGuests := len(guestsByTable[""])
	for _, tableName := range tableOrder {
		if tableGuests := len(guestsByTable[tableName]); tableGuests > maxGuests {
			maxGuests = tableGuests
		}
	}

	values := make([][]interface{}, maxGuests+1)
	header := make([]interface{}, len(tableOrder)+1)
	if strings.TrimSpace(seatingSnapshot.UnassignedHeader) == "" {
		header[0] = editableSeatingUnassignedLabel
	} else {
		header[0] = seatingSnapshot.UnassignedHeader
	}
	for idx, tableName := range tableOrder {
		header[idx+1] = tableName
	}
	values[0] = header

	for rowIdx := 0; rowIdx < maxGuests; rowIdx++ {
		row := make([]interface{}, len(tableOrder)+1)
		for idx := range row {
			row[idx] = ""
		}
		if rowIdx < len(guestsByTable[""]) {
			row[0] = guestsByTable[""][rowIdx]
		}

		for colIdx, tableName := range tableOrder {
			if rowIdx < len(guestsByTable[tableName]) {
				row[colIdx+1] = guestsByTable[tableName][rowIdx]
			}
		}

		values[rowIdx+1] = row
	}

	return values
}

func rewriteEditableSeatingSheet(ctx context.Context, values [][]interface{}) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	clearRange := fmt.Sprintf("%s!%s", editableSeatingSheetName, editableSeatingReadRange)
	if _, err := service.Spreadsheets.Values.Clear(config.GoogleSheetsID, clearRange, &sheets.ClearValuesRequest{}).Do(); err != nil {
		return fmt.Errorf("ошибка очистки листа рассадки: %w", err)
	}

	if err := UpdateSheetValues(config.GoogleSheetsID, editableSeatingSheetName, "A1", values); err != nil {
		return fmt.Errorf("ошибка записи рассадки: %w", err)
	}

	return nil
}

func mergeEditableTableOrder(existingOrder []string, guestRows []guestSeatRow) []string {
	seen := make(map[string]struct{}, len(existingOrder))
	order := make([]string, 0, len(existingOrder))

	for _, tableName := range existingOrder {
		tableName = normalizeSeatTable(tableName)
		if tableName == "" {
			continue
		}
		if _, exists := seen[tableName]; exists {
			continue
		}
		seen[tableName] = struct{}{}
		order = append(order, tableName)
	}

	additional := make([]string, 0)
	for _, guestRow := range guestRows {
		if guestRow.Table == "" {
			continue
		}
		if _, exists := seen[guestRow.Table]; exists {
			continue
		}
		seen[guestRow.Table] = struct{}{}
		additional = append(additional, guestRow.Table)
	}

	sort.Slice(additional, func(i, j int) bool {
		return compareSeatTableLabels(additional[i], additional[j]) < 0
	})

	return append(order, additional...)
}

func buildGuestRowsByComparableName(guestRows []guestSeatRow) (map[string]guestSeatRow, map[string]struct{}) {
	rowsByComparableName := make(map[string]guestSeatRow, len(guestRows))
	ambiguous := make(map[string]struct{})

	for _, guestRow := range guestRows {
		if _, exists := rowsByComparableName[guestRow.ComparableName]; exists {
			ambiguous[guestRow.ComparableName] = struct{}{}
			continue
		}
		rowsByComparableName[guestRow.ComparableName] = guestRow
	}

	return rowsByComparableName, ambiguous
}

func compareSeatTableLabels(left, right string) int {
	left = normalizeSeatTable(left)
	right = normalizeSeatTable(right)

	leftInt, leftIsInt := parsePositiveInt(left)
	rightInt, rightIsInt := parsePositiveInt(right)

	switch {
	case leftIsInt && rightIsInt:
		switch {
		case leftInt < rightInt:
			return -1
		case leftInt > rightInt:
			return 1
		default:
			return 0
		}
	case leftIsInt:
		return -1
	case rightIsInt:
		return 1
	default:
		return strings.Compare(left, right)
	}
}

func parsePositiveInt(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	number := 0
	for _, symbol := range value {
		if symbol < '0' || symbol > '9' {
			return 0, false
		}
		number = number*10 + int(symbol-'0')
	}

	if number <= 0 {
		return 0, false
	}

	return number, true
}

func normalizeSeatTable(value string) string {
	clean := strings.TrimSpace(value)
	switch strings.ToLower(clean) {
	case "", strings.ToLower(editableSeatingUnassignedLabel):
		return ""
	default:
		return clean
	}
}

func looksLikeGuestHeaderRow(row []interface{}) bool {
	firstCell := strings.ToLower(cellToStringAt(row, 0))
	return strings.Contains(firstCell, "имя") || strings.Contains(firstCell, "фио") || strings.Contains(firstCell, "name")
}

func cellToStringAt(row []interface{}, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}

	return cellToString(row[idx])
}

func sheetMatrixEquals(left [][]interface{}, right [][]interface{}) bool {
	leftNormalized := normalizeSheetMatrix(left)
	rightNormalized := normalizeSheetMatrix(right)

	if len(leftNormalized) != len(rightNormalized) {
		return false
	}

	for rowIdx := range leftNormalized {
		if len(leftNormalized[rowIdx]) != len(rightNormalized[rowIdx]) {
			return false
		}
		for colIdx := range leftNormalized[rowIdx] {
			if leftNormalized[rowIdx][colIdx] != rightNormalized[rowIdx][colIdx] {
				return false
			}
		}
	}

	return true
}

func normalizeSheetMatrix(values [][]interface{}) [][]string {
	result := make([][]string, 0, len(values))

	for _, row := range values {
		normalizedRow := make([]string, 0, len(row))
		for _, value := range row {
			if value == nil {
				normalizedRow = append(normalizedRow, "")
				continue
			}
			normalizedRow = append(normalizedRow, strings.TrimSpace(fmt.Sprintf("%v", value)))
		}
		for len(normalizedRow) > 0 && normalizedRow[len(normalizedRow)-1] == "" {
			normalizedRow = normalizedRow[:len(normalizedRow)-1]
		}
		result = append(result, normalizedRow)
	}

	for len(result) > 0 && len(result[len(result)-1]) == 0 {
		result = result[:len(result)-1]
	}

	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func hasComparableGuestName(collection map[string]struct{}, comparableName string) bool {
	_, exists := collection[comparableName]
	return exists
}

func countUnassignedGuestRows(guestRows []guestSeatRow) int {
	count := 0
	for _, guestRow := range guestRows {
		if guestRow.Table == "" {
			count++
		}
	}
	return count
}
