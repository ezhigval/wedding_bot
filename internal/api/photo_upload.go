package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"

	"wedding-bot/internal/google_sheets"
)

const (
	maxImageUploadSize    = 10 * 1024 * 1024
	maxVideoUploadSize    = 50 * 1024 * 1024
	maxMediaUploadSize    = maxVideoUploadSize
	maxMediaRequestSize   = 52 * 1024 * 1024
	multipartMemoryBuffer = 12 * 1024 * 1024
)

var (
	errPhotoRequired = errors.New("photo_required")
	errPhotoTooLarge = errors.New("photo_too_large")
	errInvalidPhoto  = errors.New("invalid_photo")
)

type uploadPhotoRequest struct {
	UserID    int    `json:"userId"`
	Username  string `json:"username"`
	FullName  string `json:"fullName"`
	FileName  string `json:"fileName"`
	PhotoData string `json:"photoData"`
	MediaData string `json:"mediaData"`
	InitData  string `json:"initData"`
	Source    string `json:"source"`
}

// uploadPhoto загружает фото или видео из Mini App.
// Поддерживает multipart/form-data и legacy JSON с data URL/base64.
func uploadPhoto(w http.ResponseWriter, r *http.Request) {
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "multipart/form-data") {
		uploadPhotoMultipart(w, r)
		return
	}

	uploadPhotoJSON(w, r)
}

func uploadPhotoMultipart(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxMediaRequestSize)
	if err := r.ParseMultipartForm(multipartMemoryBuffer); err != nil {
		JSONError(w, http.StatusBadRequest, "photo_too_large")
		return
	}

	userID, err := parseOptionalUserID(r.FormValue("userId"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid_user_id")
		return
	}

	userID, username, ok := requireResolvedAuthIdentity(
		w,
		r,
		userID,
		r.FormValue("username"),
		r.FormValue("initData"),
	)
	if !ok {
		return
	}

	file, header, err := readMultipartMedia(r)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "photo_required")
		return
	}
	defer file.Close()

	content, mimeType, err := readUploadedMedia(file, header.Header.Get("Content-Type"))
	if err != nil {
		handlePhotoUploadError(w, err)
		return
	}

	var usernamePtr *string
	if username != "" {
		usernamePtr = &username
	}

	originalRef := strings.TrimSpace(header.Filename)
	if originalRef == "" {
		originalRef = strings.TrimSpace(r.FormValue("fileName"))
	}

	if err := google_sheets.SaveMediaFromWebapp(
		r.Context(),
		userID,
		usernamePtr,
		r.FormValue("fullName"),
		originalRef,
		mimeType,
		content,
		r.FormValue("source"),
	); err != nil {
		handlePhotoStorageError(w, err)
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func uploadPhotoJSON(w http.ResponseWriter, r *http.Request) {
	var req uploadPhotoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	userID, username, ok := requireResolvedAuthIdentity(w, r, req.UserID, req.Username, req.InitData)
	if !ok {
		return
	}

	content, mimeType, err := decodeBase64MediaData(firstNonEmptyMediaPayload(req.MediaData, req.PhotoData))
	if err != nil {
		handlePhotoUploadError(w, err)
		return
	}

	var usernamePtr *string
	if username != "" {
		usernamePtr = &username
	}

	if err := google_sheets.SaveMediaFromWebapp(
		r.Context(),
		userID,
		usernamePtr,
		req.FullName,
		req.FileName,
		mimeType,
		content,
		req.Source,
	); err != nil {
		handlePhotoStorageError(w, err)
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func readMultipartMedia(r *http.Request) (multipartFile multipart.File, header *multipart.FileHeader, err error) {
	file, fileHeader, fileErr := r.FormFile("media")
	if fileErr == nil {
		return file, fileHeader, nil
	}

	return r.FormFile("photo")
}

func firstNonEmptyMediaPayload(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func handlePhotoUploadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errPhotoRequired):
		JSONError(w, http.StatusBadRequest, "photo_required")
	case errors.Is(err, errPhotoTooLarge):
		JSONError(w, http.StatusBadRequest, "photo_too_large")
	default:
		JSONError(w, http.StatusBadRequest, "invalid_photo")
	}
}

func handlePhotoStorageError(w http.ResponseWriter, err error) {
	if errors.Is(err, google_sheets.ErrGoogleDriveNotConfigured) {
		JSONError(w, http.StatusServiceUnavailable, "photo_storage_not_configured")
		return
	}

	log.Printf("Error saving media: %v", err)
	JSONError(w, http.StatusInternalServerError, "failed to save photo")
}

func readUploadedMedia(reader io.Reader, declaredMimeType string) ([]byte, string, error) {
	limited := io.LimitReader(reader, maxMediaUploadSize+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", errInvalidPhoto
	}

	return normalizeMediaPayload(content, declaredMimeType)
}

func decodeBase64PhotoData(raw string) ([]byte, string, error) {
	return decodeBase64MediaData(raw)
}

func decodeBase64MediaData(raw string) ([]byte, string, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil, "", errPhotoRequired
	}

	mimeType := ""
	if strings.HasPrefix(clean, "data:") {
		commaIndex := strings.Index(clean, ",")
		if commaIndex <= 5 {
			return nil, "", errInvalidPhoto
		}

		meta := clean[5:commaIndex]
		if !strings.Contains(meta, ";base64") {
			return nil, "", errInvalidPhoto
		}

		mimeType = strings.TrimSpace(strings.TrimSuffix(meta, ";base64"))
		clean = clean[commaIndex+1:]
	}

	decoded, err := decodeBase64String(clean)
	if err != nil {
		return nil, "", errInvalidPhoto
	}

	return normalizeMediaPayload(decoded, mimeType)
}

