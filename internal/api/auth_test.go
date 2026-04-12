package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"wedding-bot/internal/cache"
	"wedding-bot/internal/config"
)

func TestEncodeDecodeAuthSessionRoundTrip(t *testing.T) {
	previousToken := config.BotToken
	config.BotToken = "test-bot-token"
	defer func() {
		config.BotToken = previousToken
	}()

	raw, err := encodeAuthSession(123456, "@TestUser")
	if err != nil {
		t.Fatalf("encodeAuthSession() error = %v", err)
	}

	userID, username, ok := decodeAuthSession(raw)
	if !ok {
		t.Fatal("decodeAuthSession() returned ok=false")
	}
	if userID != 123456 {
		t.Fatalf("decodeAuthSession() userID = %d, want %d", userID, 123456)
	}
	if username != "testuser" {
		t.Fatalf("decodeAuthSession() username = %q, want %q", username, "testuser")
	}
}

func TestDecodeAuthSessionRejectsTamperedValue(t *testing.T) {
	previousToken := config.BotToken
	config.BotToken = "test-bot-token"
	defer func() {
		config.BotToken = previousToken
	}()

	raw, err := encodeAuthSession(42, "@Tampered")
	if err != nil {
		t.Fatalf("encodeAuthSession() error = %v", err)
	}

	tampered := raw[:len(raw)-1] + "A"
	if _, _, ok := decodeAuthSession(tampered); ok {
		t.Fatal("decodeAuthSession() accepted tampered session")
	}
}

func TestResolveAuthIdentityUsesInitData(t *testing.T) {
	previousToken := config.BotToken
	config.BotToken = "test-bot-token"
	defer func() {
		config.BotToken = previousToken
	}()

	initData := buildSignedInitData(t, config.BotToken, map[string]interface{}{
		"id":         777001,
		"username":   "RoadMapUser",
		"first_name": "Road",
		"last_name":  "Map",
	})

	userID, username := resolveAuthIdentity(0, "", initData)
	if userID != 777001 {
		t.Fatalf("resolveAuthIdentity() userID = %d, want %d", userID, 777001)
	}
	if username != "roadmapuser" {
		t.Fatalf("resolveAuthIdentity() username = %q, want %q", username, "roadmapuser")
	}
}

func TestResolveAuthIdentityFromRequestUsesCookieFallback(t *testing.T) {
	previousToken := config.BotToken
	config.BotToken = "test-bot-token"
	defer func() {
		config.BotToken = previousToken
	}()

	rawSession, err := encodeAuthSession(555123, "@CookieUser")
	if err != nil {
		t.Fatalf("encodeAuthSession() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  authSessionCookieName,
		Value: rawSession,
	})

	userID, username := resolveAuthIdentityFromRequest(req, 0, "", "")
	if userID != 555123 {
		t.Fatalf("resolveAuthIdentityFromRequest() userID = %d, want %d", userID, 555123)
	}
	if username != "cookieuser" {
		t.Fatalf("resolveAuthIdentityFromRequest() username = %q, want %q", username, "cookieuser")
	}
}

func TestResolveAuthIdentityUsesCachedUsername(t *testing.T) {
	cache.InitMemoryCache()
	cache.ClearMemoryCache()

	cacheUsernameToUserID("@CachedUser", 998877)

	userID, username := resolveAuthIdentity(0, "@CachedUser", "")
	if userID != 998877 {
		t.Fatalf("resolveAuthIdentity() userID = %d, want %d", userID, 998877)
	}
	if username != "cacheduser" {
		t.Fatalf("resolveAuthIdentity() username = %q, want %q", username, "cacheduser")
	}
}

func TestBuildDataCheckStringSortsKeysAndSkipsHash(t *testing.T) {
	got := buildDataCheckString(map[string]string{
		"hash":      "ignored",
		"user":      "payload",
		"auth_date": "1700000000",
		"query_id":  "query",
	})

	want := "auth_date=1700000000\nquery_id=query\nuser=payload"
	if got != want {
		t.Fatalf("buildDataCheckString() = %q, want %q", got, want)
	}
}

func buildSignedInitData(t *testing.T, botToken string, user map[string]interface{}) string {
	t.Helper()

	userJSON, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("json.Marshal(user) error = %v", err)
	}

	params := map[string]string{
		"auth_date": "1700000000",
		"query_id":  "AAEAAAE",
		"user":      string(userJSON),
	}

	dataCheckString := buildDataCheckString(params)

	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))

	signature := hmac.New(sha256.New, secret.Sum(nil))
	signature.Write([]byte(dataCheckString))

	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	values.Set("hash", hex.EncodeToString(signature.Sum(nil)))

	return values.Encode()
}
