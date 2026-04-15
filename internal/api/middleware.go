package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/unrolled/secure"

	"wedding-bot/internal/config"
)

type contextKey string

const requestIDContextKey contextKey = "request_id"

// initLogger инициализирует структурированное логирование
func initLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if config.IsDebug() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

// securityMiddleware добавляет security headers
func securityMiddleware(next http.Handler) http.Handler {
	isDev := config.IsDebug()

	connectSrc := []string{"'self'"}
	if isDev {
		connectSrc = append(connectSrc, "http://localhost:5173", "http://127.0.0.1:5173")
	}

	cspParts := []string{
		"default-src 'self'",
		"script-src 'self' https://telegram.org https://*.telegram.org https://megatimer.ru",
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"img-src 'self' data: blob: https://*.telegram.org",
		"font-src 'self' data: https://fonts.gstatic.com",
		"connect-src " + strings.Join(connectSrc, " "),
		"frame-src 'self' https://www.google.com https://maps.google.com",
		"frame-ancestors 'self'",
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
		requestID, _ := r.Context().Value(requestIDContextKey).(string)

		// Создаем logger с контекстом запроса
		logger := log.With().
			Str("request_id", requestID).
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
			Dur("duration", duration).
			Int64("duration_ms", duration.Milliseconds())

		if rw.statusCode >= 400 {
			event = logger.Error().
				Int("status", rw.statusCode).
				Dur("duration", duration).
				Int64("duration_ms", duration.Milliseconds())
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
		ctx := context.WithValue(r.Context(), requestIDContextKey, requestID)
		r = r.WithContext(ctx)

		// Добавляем в заголовок ответа
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r)
	})
}

// generateRequestID генерирует уникальный ID запроса
func generateRequestID() string {
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}

	return time.Now().UTC().Format("20060102150405") + "-" + hex.EncodeToString(suffix[:])
}
