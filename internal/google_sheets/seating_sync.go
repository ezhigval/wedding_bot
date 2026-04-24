package google_sheets

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

const (
	editableSeatingSheetName       = "Рассадка"
	editableSeatingReadRange       = "A:AZ"
	editableSeatingUnassignedLabel = "Без стола"
)

var (
	defaultEditableSeatingBackgroundColor = rgbSheetsColor(255, 255, 255)
	defaultEditableSeatingForegroundColor = rgbSheetsColor(32, 35, 41)
	familyBackgroundColor                 = rgbSheetsColor(247, 230, 236)
	relativesBackgroundColor              = rgbSheetsColor(248, 242, 214)
	friendsBackgroundColor                = rgbSheetsColor(226, 240, 227)
	groomForegroundColor                  = rgbSheetsColor(69, 103, 178)
	brideForegroundColor                  = rgbSheetsColor(183, 91, 137)
	commonForegroundColor                 = rgbSheetsColor(80, 136, 103)
	editableSeatingSheetIDCache           editableSeatingSheetIDState
)

type editableSeatingSheetIDState struct {
	mu            sync.RWMutex
	spreadsheetID string
	sheetID       int64
	ok            bool
}

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
	Confirmed      bool
	Category       string
	Side           string
}

type editableSeatingSnapshot struct {
	RawValues                  [][]interface{}
	UnassignedHeader           string
	TableOrder                 []string
	OrderedComparableByTable   map[string][]string
	AssignmentByComparableName map[string]string
	AmbiguousComparableNames   map[string]struct{}
}

type editableSeatingCellFormat struct {
	RowIndex    int64
	ColumnIndex int64
	Format      *sheets.CellFormat
}

type editableSeatingLayout struct {
	Values          [][]interface{}
	TableOrder      []string
	GuestCellStyles []editableSeatingCellFormat
}

// SyncSeatingFromGuestList перестраивает лист "Рассадка" по колонке G листа "Список гостей".
func SyncSeatingFromGuestList(ctx context.Context) (*SeatingSyncReport, error) {
	sheetID, err := ensureEditableSeatingSheetID(ctx)
	if err != nil {
		return nil, err
	}

	return syncEditableSeatingFromGuestList(ctx, sheetID)
}

func syncEditableSeatingFromGuestList(ctx context.Context, sheetID int64) (*SeatingSyncReport, error) {
	guestRows, _, err := readGuestSeatRows(ctx)
	if err != nil {
		return nil, err
	}

	seatingSnapshot, err := readEditableSeatingSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	return syncEditableSeatingWithSnapshot(ctx, sheetID, guestRows, seatingSnapshot)
}

