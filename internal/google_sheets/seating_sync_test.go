package google_sheets

import (
	"math"
	"testing"

	"google.golang.org/api/sheets/v4"
)

func TestBuildEditableSeatingMatrixUsesColumnAAndAppliesCanonicalOrder(t *testing.T) {
	t.Parallel()

	guestRows := []guestSeatRow{
		{Row: 2, FullName: "Зоя Иванова", ComparableName: normalizeComparableGuestName("Зоя Иванова"), Table: "", Confirmed: false},
		{Row: 3, FullName: "Анна Петрова", ComparableName: normalizeComparableGuestName("Анна Петрова"), Table: "", Confirmed: true},
		{Row: 4, FullName: "Борис Петров", ComparableName: normalizeComparableGuestName("Борис Петров"), Table: "2", Confirmed: true},
		{Row: 5, FullName: "Антон Орлов", ComparableName: normalizeComparableGuestName("Антон Орлов"), Table: "2", Confirmed: false},
		{Row: 6, FullName: "Вера Сидорова", ComparableName: normalizeComparableGuestName("Вера Сидорова"), Table: "1", Confirmed: true},
		{Row: 7, FullName: "Глеб Смирнов", ComparableName: normalizeComparableGuestName("Глеб Смирнов"), Table: "2", Confirmed: true},
	}

	snapshot := &editableSeatingSnapshot{
		UnassignedHeader: editableSeatingUnassignedLabel,
		TableOrder:       []string{"1", "2"},
	}

	got := buildEditableSeatingMatrix(guestRows, snapshot)

	want := [][]interface{}{
		{editableSeatingUnassignedLabel, "1", "2"},
		{"Анна Петрова", "Вера Сидорова", "Борис Петров"},
		{"Зоя Иванова", "", "Глеб Смирнов"},
		{"", "", "Антон Орлов"},
	}

	if !sheetMatrixEquals(got, want) {
		t.Fatalf("buildEditableSeatingMatrix() = %#v, want %#v", got, want)
	}
}

func TestParseGuestSeatRowsReadsGuestMetadata(t *testing.T) {
	t.Parallel()

	values := [][]interface{}{
		{"Имя", "B", "Подтверждение", "Родство", "Сторона", "USER_ID", "Стол"},
		{"Иванов Иван", "", "TRUE", "Друзья", "Жених", "101", "5"},
		{"Петров Петр", "", "ЛОЖЬ", "Семья", "Невеста", "102", ""},
		{"Иванов Иван", "", "ДА", "Родственники", "Общие", "103", "2"},
	}

	rows, ambiguous := parseGuestSeatRows(values)
	if len(rows) != 3 {
		t.Fatalf("parseGuestSeatRows() rows len = %d, want 3", len(rows))
	}

	first := rows[0]
	if first.Row != 2 || first.FullName != "Иванов Иван" || !first.Confirmed || first.Table != "5" {
		t.Fatalf("first parsed row = %#v", first)
	}
	if first.Category != "Друзья" || first.Side != "Жених" {
		t.Fatalf("first parsed metadata = %#v", first)
	}

	second := rows[1]
	if second.Row != 3 || second.FullName != "Петров Петр" || second.Confirmed || second.Table != "" {
		t.Fatalf("second parsed row = %#v", second)
	}

	if _, exists := ambiguous[normalizeComparableGuestName("Иванов Иван")]; !exists {
		t.Fatalf("ambiguous names = %v, want duplicate for Иванов Иван", ambiguous)
	}
}

