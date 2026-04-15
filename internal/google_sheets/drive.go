package google_sheets

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"wedding-bot/internal/config"
)

var driveService *drive.Service

var ErrGoogleDriveNotConfigured = errors.New("google drive folder is not configured")

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

	credsJSON, err := getCredentialsJSON()
	if err != nil {
		return nil, fmt.Errorf("credentials не установлены: %w", err)
	}

	creds, err := google.CredentialsFromJSON(
		context.Background(),
		credsJSON,
		drive.DriveScope,
	)
	if err != nil {
		driveService = nil
		return nil, fmt.Errorf("ошибка создания Drive credentials: %w", err)
	}

	service, err := drive.NewService(
		context.Background(),
		option.WithCredentials(creds),
	)
	if err != nil {
		driveService = nil
		return nil, fmt.Errorf("ошибка создания сервиса Google Drive: %w", err)
	}

	driveService = service
	return driveService, nil
}

// UploadPhotoToDrive загружает фото в заранее настроенную папку Google Drive.
func UploadPhotoToDrive(ctx context.Context, params DriveUploadParams) (*DriveUploadResult, error) {
	if config.GoogleDriveFolderID == "" {
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
		Parents:     []string{config.GoogleDriveFolderID},
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
