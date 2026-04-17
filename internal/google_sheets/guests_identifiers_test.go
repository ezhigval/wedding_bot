package google_sheets

import "testing"

func TestNormalizeGuestIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "numeric", input: "123456", expected: "123456"},
		{name: "numeric float", input: "123456.0", expected: "123456"},
		{name: "username with at", input: "@TestUser", expected: "testuser"},
		{name: "username url", input: "https://t.me/TestUser", expected: "testuser"},
		{name: "trim quotes", input: `"@QuotedUser"`, expected: "quoteduser"},
		{name: "empty", input: "   ", expected: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeGuestIdentifier(tt.input)
			if got != tt.expected {
				t.Fatalf("normalizeGuestIdentifier(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildAuthIdentifierForStorage(t *testing.T) {
	t.Parallel()

	if got := buildAuthIdentifierForStorage(123456, ""); got != "123456" {
		t.Fatalf("expected numeric identifier, got %q", got)
	}

	// user_id приоритетнее username в основной логике регистрации.
	if got := buildAuthIdentifierForStorage(123456, "@TestUser"); got != "123456" {
		t.Fatalf("expected user_id identifier, got %q", got)
	}

	if got := buildAuthIdentifierForStorage(0, "@TestUser"); got != "testuser" {
		t.Fatalf("expected username fallback identifier, got %q", got)
	}
}

func TestGuestIdentifierMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sheetCell string
		userID    int
		username  string
		expected  bool
	}{
		{name: "match by user id", sheetCell: "123456", userID: 123456, username: "", expected: true},
		{name: "match by user id float", sheetCell: "123456.0", userID: 123456, username: "", expected: true},
		{name: "match by username", sheetCell: "@TestUser", userID: 0, username: "testuser", expected: true},
		{name: "match by username url", sheetCell: "https://t.me/TestUser", userID: 0, username: "@testuser", expected: true},
		{name: "mismatch", sheetCell: "another_user", userID: 123456, username: "testuser", expected: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := guestIdentifierMatches(tt.sheetCell, tt.userID, tt.username)
			if got != tt.expected {
				t.Fatalf("guestIdentifierMatches(%q, %d, %q) = %v, want %v", tt.sheetCell, tt.userID, tt.username, got, tt.expected)
			}
		})
	}
}

func TestResolveGuestIdentifierUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		existingIdentifier string
		userID             int
		username           string
		expectedIdentifier string
		expectedWrite      bool
	}{
		{
			name:               "promote username to user id",
			existingIdentifier: "@TestUser",
			userID:             123456,
			username:           "testuser",
			expectedIdentifier: "123456",
			expectedWrite:      true,
		},
		{
			name:               "keep same numeric user id",
			existingIdentifier: "123456",
			userID:             123456,
			username:           "testuser",
			expectedIdentifier: "",
			expectedWrite:      false,
		},
		{
			name:               "do not overwrite foreign numeric user id",
			existingIdentifier: "654321",
			userID:             123456,
			username:           "testuser",
			expectedIdentifier: "",
			expectedWrite:      false,
		},
		{
			name:               "write username only for empty cell fallback",
			existingIdentifier: "",
			userID:             0,
			username:           "@TestUser",
			expectedIdentifier: "testuser",
			expectedWrite:      true,
		},
		{
			name:               "do not downgrade numeric user id to username",
			existingIdentifier: "123456",
			userID:             0,
			username:           "@TestUser",
			expectedIdentifier: "",
			expectedWrite:      false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotIdentifier, gotWrite := resolveGuestIdentifierUpdate(tt.existingIdentifier, tt.userID, tt.username)
			if gotIdentifier != tt.expectedIdentifier || gotWrite != tt.expectedWrite {
				t.Fatalf("resolveGuestIdentifierUpdate(%q, %d, %q) = (%q, %v), want (%q, %v)", tt.existingIdentifier, tt.userID, tt.username, gotIdentifier, gotWrite, tt.expectedIdentifier, tt.expectedWrite)
			}
		})
	}
}
