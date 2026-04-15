package google_sheets

import "testing"

func TestBuildEditableSeatingMatrixUsesColumnAForUnassignedAndPreservesOrder(t *testing.T) {
	t.Parallel()

	guestRows := []guestSeatRow{
		{Row: 2, FullName: "Анна Иванова", ComparableName: normalizeComparableGuestName("Анна Иванова"), Table: ""},
		{Row: 3, FullName: "Борис Петров", ComparableName: normalizeComparableGuestName("Борис Петров"), Table: "2"},
		{Row: 4, FullName: "Вера Сидорова", ComparableName: normalizeComparableGuestName("Вера Сидорова"), Table: "1"},
		{Row: 5, FullName: "Глеб Орлов", ComparableName: normalizeComparableGuestName("Глеб Орлов"), Table: "2"},
	}

	snapshot := &editableSeatingSnapshot{
		UnassignedHeader: editableSeatingUnassignedLabel,
		TableOrder:       []string{"1", "2"},
		OrderedComparableByTable: map[string][]string{
			"2": {
				normalizeComparableGuestName("Глеб Орлов"),
				normalizeComparableGuestName("Борис Петров"),
			},
		},
	}

	got := buildEditableSeatingMatrix(guestRows, snapshot)

	want := [][]interface{}{
		{editableSeatingUnassignedLabel, "1", "2"},
		{"Анна Иванова", "Вера Сидорова", "Глеб Орлов"},
		{"", "", "Борис Петров"},
	}

	if !sheetMatrixEquals(got, want) {
		t.Fatalf("buildEditableSeatingMatrix() = %#v, want %#v", got, want)
	}
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
