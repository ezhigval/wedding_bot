package daily_reset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"wedding-bot/internal/config"
	"wedding-bot/internal/google_sheets"
)

const (
	wordlePartialPoints    = 3
	crosswordPartialPoints = 13

	wordleAwardDateKey  = "WORDLE_PARTIAL_AWARD_DATE"
	wordleAwardUsersKey = "WORDLE_PARTIAL_AWARDED_USERS"

	crosswordAwardDateKey  = "CROSSWORD_PARTIAL_AWARD_DATE"
	crosswordAwardUsersKey = "CROSSWORD_PARTIAL_AWARDED_USERS"
)

type wordleResetCandidate struct {
	UserID       int
	LastWordDate string
	Attempts     [][]map[string]interface{}
}

type crosswordResetCandidate struct {
	UserID         int
	CrosswordIndex int
	StartDate      string
	Progress       map[string]interface{}
}

// ProcessDailyReset обрабатывает ежедневную смену слов/кроссвордов в 00:00
func ProcessDailyReset(ctx context.Context) error {
	if err := google_sheets.EnsureRequiredSheets(ctx); err != nil {
		return fmt.Errorf("ошибка проверки обязательных листов: %w", err)
	}

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	log.Printf("Начинаем ежедневный сброс: today=%s yesterday=%s", today, yesterday)

	var errs []string

	if err := processWordleReset(ctx, today, yesterday); err != nil {
		log.Printf("Ошибка обработки Wordle reset: %v", err)
		errs = append(errs, "wordle: "+err.Error())
	}

	if err := processCrosswordReset(ctx, today, yesterday); err != nil {
		log.Printf("Ошибка обработки Crossword reset: %v", err)
		errs = append(errs, "crossword: "+err.Error())
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	log.Println("Ежедневный сброс завершен успешно")
	return nil
}

func processWordleReset(ctx context.Context, today, yesterday string) error {
	lastSwitchDate, err := google_sheets.GetConfigValue(ctx, "WORDLE_LAST_SWITCH")
	if err != nil {
		return fmt.Errorf("не удалось прочитать WORDLE_LAST_SWITCH: %w", err)
	}
	if lastSwitchDate == today {
		log.Printf("Wordle reset уже выполнен %s, повтор пропускаем", today)
		return nil
	}

	awardedUsers, err := loadAwardedUsers(ctx, wordleAwardDateKey, wordleAwardUsersKey, today)
	if err != nil {
		return fmt.Errorf("не удалось прочитать прогресс partial reward Wordle: %w", err)
	}

	candidates, err := listWordleResetCandidates(ctx)
	if err != nil {
		return err
	}

	awardedCount := 0
	for _, candidate := range candidates {
		if candidate.UserID <= 0 || candidate.LastWordDate != yesterday {
			continue
		}
		if _, ok := awardedUsers[candidate.UserID]; ok {
			continue
		}
		if isWinningWordleAttempt(candidate.Attempts) {
			continue
		}

		if err := google_sheets.AddDirectGamePoints(ctx, candidate.UserID, "wordle", wordlePartialPoints); err != nil {
			return fmt.Errorf("не удалось начислить partial reward Wordle user_id=%d: %w", candidate.UserID, err)
		}

		awardedUsers[candidate.UserID] = struct{}{}
		if err := persistAwardedUsers(ctx, wordleAwardDateKey, wordleAwardUsersKey, today, awardedUsers); err != nil {
			return fmt.Errorf("не удалось сохранить прогресс partial reward Wordle: %w", err)
		}
		awardedCount++
	}

	if awardedCount == 0 {
		if err := persistAwardedUsers(ctx, wordleAwardDateKey, wordleAwardUsersKey, today, awardedUsers); err != nil {
			return fmt.Errorf("не удалось зафиксировать пустой прогресс partial reward Wordle: %w", err)
		}
	}

	if err := google_sheets.SwitchWordleWordForAll(ctx); err != nil {
		return fmt.Errorf("не удалось переключить Wordle: %w", err)
	}

	log.Printf("Wordle reset завершен: awarded=%d", awardedCount)
	return nil
}

func processCrosswordReset(ctx context.Context, today, yesterday string) error {
	lastSwitchDate, err := google_sheets.GetConfigValue(ctx, "CROSSWORD_LAST_SWITCH")
	if err != nil {
		return fmt.Errorf("не удалось прочитать CROSSWORD_LAST_SWITCH: %w", err)
	}
	if lastSwitchDate == today {
		log.Printf("Crossword reset уже выполнен %s, повтор пропускаем", today)
		return nil
	}

	awardedUsers, err := loadAwardedUsers(ctx, crosswordAwardDateKey, crosswordAwardUsersKey, today)
	if err != nil {
		return fmt.Errorf("не удалось прочитать прогресс partial reward Crossword: %w", err)
	}

	candidates, err := listCrosswordResetCandidates(ctx)
	if err != nil {
		return err
	}

	wordCounts := make(map[int]int)
	awardedCount := 0

	for _, candidate := range candidates {
		if candidate.UserID <= 0 || candidate.StartDate != yesterday {
			continue
		}
		if _, ok := awardedUsers[candidate.UserID]; ok {
			continue
		}

		wordCount, ok := wordCounts[candidate.CrosswordIndex]
		if !ok {
			words, err := google_sheets.GetCrosswordWords(ctx, candidate.CrosswordIndex)
			if err != nil {
				return fmt.Errorf("не удалось получить слова кроссворда index=%d: %w", candidate.CrosswordIndex, err)
			}
			wordCount = len(words)
			wordCounts[candidate.CrosswordIndex] = wordCount
		}
		if wordCount == 0 {
			continue
		}
		if isSolvedCrossword(candidate.Progress, candidate.CrosswordIndex, wordCount) {
			continue
		}

		if err := google_sheets.AddDirectGamePoints(ctx, candidate.UserID, "crossword", crosswordPartialPoints); err != nil {
			return fmt.Errorf("не удалось начислить partial reward Crossword user_id=%d: %w", candidate.UserID, err)
		}

		awardedUsers[candidate.UserID] = struct{}{}
		if err := persistAwardedUsers(ctx, crosswordAwardDateKey, crosswordAwardUsersKey, today, awardedUsers); err != nil {
			return fmt.Errorf("не удалось сохранить прогресс partial reward Crossword: %w", err)
		}
		awardedCount++
	}

	if awardedCount == 0 {
		if err := persistAwardedUsers(ctx, crosswordAwardDateKey, crosswordAwardUsersKey, today, awardedUsers); err != nil {
			return fmt.Errorf("не удалось зафиксировать пустой прогресс partial reward Crossword: %w", err)
		}
	}

	nextIndex, err := google_sheets.SwitchCrosswordForAll(ctx)
	if err != nil {
		return fmt.Errorf("не удалось переключить Crossword: %w", err)
	}

	log.Printf("Crossword reset завершен: awarded=%d next_index=%d", awardedCount, nextIndex)
	return nil
}

func listWordleResetCandidates(ctx context.Context) ([]wordleResetCandidate, error) {
	service, err := google_sheets.GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	resp, err := service.Spreadsheets.Values.Get(config.GoogleSheetsID, "Wordle_Состояние!A:E").Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения Wordle_Состояние: %w", err)
	}

	candidates := make([]wordleResetCandidate, 0, maxInt(len(resp.Values)-1, 0))
	for i, row := range resp.Values {
		if i == 0 || len(row) == 0 {
			continue
		}

		userID, err := parsePositiveIntCell(row[0])
		if err != nil {
			log.Printf("Wordle reset: пропускаем строку %d, невалидный user_id: %v", i+1, err)
			continue
		}

		lastWordDate := ""
		if len(row) > 4 {
			lastWordDate = sheetString(row[4])
		}

		var attempts [][]map[string]interface{}
		if len(row) > 2 {
			rawAttempts := sheetString(row[2])
			if rawAttempts != "" {
				if err := json.Unmarshal([]byte(rawAttempts), &attempts); err != nil {
					log.Printf("Wordle reset: пропускаем user_id=%d, поврежден attempts JSON: %v", userID, err)
					continue
				}
			}
		}

		candidates = append(candidates, wordleResetCandidate{
			UserID:       userID,
			LastWordDate: lastWordDate,
			Attempts:     attempts,
		})
	}

	return candidates, nil
}

