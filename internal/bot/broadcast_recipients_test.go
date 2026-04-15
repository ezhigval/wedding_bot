package bot

import (
	"reflect"
	"testing"

	"wedding-bot/internal/google_sheets"
)

func TestBuildBroadcastRecipientsInfoDeduplicatesUserIDs(t *testing.T) {
	t.Parallel()

	recipients := buildBroadcastRecipientsInfo([]google_sheets.GuestInfo{
		{FirstName: "Иван", LastName: "Петров", Username: "ivan", UserID: "100"},
		{FirstName: "Иван", LastName: "Петров", Username: "ivan", UserID: "100"},
		{FirstName: "Мария", LastName: "Соколова", Username: "maria", UserID: "200"},
		{FirstName: "", LastName: "", Username: "ghost", UserID: "300"},
		{FirstName: "Bad", LastName: "Row", Username: "bad", UserID: "not-a-number"},
	})

	if len(recipients) != 3 {
		t.Fatalf("buildBroadcastRecipientsInfo() len = %d, want 3", len(recipients))
	}

	if recipients[0].UserID != 100 || recipients[0].DisplayName != "Иван Петров" {
		t.Fatalf("recipient[0] = %+v, want user 100 Иван Петров", recipients[0])
	}
	if recipients[1].UserID != 200 || recipients[1].DisplayName != "Мария Соколова" {
		t.Fatalf("recipient[1] = %+v, want user 200 Мария Соколова", recipients[1])
	}
	if recipients[2].UserID != 300 || recipients[2].DisplayName != "ghost" {
		t.Fatalf("recipient[2] = %+v, want user 300 ghost", recipients[2])
	}
}

func TestRecipientSelectionHelpers(t *testing.T) {
	t.Parallel()

	ids := addRecipientID(nil, 10)
	ids = addRecipientID(ids, 20)
	ids = addRecipientID(ids, 10)

	if !reflect.DeepEqual(ids, []int64{10, 20}) {
		t.Fatalf("addRecipientID() = %v, want [10 20]", ids)
	}

	ids = removeRecipientID(ids, 10)
	if !reflect.DeepEqual(ids, []int64{20}) {
		t.Fatalf("removeRecipientID() = %v, want [20]", ids)
	}

	filtered := filterRecipientIDsByAvailable([]int64{20, 30, 40}, []BroadcastRecipientInfo{
		{UserID: 20},
		{UserID: 40},
	})
	if !reflect.DeepEqual(filtered, []int64{20, 40}) {
		t.Fatalf("filterRecipientIDsByAvailable() = %v, want [20 40]", filtered)
	}
}

func TestBroadcastButtonHelpers(t *testing.T) {
	t.Parallel()

	state := &BroadcastState{}

	addBroadcastButton(state, "Сайт", "https://example.com")
	addBroadcastButton(state, "Сайт", "https://example.com")
	addBroadcastButton(state, "Чат", "https://t.me/example")

	if len(state.Buttons) != 2 {
		t.Fatalf("addBroadcastButton() buttons len = %d, want 2", len(state.Buttons))
	}

	rows := broadcastButtonRows(state.Buttons)
	if len(rows) != 1 || len(rows[0]) != 2 {
		t.Fatalf("broadcastButtonRows() = %#v, want one row with two buttons", rows)
	}

	replyMarkup := broadcastReplyMarkup(state)
	if replyMarkup == nil {
		t.Fatal("broadcastReplyMarkup() = nil, want markup")
	}

	legacyState := &BroadcastState{
		ButtonText: "Legacy",
		ButtonURL:  "https://legacy.example.com",
	}
	legacyMarkup := broadcastReplyMarkup(legacyState)
	if legacyMarkup == nil {
		t.Fatal("broadcastReplyMarkup() legacy fallback = nil, want markup")
	}
	if len(legacyMarkup.InlineKeyboard) != 1 || len(legacyMarkup.InlineKeyboard[0]) != 1 {
		t.Fatalf("broadcastReplyMarkup() legacy rows = %#v, want single button", legacyMarkup.InlineKeyboard)
	}
	legacyButton := legacyMarkup.InlineKeyboard[0][0]
	if legacyButton.Text != "Legacy" || broadcastButtonURL(legacyButton) != "https://legacy.example.com" {
		t.Fatalf("broadcastReplyMarkup() legacy button = %#v, want Legacy button", legacyButton)
	}

	clearBroadcastButtons(state)
	if len(state.Buttons) != 0 || state.ButtonText != "" || state.ButtonURL != "" {
		t.Fatalf("clearBroadcastButtons() state = %+v, want empty buttons", state)
	}
}
