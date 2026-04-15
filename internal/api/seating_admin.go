package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"wedding-bot/internal/config"
	"wedding-bot/internal/google_sheets"
)

type seatingEditRequest struct {
	Event     string `json:"event"`
	SheetName string `json:"sheetName"`
	RowStart  int    `json:"rowStart"`
	ColStart  int    `json:"colStart"`
	NumRows   int    `json:"numRows"`
	NumCols   int    `json:"numCols"`
	RangeA1   string `json:"rangeA1"`
}

func requireSeatingAPIToken(w http.ResponseWriter, r *http.Request) bool {
	expectedToken := strings.TrimSpace(config.SeatingAPIToken)
	if expectedToken == "" {
		return true
	}

	providedToken := strings.TrimSpace(r.Header.Get("X-Api-Token"))
	if providedToken == "" {
		JSONError(w, http.StatusUnauthorized, "invalid_api_token")
		return false
	}

	if subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) != 1 {
		JSONError(w, http.StatusUnauthorized, "invalid_api_token")
		return false
	}

	return true
}

func handleSeatingOnEdit(w http.ResponseWriter, r *http.Request) {
	if !requireSeatingAPIToken(w, r) {
		return
	}

	var req seatingEditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	sheetName := strings.TrimSpace(req.SheetName)
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch sheetName {
	case config.GoogleSheetsSheetName, "Список гостей":
		report, err := google_sheets.SyncSeatingFromGuestList(ctx)
		if err != nil {
			log.Printf("handleSeatingOnEdit guests->seating error: %v", err)
			JSONError(w, http.StatusInternalServerError, "server_error")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"direction": "guests_to_seating",
			"report":    report,
		})
	case "Рассадка":
		report, err := google_sheets.SyncGuestListTablesFromSeating(ctx)
		if err != nil {
			log.Printf("handleSeatingOnEdit seating->guests error: %v", err)
			JSONError(w, http.StatusInternalServerError, "server_error")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"direction": "seating_to_guests",
			"report":    report,
		})
	default:
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"status": "ignored",
			"reason": "irrelevant_sheet",
			"sheet":  sheetName,
		})
	}
}

func handleSeatingFullReconcile(w http.ResponseWriter, r *http.Request) {
	if !requireSeatingAPIToken(w, r) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	fromSeatingReport, err := google_sheets.SyncGuestListTablesFromSeating(ctx)
	if err != nil {
		log.Printf("handleSeatingFullReconcile seating->guests error: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	fromGuestsReport, err := google_sheets.SyncSeatingFromGuestList(ctx)
	if err != nil {
		log.Printf("handleSeatingFullReconcile guests->seating error: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":       "ok",
		"seating_sync": fromSeatingReport,
		"guests_sync":  fromGuestsReport,
		"description":  "full_reconcile_completed",
	})
}

func handleSeatingRebuildHeader(w http.ResponseWriter, r *http.Request) {
	if !requireSeatingAPIToken(w, r) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	report, err := google_sheets.SyncSeatingFromGuestList(ctx)
	if err != nil {
		log.Printf("handleSeatingRebuildHeader error: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"report": report,
	})
}

func handleSeatingSyncFromSeating(w http.ResponseWriter, r *http.Request) {
	if !requireSeatingAPIToken(w, r) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	report, err := google_sheets.SyncGuestListTablesFromSeating(ctx)
	if err != nil {
		log.Printf("handleSeatingSyncFromSeating error: %v", err)
		JSONError(w, http.StatusInternalServerError, "server_error")
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"report": report,
	})
}

func handlePingFromSheets(w http.ResponseWriter, r *http.Request) {
	if !requireSeatingAPIToken(w, r) {
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}