func listCrosswordResetCandidates(ctx context.Context) ([]crosswordResetCandidate, error) {
	service, err := google_sheets.GetGoogleSheetsClient()
	if err != nil {
		return nil, err
	}

	resp, err := service.Spreadsheets.Values.Get(config.GoogleSheetsID, "Кроссворд_Прогресс!A:D").Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения Кроссворд_Прогресс: %w", err)
	}

	candidates := make([]crosswordResetCandidate, 0, maxInt(len(resp.Values)-1, 0))
	for i, row := range resp.Values {
		if i == 0 || len(row) == 0 {
			continue
		}

		userID, err := parsePositiveIntCell(row[0])
		if err != nil {
			log.Printf("Crossword reset: пропускаем строку %d, невалидный user_id: %v", i+1, err)
			continue
		}

		crosswordIndex := 0
		if len(row) > 1 {
			if parsedIndex, err := parseOptionalIntCell(row[1]); err == nil && parsedIndex >= 0 {
				crosswordIndex = parsedIndex
			}
		}

		progress := map[string]interface{}{}
		if len(row) > 2 {
			rawProgress := sheetString(row[2])
			if rawProgress != "" {
				if err := json.Unmarshal([]byte(rawProgress), &progress); err != nil {
					log.Printf("Crossword reset: пропускаем user_id=%d, поврежден progress JSON: %v", userID, err)
					continue
				}
			}
		}

		startDate := ""
		if len(row) > 3 {
			startDate = sheetString(row[3])
		}

		candidates = append(candidates, crosswordResetCandidate{
			UserID:         userID,
			CrosswordIndex: crosswordIndex,
			StartDate:      startDate,
			Progress:       progress,
		})
	}

	return candidates, nil
}

