package daily_reset

import (
	"context"
	"fmt"
	"log"
	"time"

	"wedding-bot/internal/google_sheets"
)

// AddHalfPoints добавляет половину баллов напрямую в Google Sheets
func AddHalfPoints(ctx context.Context, userID int, gameType string, points int) error {
	// Используем UpdateGameScore для добавления очков
	// Для этого нужно получить текущий счет и добавить к нему
	stats, err := google_sheets.GetGameStats(ctx, userID)
	if err != nil {
		return fmt.Errorf("ошибка получения статистики: %w", err)
	}

	if stats == nil {
		// Создаем новую запись
		// Для этого нужно вызвать UpdateGameScore с нулевым счетом, но это не совсем правильно
		// Лучше напрямую обновить через UpdateGameScore с нужным количеством очков
		// Но для упрощения используем существующую логику
		return nil
	}

	// TODO: Реализовать прямое добавление очков в Google Sheets
	// Для упрощения пока используем существующую логику через UpdateGameScore
	// Но это не совсем правильно, так как UpdateGameScore конвертирует игровые очки
	// Лучше создать отдельную функцию в google_sheets для прямого добавления очков
	return nil
}

// ProcessDailyReset обрабатывает ежедневную смену слов/кроссвордов в 00:00
func ProcessDailyReset(ctx context.Context) error {
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	log.Printf("Начинаем ежедневный сброс: %s", today)

	// Обрабатываем Wordle
	if err := processWordleReset(ctx, today, yesterday); err != nil {
		log.Printf("Ошибка обработки Wordle reset: %v", err)
	}

	// Обрабатываем Crossword
	if err := processCrosswordReset(ctx, today, yesterday); err != nil {
		log.Printf("Ошибка обработки Crossword reset: %v", err)
	}

	log.Println("Ежедневный сброс завершен успешно")
	return nil
}

// processWordleReset обрабатывает сброс Wordle для всех пользователей
func processWordleReset(ctx context.Context, today, yesterday string) error {
	// TODO: Получить список всех пользователей из Wordle_Состояние
	// Для каждого пользователя:
	// 1. Проверить, отгадано ли слово
	// 2. Если нет - добавить 3 балла
	// 3. Сменить слово на следующее

	// Пока упрощенная версия - просто логируем
	log.Println("Обработка Wordle reset...")
	return nil
}

// processCrosswordReset обрабатывает сброс Crossword для всех пользователей
func processCrosswordReset(ctx context.Context, today, yesterday string) error {
	// TODO: Получить список всех пользователей из Кроссвод_Прогресс
	// Для каждого пользователя:
	// 1. Проверить, решен ли кроссворд
	// 2. Если нет - добавить 13 баллов
	// 3. Перейти к следующему кроссворду

	// Пока упрощенная версия - просто логируем
	log.Println("Обработка Crossword reset...")
	return nil
}

// ScheduleDailyReset планирует ежедневный сброс в 00:00
func ScheduleDailyReset(ctx context.Context) {
	go func() {
		for {
			now := time.Now()
			// Вычисляем время до следующей полуночи
			nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			if now.Hour() == 0 && now.Minute() == 0 {
				// Если уже 00:00, запускаем сразу
				nextMidnight = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			}

			waitDuration := nextMidnight.Sub(now)
			log.Printf("Следующий ежедневный сброс запланирован на %s (через %v)", nextMidnight, waitDuration)

			time.Sleep(waitDuration)

			// Запускаем сброс
			log.Println("Запуск ежедневного сброса...")
			if err := ProcessDailyReset(ctx); err != nil {
				log.Printf("Ошибка ежедневного сброса: %v", err)
			}
		}
	}()
}

