package bot

import (
	"testing"

	"wedding-bot/internal/config"
)

func TestToggleBroadcastPresetButton(t *testing.T) {
	t.Parallel()

	oldWebappURL := config.WebappURL
	oldGroupLink := config.GroupLink
	oldGroupID := config.GroupID
	t.Cleanup(func() {
		config.WebappURL = oldWebappURL
		config.GroupLink = oldGroupLink
		config.GroupID = oldGroupID
	})

	config.WebappURL = "https://example.com/app"
	config.GroupLink = "https://t.me/examplechat"
	config.GroupID = ""

	state := &BroadcastState{}

	if err := toggleBroadcastPresetButton(state, broadcastPresetMiniApp); err != nil {
		t.Fatalf("toggleBroadcastPresetButton(miniapp) error = %v", err)
	}
	if !isBroadcastPresetSelected(state, broadcastPresetMiniApp) {
		t.Fatal("miniapp preset should be selected")
	}
	if len(state.Buttons) != 1 || broadcastButtonURL(state.Buttons[0]) != "https://example.com/app" {
		t.Fatalf("state.Buttons after miniapp = %#v, want one webapp button", state.Buttons)
	}

	if err := toggleBroadcastPresetButton(state, broadcastPresetConfirmAttendance); err != nil {
		t.Fatalf("toggleBroadcastPresetButton(confirm_attendance) error = %v", err)
	}
	if len(state.Buttons) != 2 {
		t.Fatalf("state.Buttons len = %d, want 2", len(state.Buttons))
	}
	if broadcastButtonCallbackData(state.Buttons[1]) != broadcastGuestAttendanceConfirm {
		t.Fatalf("confirm button callback = %q, want %q", broadcastButtonCallbackData(state.Buttons[1]), broadcastGuestAttendanceConfirm)
	}

	if err := toggleBroadcastPresetButton(state, broadcastPresetMiniApp); err != nil {
		t.Fatalf("second toggleBroadcastPresetButton(miniapp) error = %v", err)
	}
	if isBroadcastPresetSelected(state, broadcastPresetMiniApp) {
		t.Fatal("miniapp preset should be deselected after second toggle")
	}
	if len(state.Buttons) != 1 || broadcastButtonCallbackData(state.Buttons[0]) != broadcastGuestAttendanceConfirm {
		t.Fatalf("state.Buttons after removing miniapp = %#v, want only confirm button", state.Buttons)
	}
}

func TestBroadcastRenderedTextAppendsQuestionFooterOnce(t *testing.T) {
	t.Parallel()

	state := &BroadcastState{
		Text:                  "Напишите нам, если что-то понадобится",
		SelectedPresetButtons: map[string]bool{broadcastPresetQuestion: true},
	}

	got := broadcastRenderedText(state)
	want := "Напишите нам, если что-то понадобится\n\n" + broadcastQuestionFooter
	if got != want {
		t.Fatalf("broadcastRenderedText() = %q, want %q", got, want)
	}

	state.Text = want
	if got := broadcastRenderedText(state); got != want {
		t.Fatalf("broadcastRenderedText() duplicated footer = %q, want %q", got, want)
	}
}

func TestBroadcastGroupLinkURLFallback(t *testing.T) {
	t.Parallel()

	oldGroupLink := config.GroupLink
	oldGroupID := config.GroupID
	t.Cleanup(func() {
		config.GroupLink = oldGroupLink
		config.GroupID = oldGroupID
	})

	config.GroupLink = ""
	config.GroupID = "@wedding_group"

	if got := broadcastGroupLinkURL(); got != "https://t.me/wedding_group" {
		t.Fatalf("broadcastGroupLinkURL() = %q, want https://t.me/wedding_group", got)
	}
}