func loadAwardedUsers(ctx context.Context, dateKey, usersKey, today string) (map[int]struct{}, error) {
	storedDate, err := google_sheets.GetConfigValue(ctx, dateKey)
	if err != nil {
		return nil, err
	}
	if storedDate != today {
		return map[int]struct{}{}, nil
	}

	rawUsers, err := google_sheets.GetConfigValue(ctx, usersKey)
	if err != nil {
		return nil, err
	}

	return parseAwardedUsers(rawUsers), nil
}

func persistAwardedUsers(ctx context.Context, dateKey, usersKey, today string, users map[int]struct{}) error {
	return google_sheets.UpsertConfigEntries(ctx, map[string]string{
		dateKey:  today,
		usersKey: joinAwardedUsers(users),
	})
}

func parseAwardedUsers(raw string) map[int]struct{} {
	result := make(map[int]struct{})
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil || id <= 0 {
			continue
		}
		result[id] = struct{}{}
	}
	return result
}

func joinAwardedUsers(users map[int]struct{}) string {
	if len(users) == 0 {
		return ""
	}

	ids := make([]int, 0, len(users))
	for id := range users {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, strconv.Itoa(id))
	}
	return strings.Join(values, ",")
}

func isWinningWordleAttempt(attempts [][]map[string]interface{}) bool {
	for _, attempt := range attempts {
		if len(attempt) != 5 {
			continue
		}

		allCorrect := true
		for _, cell := range attempt {
			state, ok := cell["state"].(string)
			if !ok || state != "correct" {
				allCorrect = false
				break
			}
		}

		if allCorrect {
			return true
		}
	}

	return false
}

func isSolvedCrossword(progress map[string]interface{}, crosswordIndex int, totalWords int) bool {
	if totalWords <= 0 {
		return false
	}

	rawWords, ok := progress[strconv.Itoa(crosswordIndex)]
	if !ok {
		return false
	}

	words, ok := rawWords.([]interface{})
	if !ok {
		return false
	}

	uniqueWords := make(map[string]struct{})
	for _, rawWord := range words {
		word, ok := rawWord.(string)
		if !ok {
			continue
		}
		word = strings.TrimSpace(strings.ToUpper(word))
		if word == "" {
			continue
		}
		uniqueWords[word] = struct{}{}
	}

	return len(uniqueWords) >= totalWords
}

func parsePositiveIntCell(value interface{}) (int, error) {
	parsed, err := parseOptionalIntCell(value)
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("expected positive integer, got %d", parsed)
	}
	return parsed, nil
}

func parseOptionalIntCell(value interface{}) (int, error) {
	raw := sheetString(value)
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}

func sheetString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(v, 'f', -1, 64))
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ScheduleDailyReset планирует ежедневный сброс в 00:00
func ScheduleDailyReset(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🚨 Паника в ScheduleDailyReset: %v", r)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				log.Println("⏹️ Планировщик ежедневного сброса остановлен")
				return
			default:
				now := time.Now()
				nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
				if now.Hour() == 0 && now.Minute() == 0 {
					nextMidnight = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				}

				waitDuration := nextMidnight.Sub(now)
				log.Printf("⏰ Следующий ежедневный сброс запланирован на %s (через %v)", nextMidnight, waitDuration)

				timer := time.NewTimer(waitDuration)
				select {
				case <-timer.C:
					log.Println("🔄 Запуск ежедневного сброса...")
					if err := ProcessDailyReset(ctx); err != nil {
						log.Printf("⚠️ Ошибка ежедневного сброса: %v", err)
					}
				case <-ctx.Done():
					timer.Stop()
					log.Println("⏹️ Планировщик ежедневного сброса остановлен")
					return
				}
			}
		}
	}()
}
