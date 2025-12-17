package google_sheets

import (
	"bufio"
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"wedding-bot/internal/config"
)

var (
	allowedWordsMu sync.RWMutex
	allowedWords   map[string]struct{}
	allowedLoaded  time.Time
)

// IsWordAllowed проверяет, допустимо ли слово для Wordle
func IsWordAllowed(ctx context.Context, word string) bool {
	word = strings.TrimSpace(strings.ToUpper(word))
	if word == "" {
		return false
	}

	ensureAllowedWords(ctx)

	allowedWordsMu.RLock()
	_, ok := allowedWords[word]
	allowedWordsMu.RUnlock()
	return ok
}

func ensureAllowedWords(ctx context.Context) {
	allowedWordsMu.RLock()
	if allowedWords != nil && time.Since(allowedLoaded) < 10*time.Minute {
		allowedWordsMu.RUnlock()
		return
	}
	allowedWordsMu.RUnlock()

	allowedWordsMu.Lock()
	defer allowedWordsMu.Unlock()

	if allowedWords == nil {
		allowedWords = make(map[string]struct{})
	}
	// Очистим и перезагрузим
	for k := range allowedWords {
		delete(allowedWords, k)
	}

	loadLocalDictionary(config.WordleDictionaryPath)

	// Добавляем слова из листа Wordle как дополнительный словарь
	if words, err := getAllWordleWords(ctx); err == nil {
		for _, w := range words {
			if w == "" {
				continue
			}
			allowedWords[strings.ToUpper(strings.TrimSpace(w))] = struct{}{}
		}
	}

	allowedLoaded = time.Now()
}

func loadLocalDictionary(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Не удалось открыть словарь Wordle (%s): %v", path, err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.ToUpper(strings.TrimSpace(scanner.Text()))
		if word == "" || strings.HasPrefix(word, "#") {
			continue
		}
		allowedWords[word] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Ошибка чтения словаря Wordle: %v", err)
	}
}