func TestParseEditableSeatingSnapshotBuildsAssignments(t *testing.T) {
	t.Parallel()

	values := [][]interface{}{
		{editableSeatingUnassignedLabel, "2", "5"},
		{"Зоя Иванова", "Борис Петров", ""},
		{"", "Антонов Антон", "Борис Петров"},
	}

	snapshot := parseEditableSeatingSnapshot(values)
	if snapshot.UnassignedHeader != editableSeatingUnassignedLabel {
		t.Fatalf("snapshot.UnassignedHeader = %q, want %q", snapshot.UnassignedHeader, editableSeatingUnassignedLabel)
	}

	wantOrder := []string{"2", "5"}
	if len(snapshot.TableOrder) != len(wantOrder) {
		t.Fatalf("snapshot.TableOrder len = %d, want %d (%v)", len(snapshot.TableOrder), len(wantOrder), snapshot.TableOrder)
	}
	for idx := range wantOrder {
		if snapshot.TableOrder[idx] != wantOrder[idx] {
			t.Fatalf("snapshot.TableOrder[%d] = %q, want %q (%v)", idx, snapshot.TableOrder[idx], wantOrder[idx], snapshot.TableOrder)
		}
	}

	if snapshot.AssignmentByComparableName[normalizeComparableGuestName("Зоя Иванова")] != "" {
		t.Fatalf("unassigned mapping = %q, want empty table", snapshot.AssignmentByComparableName[normalizeComparableGuestName("Зоя Иванова")])
	}
	if snapshot.AssignmentByComparableName[normalizeComparableGuestName("Антонов Антон")] != "2" {
		t.Fatalf("Антонов mapping = %q, want 2", snapshot.AssignmentByComparableName[normalizeComparableGuestName("Антонов Антон")])
	}
	if _, exists := snapshot.AmbiguousComparableNames[normalizeComparableGuestName("Борис Петров")]; !exists {
		t.Fatalf("ambiguous seating names = %v, want duplicate for Борис Петров", snapshot.AmbiguousComparableNames)
	}
}

func TestBuildEditableSeatingLayoutIncludesCellFormatting(t *testing.T) {
	t.Parallel()

	guestRows := []guestSeatRow{
		{
			Row:            2,
			FullName:       "Иванов Иван",
			ComparableName: normalizeComparableGuestName("Иванов Иван"),
			Table:          "5",
			Confirmed:      true,
			Category:       "Друзья",
			Side:           "Жених",
		},
		{
			Row:            3,
			FullName:       "Антонов Антон",
			ComparableName: normalizeComparableGuestName("Антонов Антон"),
			Table:          "5",
			Confirmed:      false,
			Category:       "Родственник",
			Side:           "Невеста",
		},
	}

	layout := buildEditableSeatingLayout(guestRows, &editableSeatingSnapshot{
		UnassignedHeader: editableSeatingUnassignedLabel,
		TableOrder:       []string{"5"},
	})

	if len(layout.GuestCellStyles) != 2 {
		t.Fatalf("buildEditableSeatingLayout() styles len = %d, want 2", len(layout.GuestCellStyles))
	}

	first := layout.GuestCellStyles[0]
	if first.RowIndex != 1 || first.ColumnIndex != 1 {
		t.Fatalf("first style position = (%d,%d), want (1,1)", first.RowIndex, first.ColumnIndex)
	}

	assertSheetsColorClose(t, first.Format.BackgroundColor, friendsBackgroundColor)
	assertSheetsColorClose(t, first.Format.TextFormat.ForegroundColor, groomForegroundColor)

	second := layout.GuestCellStyles[1]
	if second.RowIndex != 2 || second.ColumnIndex != 1 {
		t.Fatalf("second style position = (%d,%d), want (2,1)", second.RowIndex, second.ColumnIndex)
	}

	assertSheetsColorClose(t, second.Format.BackgroundColor, relativesBackgroundColor)
	assertSheetsColorClose(t, second.Format.TextFormat.ForegroundColor, brideForegroundColor)
}

func TestMergeEditableTableOrderAppendsMissingTablesInNumericOrder(t *testing.T) {
	t.Parallel()

	guestRows := []guestSeatRow{
		{Table: "10"},
		{Table: "3"},
		{Table: ""},
		{Table: "2"},
	}

	got := mergeEditableTableOrder([]string{"1", "5"}, guestRows)
	want := []string{"1", "5", "2", "3", "10"}

	if len(got) != len(want) {
		t.Fatalf("mergeEditableTableOrder() length = %d, want %d (%v)", len(got), len(want), got)
	}

	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("mergeEditableTableOrder()[%d] = %q, want %q (full=%v)", idx, got[idx], want[idx], got)
		}
	}
}

func assertSheetsColorClose(t *testing.T, got, want *sheets.Color) {
	t.Helper()

	if got == nil || want == nil {
		t.Fatalf("assertSheetsColorClose() got=%v want=%v", got, want)
	}

	if math.Abs(got.Red-want.Red) > 0.0001 || math.Abs(got.Green-want.Green) > 0.0001 || math.Abs(got.Blue-want.Blue) > 0.0001 {
		t.Fatalf("assertSheetsColorClose() got=(%f,%f,%f) want=(%f,%f,%f)", got.Red, got.Green, got.Blue, want.Red, want.Green, want.Blue)
	}
}
