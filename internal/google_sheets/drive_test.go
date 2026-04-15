package google_sheets

import "testing"

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