func syncEditableSeatingWithSnapshot(ctx context.Context, sheetID int64, guestRows []guestSeatRow, seatingSnapshot *editableSeatingSnapshot) (*SeatingSyncReport, error) {
	layout := buildEditableSeatingLayout(guestRows, seatingSnapshot)
	rewritten := false
	if !sheetMatrixEquals(seatingSnapshot.RawValues, layout.Values) {
		if err := rewriteEditableSeatingSheet(ctx, layout.Values); err != nil {
			return nil, err
		}
		rewritten = true
	}
	if err := applyEditableSeatingFormatting(ctx, sheetID, seatingSnapshot, layout); err != nil {
		return nil, err
	}

	report := &SeatingSyncReport{
		GuestRows:        len(guestRows),
		SeatingTables:    len(layout.TableOrder),
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
	sheetID, err := ensureEditableSeatingSheetID(ctx)
	if err != nil {
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

	for idx := range guestRows {
		guestRow := &guestRows[idx]
		if hasComparableGuestName(guestAmbiguous, guestRow.ComparableName) || hasComparableGuestName(seatingSnapshot.AmbiguousComparableNames, guestRow.ComparableName) {
			continue
		}

		targetTable := normalizeSeatTable(seatingSnapshot.AssignmentByComparableName[guestRow.ComparableName])

		if targetTable == guestRow.Table {
			continue
		}

		updates = append(updates, &sheets.ValueRange{
			Range:  fmt.Sprintf("%s!G%d", config.GoogleSheetsSheetName, guestRow.Row),
			Values: [][]interface{}{{targetTable}},
		})
		guestRow.Table = targetTable
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

	canonicalReport, err := syncEditableSeatingWithSnapshot(ctx, sheetID, guestRows, seatingSnapshot)
	if err != nil {
		return nil, err
	}

	report := &SeatingSyncReport{
		GuestRows:        len(guestRows),
		SeatingTables:    canonicalReport.SeatingTables,
		UnassignedGuests: canonicalReport.UnassignedGuests,
		UpdatedGuestRows: len(updates),
		RewrittenSeating: canonicalReport.RewrittenSeating,
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
			Confirmed:      isConfirmedValue(cellToStringAt(row, 2)),
			Category:       cellToStringAt(row, 3),
			Side:           cellToStringAt(row, 4),
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
	return buildEditableSeatingLayout(guestRows, seatingSnapshot).Values
}

func buildEditableSeatingLayout(guestRows []guestSeatRow, seatingSnapshot *editableSeatingSnapshot) *editableSeatingLayout {
	tableOrder := mergeEditableTableOrder(seatingSnapshot.TableOrder, guestRows)
	guestsByTable := buildSeatingGuestsByTable(guestRows)

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

	cellStyles := make([]editableSeatingCellFormat, 0, len(guestRows))

	for rowIdx := 0; rowIdx < maxGuests; rowIdx++ {
		row := make([]interface{}, len(tableOrder)+1)
		for idx := range row {
			row[idx] = ""
		}

		if guestRow, ok := seatingGuestAt(guestsByTable[""], rowIdx); ok {
			row[0] = guestRow.FullName
			cellStyles = append(cellStyles, editableSeatingCellFormat{
				RowIndex:    int64(rowIdx + 1),
				ColumnIndex: 0,
				Format:      buildEditableSeatingCellFormat(guestRow),
			})
		}

		for colIdx, tableName := range tableOrder {
			if guestRow, ok := seatingGuestAt(guestsByTable[tableName], rowIdx); ok {
				row[colIdx+1] = guestRow.FullName
				cellStyles = append(cellStyles, editableSeatingCellFormat{
					RowIndex:    int64(rowIdx + 1),
					ColumnIndex: int64(colIdx + 1),
					Format:      buildEditableSeatingCellFormat(guestRow),
				})
			}
		}

		values[rowIdx+1] = row
	}

	return &editableSeatingLayout{
		Values:          values,
		TableOrder:      tableOrder,
		GuestCellStyles: cellStyles,
	}
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

func applyEditableSeatingFormatting(ctx context.Context, sheetID int64, seatingSnapshot *editableSeatingSnapshot, layout *editableSeatingLayout) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	maxRows := maxInt(len(seatingSnapshot.RawValues), len(layout.Values))
	maxCols := maxSheetMatrixWidth(seatingSnapshot.RawValues, layout.Values)

	requests := make([]*sheets.Request, 0, len(layout.GuestCellStyles)+1)
	if maxRows > 1 && maxCols > 0 {
		requests = append(requests, &sheets.Request{
			RepeatCell: &sheets.RepeatCellRequest{
				Range: &sheets.GridRange{
					SheetId:          sheetID,
					StartRowIndex:    1,
					EndRowIndex:      int64(maxRows),
					StartColumnIndex: 0,
					EndColumnIndex:   int64(maxCols),
				},
				Cell: &sheets.CellData{
					UserEnteredFormat: &sheets.CellFormat{
						BackgroundColor: defaultEditableSeatingBackgroundColor,
						TextFormat: &sheets.TextFormat{
							ForegroundColor: defaultEditableSeatingForegroundColor,
						},
					},
				},
				Fields: "userEnteredFormat(backgroundColor,textFormat.foregroundColor)",
			},
		})
	}

	for _, cellStyle := range layout.GuestCellStyles {
		requests = append(requests, &sheets.Request{
			RepeatCell: &sheets.RepeatCellRequest{
				Range: &sheets.GridRange{
					SheetId:          sheetID,
					StartRowIndex:    cellStyle.RowIndex,
					EndRowIndex:      cellStyle.RowIndex + 1,
					StartColumnIndex: cellStyle.ColumnIndex,
					EndColumnIndex:   cellStyle.ColumnIndex + 1,
				},
				Cell: &sheets.CellData{
					UserEnteredFormat: cellStyle.Format,
				},
				Fields: "userEnteredFormat(backgroundColor,textFormat.foregroundColor)",
			},
		})
	}

	if len(requests) == 0 {
		return nil
	}

	if _, err := service.Spreadsheets.BatchUpdate(config.GoogleSheetsID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do(); err != nil {
		return fmt.Errorf("ошибка форматирования листа рассадки: %w", err)
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

func buildSeatingGuestsByTable(guestRows []guestSeatRow) map[string][]guestSeatRow {
	guestsByTable := make(map[string][]guestSeatRow, 4)
	for _, guestRow := range guestRows {
		guestsByTable[guestRow.Table] = append(guestsByTable[guestRow.Table], guestRow)
	}

	for tableName := range guestsByTable {
		sortGuestRowsForSeating(guestsByTable[tableName])
	}

	return guestsByTable
}

func sortGuestRowsForSeating(guestRows []guestSeatRow) {
	sort.SliceStable(guestRows, func(i, j int) bool {
		left := guestRows[i]
		right := guestRows[j]

		if left.Confirmed != right.Confirmed {
			return left.Confirmed && !right.Confirmed
		}

		leftName := normalizeSeatingSortName(left.FullName)
		rightName := normalizeSeatingSortName(right.FullName)
		if leftName != rightName {
			return strings.Compare(leftName, rightName) < 0
		}

		if left.ComparableName != right.ComparableName {
			return strings.Compare(left.ComparableName, right.ComparableName) < 0
		}

		return left.Row < right.Row
	})
}

func seatingGuestAt(guestRows []guestSeatRow, idx int) (guestSeatRow, bool) {
	if idx < 0 || idx >= len(guestRows) {
		return guestSeatRow{}, false
	}

	return guestRows[idx], true
}

func normalizeSeatingSortName(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
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

func normalizeSeatingCategory(value string) string {
	clean := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.HasPrefix(clean, "сем"):
		return "семья"
	case strings.HasPrefix(clean, "друз"):
		return "друзья"
	case strings.HasPrefix(clean, "родствен"):
		return "родственники"
	default:
		return clean
	}
}

func normalizeSeatingSide(value string) string {
	clean := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.HasPrefix(clean, "жених"):
		return "жених"
	case strings.HasPrefix(clean, "невест"):
		return "невеста"
	case strings.HasPrefix(clean, "общ"):
		return "общие"
	default:
		return clean
	}
}

func buildEditableSeatingCellFormat(guestRow guestSeatRow) *sheets.CellFormat {
	return &sheets.CellFormat{
		BackgroundColor: seatingCategoryBackgroundColor(guestRow.Category),
		TextFormat: &sheets.TextFormat{
			ForegroundColor: seatingSideForegroundColor(guestRow.Side),
		},
	}
}

func seatingCategoryBackgroundColor(category string) *sheets.Color {
	switch normalizeSeatingCategory(category) {
	case "семья":
		return familyBackgroundColor
	case "родственники":
		return relativesBackgroundColor
	case "друзья":
		return friendsBackgroundColor
	default:
		return defaultEditableSeatingBackgroundColor
	}
}

func seatingSideForegroundColor(side string) *sheets.Color {
	switch normalizeSeatingSide(side) {
	case "жених":
		return groomForegroundColor
	case "невеста":
		return brideForegroundColor
	case "общие":
		return commonForegroundColor
	default:
		return defaultEditableSeatingForegroundColor
	}
}

func rgbSheetsColor(r, g, b int) *sheets.Color {
	return &sheets.Color{
		Red:   float64(r) / 255.0,
		Green: float64(g) / 255.0,
		Blue:  float64(b) / 255.0,
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

func ensureEditableSeatingSheetID(ctx context.Context) (int64, error) {
	if cachedSheetID, ok := getCachedEditableSeatingSheetID(config.GoogleSheetsID); ok {
		return cachedSheetID, nil
	}

	spreadsheet, err := GetSpreadsheet(config.GoogleSheetsID)
	if err != nil {
		return 0, err
	}

	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == editableSeatingSheetName {
			cacheEditableSeatingSheetID(config.GoogleSheetsID, sheet.Properties.SheetId)
			return sheet.Properties.SheetId, nil
		}
	}

	service, err := GetGoogleSheetsClient()
	if err != nil {
		return 0, err
	}

	resp, err := service.Spreadsheets.BatchUpdate(config.GoogleSheetsID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{
						Title: editableSeatingSheetName,
					},
				},
			},
		},
	}).Do()
	if err != nil {
		return 0, fmt.Errorf("ошибка создания листа %s: %w", editableSeatingSheetName, err)
	}

	if len(resp.Replies) > 0 && resp.Replies[0].AddSheet != nil && resp.Replies[0].AddSheet.Properties != nil {
		sheetID := resp.Replies[0].AddSheet.Properties.SheetId
		cacheEditableSeatingSheetID(config.GoogleSheetsID, sheetID)
		return sheetID, nil
	}

	invalidateEditableSeatingSheetID(config.GoogleSheetsID)
	return 0, fmt.Errorf("лист %s создан, но sheet_id не получен", editableSeatingSheetName)
}

func maxSheetMatrixWidth(matrices ...[][]interface{}) int {
	maxWidth := 0
	for _, matrix := range matrices {
		for _, row := range matrix {
			if len(row) > maxWidth {
				maxWidth = len(row)
			}
		}
	}
	return maxWidth
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func getCachedEditableSeatingSheetID(spreadsheetID string) (int64, bool) {
	editableSeatingSheetIDCache.mu.RLock()
	defer editableSeatingSheetIDCache.mu.RUnlock()

	if !editableSeatingSheetIDCache.ok || editableSeatingSheetIDCache.spreadsheetID != spreadsheetID {
		return 0, false
	}

	return editableSeatingSheetIDCache.sheetID, true
}

func cacheEditableSeatingSheetID(spreadsheetID string, sheetID int64) {
	editableSeatingSheetIDCache.mu.Lock()
	defer editableSeatingSheetIDCache.mu.Unlock()

	editableSeatingSheetIDCache.spreadsheetID = spreadsheetID
	editableSeatingSheetIDCache.sheetID = sheetID
	editableSeatingSheetIDCache.ok = true
}

func invalidateEditableSeatingSheetID(spreadsheetID string) {
	editableSeatingSheetIDCache.mu.Lock()
	defer editableSeatingSheetIDCache.mu.Unlock()

	if editableSeatingSheetIDCache.spreadsheetID != spreadsheetID {
		return
	}

	editableSeatingSheetIDCache.spreadsheetID = ""
	editableSeatingSheetIDCache.sheetID = 0
	editableSeatingSheetIDCache.ok = false
}
