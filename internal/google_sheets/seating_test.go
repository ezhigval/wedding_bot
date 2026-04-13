package google_sheets

import "testing"

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
