package api

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/unrolled/secure"
)

// initLogger инициализирует структурированное логирование
func initLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Проверяем переменную окружения для debug режима
	if os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

// securityMiddleware добавляет security headers
func securityMiddleware(next http.Handler) http.Handler {
	isDev := os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1"

	cspParts := []string{
		"default-src 'self'",
		"script-src 'self' https://telegram.org https://*.telegram.org",
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"img-src 'self' data: blob: https://*.telegram.org",
		"font-src 'self' data: https://fonts.gstatic.com",
		"connect-src 'self'",
		"frame-ancestors 'self'",
	}
	if isDev {
		cspParts = append(cspParts, "connect-src 'self' http://localhost:5173 http://127.0.0.1:5173")
	}

	secureMiddleware := secure.New(secure.Options{
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		BrowserXssFilter:      true,
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		PermissionsPolicy:     "geolocation=(), microphone=(), camera=()",
		ContentSecurityPolicy: strings.Join(cspParts, "; "),
		IsDevelopment:         isDev,
	})

	return secureMiddleware.Handler(next)
}

// rateLimitMiddleware добавляет rate limiting
func rateLimitMiddleware(next http.Handler) http.Handler {
	// Создаем лимитер: 100 запросов в минуту на IP
	limiter := tollbooth.NewLimiter(100, &limiter.ExpirableOptions{
		DefaultExpirationTTL: time.Minute,
	})

	// Исключаем health check из rate limiting
	return tollbooth.LimitFuncHandler(limiter, func(w http.ResponseWriter, r *http.Request) {
		// Пропускаем health check
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// structuredLoggingMiddleware логирует запросы структурированно
func structuredLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем logger с контекстом запроса
		logger := log.With().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Logger()

		// Создаем response writer для отслеживания статуса
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Выполняем запрос
		next.ServeHTTP(rw, r)

		// Логируем результат
		duration := time.Since(start)
		event := logger.Info().
			Int("status", rw.statusCode).
			Dur("duration_ms", duration).
			Int64("duration_ns", duration.Nanoseconds())

		if rw.statusCode >= 400 {
			event = logger.Error().
				Int("status", rw.statusCode).
				Dur("duration_ms", duration).
				Int64("duration_ns", duration.Nanoseconds())
		}

		event.Msg("HTTP request")
	})
}

// responseWriter обертка для ResponseWriter для отслеживания статуса
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// requestIDMiddleware добавляет уникальный ID к каждому запросу
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Добавляем request ID в контекст
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		r = r.WithContext(ctx)

		// Добавляем в заголовок ответа
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r)
	})
}

// generateRequestID генерирует уникальный ID запроса
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString генерирует случайную строку
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
