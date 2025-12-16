package cache

import (
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

var (
	memoryCache *cache.Cache
	memoryMu    sync.RWMutex
)

// InitMemoryCache инициализирует in-memory кэш
func InitMemoryCache() {
	memoryMu.Lock()
	defer memoryMu.Unlock()

	// Кэш с TTL 5 минут и очисткой каждые 10 минут
	memoryCache = cache.New(5*time.Minute, 10*time.Minute)
}

// GetMemoryCache возвращает экземпляр in-memory кэша
func GetMemoryCache() *cache.Cache {
	memoryMu.RLock()
	defer memoryMu.RUnlock()
	return memoryCache
}

// SetMemoryCache устанавливает значение в кэш
func SetMemoryCache(key string, value interface{}, expiration time.Duration) {
	if memoryCache == nil {
		InitMemoryCache()
	}
	memoryCache.Set(key, value, expiration)
}

// GetMemoryCacheValue получает значение из кэша
func GetMemoryCacheValue(key string) (interface{}, bool) {
	if memoryCache == nil {
		return nil, false
	}
	return memoryCache.Get(key)
}

// DeleteMemoryCache удаляет значение из кэша
func DeleteMemoryCache(key string) {
	if memoryCache != nil {
		memoryCache.Delete(key)
	}
}

// ClearMemoryCache очищает весь кэш
func ClearMemoryCache() {
	if memoryCache != nil {
		memoryCache.Flush()
	}
}

