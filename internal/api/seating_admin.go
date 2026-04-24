package api

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"wedding-bot/internal/config"
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
	log.Printf("handleSeatingOnEdit: backend sync disabled, sheet=%s", sheetName)
	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"status": "disabled",
		"reason": "moved_to_apps_script",
		"sheet":  sheetName,
	})
}

func handleSeatingFullReconcile(w http.ResponseWriter, r *http.Request) {
	if !requireSeatingAPIToken(w, r) {
		return
	}

	log.Printf("handleSeatingFullReconcile: backend sync disabled")
	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":      "disabled",
		"reason":      "moved_to_apps_script",
		"description": "full_reconcile_disabled",
	})
}

func handleSeatingRebuildHeader(w http.ResponseWriter, r *http.Request) {
	if !requireSeatingAPIToken(w, r) {
		return
	}

	log.Printf("handleSeatingRebuildHeader: backend sync disabled")
	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"status": "disabled",
		"reason": "moved_to_apps_script",
	})
}

func handleSeatingSyncFromSeating(w http.ResponseWriter, r *http.Request) {
	if !requireSeatingAPIToken(w, r) {
		return
	}

	log.Printf("handleSeatingSyncFromSeating: backend sync disabled")
	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"status": "disabled",
		"reason": "moved_to_apps_script",
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