func decodeBase64String(raw string) ([]byte, error) {
	candidates := []struct {
		fn func(string) ([]byte, error)
	}{
		{fn: base64.StdEncoding.DecodeString},
		{fn: base64.RawStdEncoding.DecodeString},
		{fn: base64.URLEncoding.DecodeString},
		{fn: base64.RawURLEncoding.DecodeString},
	}

	for _, candidate := range candidates {
		decoded, err := candidate.fn(raw)
		if err == nil {
			return decoded, nil
		}
	}

	return nil, errInvalidPhoto
}

func normalizeImagePayload(content []byte, declaredMimeType string) ([]byte, string, error) {
	return normalizeMediaPayload(content, declaredMimeType)
}

func normalizeMediaPayload(content []byte, declaredMimeType string) ([]byte, string, error) {
	if len(content) == 0 {
		return nil, "", errPhotoRequired
	}
	if len(content) > maxMediaUploadSize {
		return nil, "", errPhotoTooLarge
	}

	mimeType := strings.ToLower(strings.TrimSpace(declaredMimeType))
	sniffedMimeType := strings.ToLower(http.DetectContentType(content))
	resolvedMimeType := resolveMediaMimeType(mimeType, sniffedMimeType)
	if resolvedMimeType == "" {
		return nil, "", errInvalidPhoto
	}

	if len(content) > maxAllowedUploadSize(resolvedMimeType) {
		return nil, "", errPhotoTooLarge
	}

	return content, resolvedMimeType, nil
}

func resolveMediaMimeType(declaredMimeType, sniffedMimeType string) string {
	switch {
	case strings.HasPrefix(declaredMimeType, "image/"), strings.HasPrefix(declaredMimeType, "video/"):
		return declaredMimeType
	case strings.HasPrefix(sniffedMimeType, "image/"), strings.HasPrefix(sniffedMimeType, "video/"):
		return sniffedMimeType
	default:
		return ""
	}
}

func maxAllowedUploadSize(mimeType string) int {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return maxImageUploadSize
	case strings.HasPrefix(mimeType, "video/"):
		return maxVideoUploadSize
	default:
		return maxImageUploadSize
	}
}
