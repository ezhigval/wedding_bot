# Рекомендуемые библиотеки для оптимизации проекта

## Бэкенд (Go)

### 1. Telegram Bot API
```go
github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.3.0
```
**Замена**: `gopkg.in/telebot.v3` → `go-telegram-bot-api/v5`

### 2. Rate Limiting
```go
github.com/didip/tollbooth/v7 v7.0.1
```
**Использование**: Защита от DDoS, ограничение запросов

### 3. Структурированное логирование
```go
github.com/rs/zerolog v1.32.0
```
**Альтернатива**: `go.uber.org/zap`
**Преимущества**: Быстрое, структурированное, JSON логи

### 4. Метрики (Prometheus)
```go
github.com/prometheus/client_golang v1.19.0
```
**Использование**: Метрики производительности, мониторинг

### 5. Кэширование
```go
// In-memory кэш (простой вариант)
github.com/patrickmn/go-cache v2.1.0+incompatible

// Redis (для продакшена)
github.com/redis/go-redis/v9 v9.4.0
```

### 6. Circuit Breaker
```go
github.com/sony/gobreaker v1.0.0
```
**Использование**: Защита от каскадных сбоев

### 7. Retry с exponential backoff
```go
github.com/cenkalti/backoff/v4 v4.2.1
```
**Использование**: Повторные попытки для Google Sheets API

### 8. Security headers
```go
github.com/unrolled/secure v1.17.0
```
**Использование**: Security headers (HSTS, CSP, etc.)

### 9. HTTP middleware
```go
github.com/gorilla/handlers v1.5.2
```
**Использование**: Compression, logging, CORS

### 10. Валидация
```go
github.com/go-playground/validator/v10 v10.18.0
```
**Использование**: Валидация входных данных

### 11. Конфигурация
```go
github.com/spf13/viper v1.18.2
```
**Опционально**: Расширенная работа с конфигурацией

---

## Фронтенд (React/TypeScript)

### 1. React Query (TanStack Query)
```json
"@tanstack/react-query": "^5.17.0"
```
**Использование**: Кэширование API запросов, синхронизация состояния

### 2. Виртуальный скролл
```json
"@tanstack/virtual": "^3.0.0"
```
**Использование**: Оптимизация рендеринга больших списков

### 3. Debounce
```json
"use-debounce": "^10.0.0"
```
**Использование**: Debounce для поиска и фильтров

### 4. Уведомления
```json
"react-hot-toast": "^2.4.1"
```
**Использование**: Красивые toast уведомления

### 5. PWA
```json
"vite-plugin-pwa": "^0.17.4"
```
**Использование**: Service Worker, offline support

### 6. Оптимизация изображений
```json
"vite-imagetools": "^6.2.7"
```
**Использование**: Автоматическая оптимизация изображений

### 7. State management (опционально)
```json
"zustand": "^4.4.7"
```
**Использование**: Легковесный state management

### 8. Error boundaries
```json
"react-error-boundary": "^4.0.11"
```
**Использование**: Улучшенная обработка ошибок

### 9. Intersection Observer
```json
"react-intersection-observer": "^9.5.3"
```
**Использование**: Lazy loading изображений

### 10. Bundle analyzer
```json
"rollup-plugin-visualizer": "^5.12.0"
```
**Использование**: Анализ размера bundle

---

## Инструменты разработки

### Go
- `golangci-lint` - линтер
- `goimports` - форматирование импортов
- `gofumpt` - строгое форматирование

### Frontend
- `eslint` - уже есть
- `prettier` - форматирование кода
- `husky` - git hooks
- `lint-staged` - проверка перед коммитом

---

## Приоритет внедрения

### Фаза 1 (Критично)
1. ✅ Миграция на go-telegram-bot-api/v5
2. ✅ Rate limiting
3. ✅ Структурированное логирование
4. ✅ React Query

### Фаза 2 (Важно)
5. Кэширование (in-memory или Redis)
6. Circuit breaker
7. Security headers
8. Оптимизация фронтенда

### Фаза 3 (Желательно)
9. Prometheus метрики
10. PWA
11. Тестирование
12. Документация

