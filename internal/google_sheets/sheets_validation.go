package google_sheets

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"wedding-bot/internal/config"

	"google.golang.org/api/sheets/v4"
)

type managedSheetHeaderSpec struct {
	canonical string
	aliases   []string
}

type managedSheetSpec struct {
	sheetName string
	headers   []managedSheetHeaderSpec
}

// ValidateCoreSheetsStructure проверяет и при необходимости восстанавливает
// заголовки служебных листов, которыми управляет само приложение.
func ValidateCoreSheetsStructure(ctx context.Context) error {
	service, err := GetGoogleSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := config.GoogleSheetsID
	specs := []managedSheetSpec{
		{
			sheetName: "Игры",
			headers: []managedSheetHeaderSpec{
				{canonical: "user_id", aliases: []string{"user_id", "userid", "telegram_user_id"}},
				{canonical: "first_name", aliases: []string{"first_name", "firstname", "имя"}},
				{canonical: "last_name", aliases: []string{"last_name", "lastname", "фамилия"}},
				{canonical: "total_score", aliases: []string{"total_score", "totalscore", "общий_счет"}},
				{canonical: "dragon_score", aliases: []string{"dragon_score", "dragonscore"}},
				{canonical: "flappy_score", aliases: []string{"flappy_score", "flappyscore"}},
				{canonical: "crossword_score", aliases: []string{"crossword_score", "crosswordscore"}},
				{canonical: "wordle_score", aliases: []string{"wordle_score", "wordlescore"}},
				{canonical: "rank", aliases: []string{"rank", "звание"}},
				{canonical: "last_updated", aliases: []string{"last_updated", "lastupdated", "updated_at"}},
			},
		},
		{
			sheetName: "Wordle",
			headers: []managedSheetHeaderSpec{
				{canonical: "слово", aliases: []string{"слово", "word"}},
			},
		},
		{
			sheetName: "Wordle_Прогресс",
			headers: []managedSheetHeaderSpec{
				{canonical: "user_id", aliases: []string{"user_id", "userid"}},
				{canonical: "отгаданные_слова", aliases: []string{"отгаданные_слова", "guessed_words"}},
			},
		},
		{
			sheetName: "Wordle_Состояние",
			headers: []managedSheetHeaderSpec{
				{canonical: "user_id", aliases: []string{"user_id", "userid"}},
				{canonical: "current_word", aliases: []string{"current_word", "word"}},
				{canonical: "attempts", aliases: []string{"attempts"}},
				{canonical: "current_guess", aliases: []string{"current_guess"}},
				{canonical: "last_word_date", aliases: []string{"last_word_date"}},
			},
		},
		{
			sheetName: "Кроссворд_Прогресс",
			headers: []managedSheetHeaderSpec{
				{canonical: "user_id", aliases: []string{"user_id", "userid"}},
				{canonical: "current_crossword_index", aliases: []string{"current_crossword_index", "crossword_index"}},
				{canonical: "guessed_words_json", aliases: []string{"guessed_words_json", "progress_json"}},
				{canonical: "crossword_start_date", aliases: []string{"crossword_start_date", "start_date"}},
			},
		},
		{
			sheetName: "Фото",
			headers: []managedSheetHeaderSpec{
				{canonical: "timestamp", aliases: []string{"timestamp", "created_at"}},
				{canonical: "user_id", aliases: []string{"user_id", "userid"}},
				{canonical: "username", aliases: []string{"username", "telegram_username"}},
				{canonical: "full_name", aliases: []string{"full_name", "fullname", "fio"}},
				{canonical: "photo_data", aliases: []string{"photo_data", "file_id", "payload"}},
			},
		},
	}

	var validationErrors []string
	for _, spec := range specs {
		if err := validateManagedSheetHeaders(service, spreadsheetID, spec); err != nil {
			validationErrors = append(validationErrors, err.Error())
		}
	}

	if len(validationErrors) > 0 {
		return errors.New(strings.Join(validationErrors, "; "))
	}

	return nil
}

func validateManagedSheetHeaders(service *sheets.Service, spreadsheetID string, spec managedSheetSpec) error {
	if err := EnsureSheetExists(spreadsheetID, spec.sheetName); err != nil {
		return fmt.Errorf("лист '%s' недоступен: %w", spec.sheetName, err)
	}

	lastColumn := getColumnLetter(len(spec.headers))
	readRange := fmt.Sprintf("%s!A1:%s1", spec.sheetName, lastColumn)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения заголовков листа '%s': %w", spec.sheetName, err)
	}

	headers := make([]string, 0, len(spec.headers))
	if len(resp.Values) > 0 {
		for _, raw := range resp.Values[0] {
			headers = append(headers, strings.TrimSpace(fmt.Sprintf("%v", raw)))
		}
	}

	needsRepair := len(headers) < len(spec.headers)
	if !needsRepair {
		for idx, headerSpec := range spec.headers {
			if idx >= len(headers) || !headerMatchesAnyAlias(headers[idx], headerSpec.aliases) {
				needsRepair = true
				break
			}
		}
	}

	if !needsRepair {
		log.Printf("✅ Служебный лист '%s' проверен", spec.sheetName)
		return nil
	}

	canonicalHeaders := make([]interface{}, len(spec.headers))
	for idx, headerSpec := range spec.headers {
		canonicalHeaders[idx] = headerSpec.canonical
	}

	if err := UpdateSheetValues(spreadsheetID, spec.sheetName, fmt.Sprintf("A1:%s1", lastColumn), [][]interface{}{canonicalHeaders}); err != nil {
		return fmt.Errorf("ошибка восстановления заголовков листа '%s': %w", spec.sheetName, err)
	}

	log.Printf("✅ Заголовки служебного листа '%s' восстановлены", spec.sheetName)
	return nil
}
