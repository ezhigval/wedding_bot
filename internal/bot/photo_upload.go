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

func downloadTelegramPhoto(ctx context.Context, bot *tgbotapi.BotAPI, fileID string) ([]byte, string, error) {
	fileURL, err := bot.GetFileDirectURL(fileID)
	if err != nil {
		return nil, "", fmt.Errorf("не удалось получить ссылку на файл Telegram: %w", err)
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
		return nil, "", fmt.Errorf("не удалось скачать фото из Telegram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Telegram вернул неожиданный статус %d", resp.StatusCode)
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, maxTelegramPhotoSize+1))
	if err != nil {
		return nil, "", fmt.Errorf("не удалось прочитать фото из Telegram: %w", err)
	}
	if len(content) == 0 {
		return nil, "", fmt.Errorf("Telegram вернул пустое фото")
	}
	if len(content) > maxTelegramPhotoSize {
		return nil, "", fmt.Errorf("размер фото превышает %d байт", maxTelegramPhotoSize)
	}

	mimeType := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = strings.ToLower(http.DetectContentType(content))
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, "", fmt.Errorf("Telegram прислал неподдерживаемый формат: %s", mimeType)
	}

	return content, mimeType, nil
}
