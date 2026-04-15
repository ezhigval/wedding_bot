package bot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const maxTelegramPhotoSize = 10 * 1024 * 1024
const maxTelegramVideoSize = 50 * 1024 * 1024

func downloadTelegramPhoto(ctx context.Context, bot *tgbotapi.BotAPI, fileID string) ([]byte, string, error) {
	return downloadTelegramMedia(ctx, bot, fileID, "image/jpeg", maxTelegramPhotoSize, "image")
}

func downloadTelegramVideo(ctx context.Context, bot *tgbotapi.BotAPI, fileID, declaredMimeType string) ([]byte, string, error) {
	fallbackMimeType := strings.ToLower(strings.TrimSpace(declaredMimeType))
	if !strings.HasPrefix(fallbackMimeType, "video/") {
		fallbackMimeType = "video/mp4"
	}

	return downloadTelegramMedia(ctx, bot, fileID, fallbackMimeType, maxTelegramVideoSize, "video")
}

func downloadTelegramMedia(ctx context.Context, bot *tgbotapi.BotAPI, fileID, fallbackMimeType string, maxSize int64, expectedKind string) ([]byte, string, error) {
	fileURL, err := bot.GetFileDirectURL(fileID)
	if err != nil {
		return nil, "", fmt.Errorf("не удалось получить ссылку на медиафайл Telegram: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("не удалось создать запрос к Telegram: %w", err)
	}

	client := bot.Client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("не удалось скачать медиафайл из Telegram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Telegram вернул неожиданный статус %d", resp.StatusCode)
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
	if err != nil {
		return nil, "", fmt.Errorf("не удалось прочитать медиафайл из Telegram: %w", err)
	}
	if len(content) == 0 {
		return nil, "", fmt.Errorf("Telegram вернул пустой медиафайл")
	}
	if int64(len(content)) > maxSize {
		return nil, "", fmt.Errorf("размер медиафайла превышает %d байт", maxSize)
	}

	mimeType := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
	if !strings.HasPrefix(mimeType, expectedKind+"/") {
		mimeType = strings.ToLower(http.DetectContentType(content))
	}
	if !strings.HasPrefix(mimeType, expectedKind+"/") {
		mimeType = fallbackMimeType
	}
	if !strings.HasPrefix(mimeType, expectedKind+"/") {
		return nil, "", fmt.Errorf("Telegram прислал неподдерживаемый формат: %s", mimeType)
	}

	return content, mimeType, nil
}
