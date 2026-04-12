package api

import "testing"

func TestSanitizeAdditionalGuests(t *testing.T) {
	t.Parallel()

	rawGuests := []registerGuestCompanionRequest{
		{FirstName: "  Иван  ", LastName: " Петров "},
		{FirstName: "иван", LastName: "петров"},
		{FirstName: "Мария", LastName: "Иванова"},
		{FirstName: " ", LastName: " "},
		{FirstName: "Анна", LastName: ""},
		{FirstName: "Валентин", LastName: "Жохов"},
	}

	guests := sanitizeAdditionalGuests(rawGuests, "Валентин", "Жохов")
	if len(guests) != 3 {
		t.Fatalf("sanitizeAdditionalGuests() len = %d, want 3", len(guests))
	}

	if guests[0].FirstName != "Иван" || guests[0].LastName != "Петров" {
		t.Fatalf("guest[0] = %+v, want Иван Петров", guests[0])
	}

	if guests[1].FirstName != "Мария" || guests[1].LastName != "Иванова" {
		t.Fatalf("guest[1] = %+v, want Мария Иванова", guests[1])
	}

	if guests[2].FirstName != "Анна" || guests[2].LastName != "" {
		t.Fatalf("guest[2] = %+v, want Анна with empty last name", guests[2])
	}
}
