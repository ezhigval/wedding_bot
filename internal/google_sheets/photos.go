package google_sheets

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

// SavePhotoFromUser сохраняет информацию о фото, присланном гостем, в лист 'Фото'
func SavePhotoFromUser(ctx context.Context, userID int, username *string, fullName string, fileID string) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Фото"

	if err := EnsureRequiredSheets(ctx); err != nil {
		return err
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	usernameStr := ""
	if username != nil {
		usernameStr = strings.TrimPrefix(*username, "@")
	}

	rowData := []interface{}{
		timestamp,
		fmt.Sprintf("%d", userID),
		usernameStr,
		fullName,
		fileID,
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{rowData},
	}

	readRange := fmt.Sprintf("%s!A:Z", sheetName)
	_, err = service.Spreadsheets.Values.Append(
		spreadsheetID,
		readRange,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("ошибка добавления: %w", err)
	}

	log.Printf("Сохранено фото от user_id=%d, username=%s, file_id=%s", userID, usernameStr, fileID)
	return nil
}

// SavePhotoFromWebapp сохраняет информацию о фото из веб-приложения в лист 'Фото'
func SavePhotoFromWebapp(ctx context.Context, userID int, username *string, fullName string, photoData string) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	sheetName := "Фото"

	if err := EnsureRequiredSheets(ctx); err != nil {
		return err
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	usernameStr := ""
	if username != nil {
		usernameStr = strings.TrimPrefix(*username, "@")
	}

	// Ограничиваем длину photo_data
	if len(photoData) > 500 {
		photoData = photoData[:500]
	}

	rowData := []interface{}{
		timestamp,
		fmt.Sprintf("%d", userID),
		usernameStr,
		fullName,
		photoData,
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{rowData},
	}

	readRange := fmt.Sprintf("%s!A:Z", sheetName)
	_, err = service.Spreadsheets.Values.Append(
		spreadsheetID,
		readRange,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("ошибка добавления: %w", err)
	}

	log.Printf("Сохранено фото из веб-приложения от user_id=%d, username=%s", userID, usernameStr)
	return nil
}

