package google_sheets

import "testing"

func TestFindInvitationRow(t *testing.T) {
	t.Parallel()

	values := [][]interface{}{
		{"Имя", "Telegram", "Статус", "user_id"},
		{"Иван Иванов", "@ivanov", "ДА", ""},
		{"Мария Петрова", "@maria", "НЕТ", ""},
	}

	tests := []struct {
		name        string
		guestName   string
		username    *string
		expectedRow int
		expectedBy  string
	}{
		{
			name:        "find by guest name first",
			guestName:   "Мария Петрова",
			expectedRow: 3,
			expectedBy:  "name",
		},
		{
			name:        "fallback to username",
			guestName:   "",
			username:    stringPtr("@ivanov"),
			expectedRow: 2,
			expectedBy:  "username",
		},
		{
			name:        "missing invitation",
			guestName:   "Неизвестный Гость",
			username:    stringPtr("@unknown"),
			expectedRow: -1,
			expectedBy:  "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotRow, gotBy := findInvitationRow(values, tt.guestName, tt.username)
			if gotRow != tt.expectedRow || gotBy != tt.expectedBy {
				t.Fatalf("findInvitationRow() = (%d, %q), want (%d, %q)", gotRow, gotBy, tt.expectedRow, tt.expectedBy)
			}
		})
	}
}

func stringPtr(value string) *string {
	return &value
}
