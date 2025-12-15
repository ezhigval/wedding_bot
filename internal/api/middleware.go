package api

import (
	"context"
	"net/http"
	"time"
)

// timeoutMiddleware добавляет таймаут к запросам
func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// getTimeoutForPath возвращает таймаут для конкретного пути
func getTimeoutForPath(path string) time.Duration {
	// Критические операции получают больше времени
	switch path {
	case "/api/register", "/api/upload-photo":
		return 30 * time.Second
	case "/api/update-game-score", "/api/crossword/progress":
		return 20 * time.Second
	default:
		return 10 * time.Second
	}
}

