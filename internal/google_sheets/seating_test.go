package google_sheets

import (
	"encoding/json"
	"testing"
)

func TestNormalizeComparableGuestNameIgnoresWordOrder(t *testing.T) {
	t.Parallel()

	left := normalizeComparableGuestName("Иван Петров")
	right := normalizeComparableGuestName("Петров Иван")

	if left == "" || right == "" {
		t.Fatal("normalized comparable names should not be empty")
	}

	if left != right {
		t.Fatalf("normalizeComparableGuestName() should ignore word order, got %q and %q", left, right)
	}
}

func TestSeatingTableJSONUsesMiniAppContract(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(SeatingTable{
		Table:  "7",
		Guests: []string{"Иван Петров", "Мария Сидорова"},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	const want = `{"table":"7","guests":["Иван Петров","Мария Сидорова"]}`
	if string(payload) != want {
		t.Fatalf("json.Marshal() = %s, want %s", payload, want)
	}
}
