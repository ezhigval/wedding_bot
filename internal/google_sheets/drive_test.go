package google_sheets

import (
	"testing"

	"wedding-bot/internal/config"
)

func TestNormalizeGoogleDriveFolderID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "raw id",
			raw:  "1AbCdEfGhIjKlMnOpQrStUvWxYz",
			want: "1AbCdEfGhIjKlMnOpQrStUvWxYz",
		},
		{
			name: "quoted id",
			raw:  `"1AbCdEfGhIjKlMnOpQrStUvWxYz"`,
			want: "1AbCdEfGhIjKlMnOpQrStUvWxYz",
		},
		{
			name: "drive folder url",
			raw:  "https://drive.google.com/drive/folders/1AbCdEfGhIjKlMnOpQrStUvWxYz?usp=sharing",
			want: "1AbCdEfGhIjKlMnOpQrStUvWxYz",
		},
		{
			name: "drive folder url without scheme",
			raw:  "drive.google.com/drive/u/0/folders/1AbCdEfGhIjKlMnOpQrStUvWxYz?usp=sharing",
			want: "1AbCdEfGhIjKlMnOpQrStUvWxYz",
		},
		{
			name: "open url with query id",
			raw:  "https://drive.google.com/open?id=1AbCdEfGhIjKlMnOpQrStUvWxYz",
			want: "1AbCdEfGhIjKlMnOpQrStUvWxYz",
		},
		{
			name: "invalid drive url",
			raw:  "https://drive.google.com/drive/folders/",
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := normalizeGoogleDriveFolderID(tt.raw); got != tt.want {
				t.Fatalf("normalizeGoogleDriveFolderID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGoogleDriveOAuthConfigHelpers(t *testing.T) {
	originalClientID := config.GoogleDriveOAuthClientID
	originalClientSecret := config.GoogleDriveOAuthClientSecret
	originalRefreshToken := config.GoogleDriveOAuthRefreshToken
	defer func() {
		config.GoogleDriveOAuthClientID = originalClientID
		config.GoogleDriveOAuthClientSecret = originalClientSecret
		config.GoogleDriveOAuthRefreshToken = originalRefreshToken
	}()

	config.GoogleDriveOAuthClientID = ""
	config.GoogleDriveOAuthClientSecret = ""
	config.GoogleDriveOAuthRefreshToken = ""

	if hasCompleteGoogleDriveOAuthConfig() {
		t.Fatal("hasCompleteGoogleDriveOAuthConfig() = true, want false")
	}
	if hasPartialGoogleDriveOAuthConfig() {
		t.Fatal("hasPartialGoogleDriveOAuthConfig() = true, want false")
	}

	config.GoogleDriveOAuthClientID = "client-id"
	if hasCompleteGoogleDriveOAuthConfig() {
		t.Fatal("hasCompleteGoogleDriveOAuthConfig() = true, want false for partial config")
	}
	if !hasPartialGoogleDriveOAuthConfig() {
		t.Fatal("hasPartialGoogleDriveOAuthConfig() = false, want true for partial config")
	}

	missing := missingGoogleDriveOAuthFields()
	if len(missing) != 2 {
		t.Fatalf("missingGoogleDriveOAuthFields() len = %d, want 2", len(missing))
	}

	config.GoogleDriveOAuthClientSecret = "client-secret"
	config.GoogleDriveOAuthRefreshToken = "refresh-token"

	if !hasCompleteGoogleDriveOAuthConfig() {
		t.Fatal("hasCompleteGoogleDriveOAuthConfig() = false, want true")
	}
	if !hasPartialGoogleDriveOAuthConfig() {
		t.Fatal("hasPartialGoogleDriveOAuthConfig() = false, want true")
	}
	if len(missingGoogleDriveOAuthFields()) != 0 {
		t.Fatalf("missingGoogleDriveOAuthFields() = %v, want []", missingGoogleDriveOAuthFields())
	}
}
