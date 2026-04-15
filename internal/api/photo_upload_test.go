package api

import "testing"

const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+XgnsAAAAASUVORK5CYII="

func TestDecodeBase64PhotoDataDataURL(t *testing.T) {
	content, mimeType, err := decodeBase64PhotoData("data:image/png;base64," + tinyPNGBase64)
	if err != nil {
		t.Fatalf("decodeBase64PhotoData() error = %v", err)
	}
	if len(content) == 0 {
		t.Fatal("decodeBase64PhotoData() returned empty content")
	}
	if mimeType != "image/png" {
		t.Fatalf("decodeBase64PhotoData() mimeType = %q, want %q", mimeType, "image/png")
	}
}

func TestDecodeBase64PhotoDataPlainBase64(t *testing.T) {
	content, mimeType, err := decodeBase64PhotoData(tinyPNGBase64)
	if err != nil {
		t.Fatalf("decodeBase64PhotoData() error = %v", err)
	}
	if len(content) == 0 {
		t.Fatal("decodeBase64PhotoData() returned empty content")
	}
	if mimeType != "image/png" {
		t.Fatalf("decodeBase64PhotoData() mimeType = %q, want %q", mimeType, "image/png")
	}
}

func TestDecodeBase64PhotoDataRejectsInvalidPayload(t *testing.T) {
	if _, _, err := decodeBase64PhotoData("not-an-image"); err == nil {
		t.Fatal("decodeBase64PhotoData() expected error for invalid payload")
	}
}
