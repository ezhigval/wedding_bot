package google_sheets

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"wedding-bot/internal/config"
)

var driveService *drive.Service

var ErrGoogleDriveNotConfigured = errors.New("google drive folder is not configured")
var ErrGoogleDriveOAuthIncomplete = errors.New("google drive oauth is configured incompletely")

type DriveUploadParams struct {
	FileName    string
	MimeType    string
	Description string
	Content     []byte
}

type DriveUploadResult struct {
	FileID      string
	WebViewLink string
	Name        string
	MimeType    string
	Size        int64
}

// GetGoogleDriveClient получает клиент Google Drive.
func GetGoogleDriveClient() (*drive.Service, error) {
	if driveService != nil {
		return driveService, nil
	}

	service, err := newGoogleDriveService(context.Background())
	if err != nil {
		driveService = nil
		return nil, err
	}

	driveService = service
	return driveService, nil
}

func newGoogleDriveService(ctx context.Context) (*drive.Service, error) {
	if hasCompleteGoogleDriveOAuthConfig() {
		return newGoogleDriveServiceFromOAuth(ctx)
	}

	if hasPartialGoogleDriveOAuthConfig() {
		return nil, fmt.Errorf("%w: заполните %s", ErrGoogleDriveOAuthIncomplete, strings.Join(missingGoogleDriveOAuthFields(), ", "))
	}

	return newGoogleDriveServiceFromServiceAccount(ctx)
}

func newGoogleDriveServiceFromOAuth(ctx context.Context) (*drive.Service, error) {
	tokenSource := (&oauth2.Config{
		ClientID:     config.GoogleDriveOAuthClientID,
		ClientSecret: config.GoogleDriveOAuthClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{drive.DriveScope},
	}).TokenSource(ctx, &oauth2.Token{
		RefreshToken: config.GoogleDriveOAuthRefreshToken,
	})

	service, err := drive.NewService(
		ctx,
		option.WithTokenSource(tokenSource),
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания OAuth-сервиса Google Drive: %w", err)
	}

	log.Printf("✅ Google Drive инициализирован через OAuth пользователя")
	return service, nil
}

func newGoogleDriveServiceFromServiceAccount(ctx context.Context) (*drive.Service, error) {
	credsJSON, err := getCredentialsJSON()
	if err != nil {
		return nil, fmt.Errorf("credentials не установлены: %w", err)
	}

	creds, err := google.CredentialsFromJSON(
		ctx,
		credsJSON,
		drive.DriveScope,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания Drive credentials: %w", err)
	}

	service, err := drive.NewService(
		ctx,
		option.WithCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сервиса Google Drive: %w", err)
	}

	log.Printf("✅ Google Drive инициализирован через service account")
	return service, nil
}

func hasCompleteGoogleDriveOAuthConfig() bool {
	return config.GoogleDriveOAuthClientID != "" &&
		config.GoogleDriveOAuthClientSecret != "" &&
		config.GoogleDriveOAuthRefreshToken != ""
}

func hasPartialGoogleDriveOAuthConfig() bool {
	return config.GoogleDriveOAuthClientID != "" ||
		config.GoogleDriveOAuthClientSecret != "" ||
		config.GoogleDriveOAuthRefreshToken != ""
}

func missingGoogleDriveOAuthFields() []string {
	missing := make([]string, 0, 3)
	if config.GoogleDriveOAuthClientID == "" {
		missing = append(missing, "GOOGLE_DRIVE_OAUTH_CLIENT_ID")
	}
	if config.GoogleDriveOAuthClientSecret == "" {
		missing = append(missing, "GOOGLE_DRIVE_OAUTH_CLIENT_SECRET")
	}
	if config.GoogleDriveOAuthRefreshToken == "" {
		missing = append(missing, "GOOGLE_DRIVE_OAUTH_REFRESH_TOKEN")
	}
	return missing
}

// UploadPhotoToDrive загружает фото в заранее настроенную папку Google Drive.
func UploadPhotoToDrive(ctx context.Context, params DriveUploadParams) (*DriveUploadResult, error) {
	folderID := normalizeGoogleDriveFolderID(config.GoogleDriveFolderID)
	if folderID == "" {
		return nil, fmt.Errorf("%w: GOOGLE_DRIVE_FOLDER_ID не установлен", ErrGoogleDriveNotConfigured)
	}
	if len(params.Content) == 0 {
		return nil, fmt.Errorf("пустое содержимое файла")
	}

	service, err := GetGoogleDriveClient()
	if err != nil {
		return nil, err
	}

	file := &drive.File{
		Name:        params.FileName,
		Description: params.Description,
		MimeType:    params.MimeType,
		Parents:     []string{folderID},
	}

	created, err := service.Files.Create(file).
		Media(bytes.NewReader(params.Content), googleapi.ContentType(params.MimeType)).
		Fields("id,webViewLink,name,mimeType,size").
		SupportsAllDrives(true).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки файла в Google Drive: %w", err)
	}

	log.Printf("📸 Фото загружено в Google Drive: file_id=%s, name=%s", created.Id, created.Name)

	return &DriveUploadResult{
		FileID:      created.Id,
		WebViewLink: created.WebViewLink,
		Name:        created.Name,
		MimeType:    created.MimeType,
		Size:        created.Size,
	}, nil
}

func normalizeGoogleDriveFolderID(raw string) string {
	clean := strings.TrimSpace(strings.Trim(raw, `"'`))
	if clean == "" {
		return ""
	}

	if !looksLikeGoogleDriveURL(clean) {
		return clean
	}

	candidate := clean
	if !strings.Contains(candidate, "://") {
		candidate = "https://" + candidate
	}

	parsed, err := url.Parse(candidate)
	if err != nil {
		return ""
	}

	return extractGoogleDriveFolderID(parsed)
}

func looksLikeGoogleDriveURL(raw string) bool {
	return strings.Contains(raw, "://") ||
		strings.HasPrefix(raw, "drive.google.com/") ||
		strings.HasPrefix(raw, "docs.google.com/")
}

func extractGoogleDriveFolderID(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}

	if id := strings.TrimSpace(strings.Trim(parsed.Query().Get("id"), `"'`)); id != "" {
		return id
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := 0; i < len(segments)-1; i++ {
		switch segments[i] {
		case "folders", "d":
			if id := strings.TrimSpace(strings.Trim(segments[i+1], `"'`)); id != "" {
				return id
			}
		}
	}

	return ""
}
