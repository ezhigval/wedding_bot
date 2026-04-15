package google_sheets

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

const photoSheetName = "Фото"

type PhotoSource string

const (
	PhotoSourceTelegramChat  PhotoSource = "telegram_chat"
	PhotoSourceWebappCamera  PhotoSource = "webapp_camera"
	PhotoSourceWebappGallery PhotoSource = "webapp_gallery"
)

type PhotoSaveInput struct {
	UserID      int
	Username    *string
	FullName    string
	Source      PhotoSource
	OriginalRef string
	MimeType    string
	Content     []byte
}

// NormalizePhotoSource приводит source к поддерживаемому значению.
func NormalizePhotoSource(raw string) PhotoSource {
	switch strings.TrimSpace(raw) {
	case string(PhotoSourceTelegramChat):
		return PhotoSourceTelegramChat
	case string(PhotoSourceWebappCamera):
		return PhotoSourceWebappCamera
	case string(PhotoSourceWebappGallery):
		return PhotoSourceWebappGallery
	default:
		return PhotoSourceWebappGallery
	}
}

// SavePhotoFromUser сохраняет фото, присланное гостем в Telegram-чат.
func SavePhotoFromUser(ctx context.Context, userID int, username *string, fullName, fileID, mimeType string, content []byte) error {
	return savePhoto(ctx, PhotoSaveInput{
		UserID:      userID,
		Username:    username,
		FullName:    fullName,
		Source:      PhotoSourceTelegramChat,
		OriginalRef: strings.TrimSpace(fileID),
		MimeType:    mimeType,
		Content:     content,
	})
}

// SavePhotoFromWebapp сохраняет фото, присланное через Mini App.
func SavePhotoFromWebapp(ctx context.Context, userID int, username *string, fullName, fileName, mimeType string, content []byte, source PhotoSource) error {
	return savePhoto(ctx, PhotoSaveInput{
		UserID:      userID,
		Username:    username,
		FullName:    fullName,
		Source:      NormalizePhotoSource(string(source)),
		OriginalRef: strings.TrimSpace(fileName),
		MimeType:    mimeType,
		Content:     content,
	})
}

func savePhoto(ctx context.Context, input PhotoSaveInput) error {
	if err := EnsureRequiredSheets(ctx); err != nil {
		return err
	}
	if len(input.Content) == 0 {
		return fmt.Errorf("пустое содержимое фото")
	}

	mimeType := strings.TrimSpace(input.MimeType)
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	if !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return fmt.Errorf("неподдерживаемый mime type: %s", mimeType)
	}

	now := time.Now()
	usernameStr := normalizedPhotoUsername(input.Username)
	resolvedFullName := resolvePhotoFullName(ctx, input.UserID, usernameStr, input.FullName)
	fileName := buildPhotoFileName(input.Source, input.UserID, mimeType, input.OriginalRef, now)
	description := buildPhotoDescription(input.Source, input.UserID, usernameStr, resolvedFullName)

	uploaded, err := UploadPhotoToDrive(ctx, DriveUploadParams{
		FileName:    fileName,
		MimeType:    mimeType,
		Description: description,
		Content:     input.Content,
	})
	if err != nil {
		return err
	}

	if err := appendPhotoRow(ctx, photoSheetRow{
		Timestamp:   now.Format("2006-01-02 15:04:05"),
		UserID:      fmt.Sprintf("%d", input.UserID),
		Username:    usernameStr,
		FullName:    resolvedFullName,
		Source:      string(input.Source),
		DriveFileID: uploaded.FileID,
		DriveURL:    uploaded.WebViewLink,
		FileName:    uploaded.Name,
		MimeType:    uploaded.MimeType,
		FileSize:    fmt.Sprintf("%d", uploaded.Size),
		OriginalRef: input.OriginalRef,
	}); err != nil {
		return err
	}

	log.Printf(
		"Сохранено фото: source=%s, user_id=%d, username=%s, drive_file_id=%s",
		input.Source,
		input.UserID,
		usernameStr,
		uploaded.FileID,
	)
	return nil
}

type photoSheetRow struct {
	Timestamp   string
	UserID      string
	Username    string
	FullName    string
	Source      string
	DriveFileID string
	DriveURL    string
	FileName    string
	MimeType    string
	FileSize    string
	OriginalRef string
}

func appendPhotoRow(ctx context.Context, row photoSheetRow) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{
			row.Timestamp,
			row.UserID,
			row.Username,
			row.FullName,
			row.Source,
			row.DriveFileID,
			row.DriveURL,
			row.FileName,
			row.MimeType,
			row.FileSize,
			row.OriginalRef,
		}},
	}

	readRange := fmt.Sprintf("%s!A:Z", photoSheetName)
	_, err = service.Spreadsheets.Values.Append(
		config.GoogleSheetsID,
		readRange,
		valueRange,
	).ValueInputOption("USER_ENTERED").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("ошибка добавления строки фото: %w", err)
	}

	return nil
}

func normalizedPhotoUsername(username *string) string {
	if username == nil {
		return ""
	}
	return NormalizeTelegramUsername(*username)
}

func resolvePhotoFullName(ctx context.Context, userID int, username, fallback string) string {
	cleanFallback := strings.TrimSpace(fallback)
	if cleanFallback != "" {
		return cleanFallback
	}

	guest, err := FindGuestByIdentifier(ctx, userID, username)
	if err == nil && guest != nil {
		if fullName := buildGuestFullName(guest.FirstName, guest.LastName); fullName != "" {
			return fullName
		}
	}

	if username != "" {
		return "@" + username
	}
	if userID > 0 {
		return fmt.Sprintf("user_%d", userID)
	}
	return "unknown"
}

func buildPhotoDescription(source PhotoSource, userID int, username, fullName string) string {
	parts := []string{
		"Свадебный альбом",
		fmt.Sprintf("source=%s", source),
	}
	if userID > 0 {
		parts = append(parts, fmt.Sprintf("user_id=%d", userID))
	}
	if username != "" {
		parts = append(parts, fmt.Sprintf("username=%s", username))
	}
	if fullName != "" {
		parts = append(parts, fmt.Sprintf("full_name=%s", fullName))
	}
	return strings.Join(parts, "; ")
}

func buildPhotoFileName(source PhotoSource, userID int, mimeType, originalRef string, now time.Time) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(originalRef)))
	if ext == "" {
		ext = extensionForMimeType(mimeType)
	}
	if ext == "" {
		ext = ".jpg"
	}

	userPart := "guest"
	if userID > 0 {
		userPart = fmt.Sprintf("%d", userID)
	}

	return fmt.Sprintf("%s-%s-%s%s", source, userPart, now.Format("20060102-150405"), ext)
}

func extensionForMimeType(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/heic":
		return ".heic"
	case "image/heif":
		return ".heif"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}
