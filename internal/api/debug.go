package api

import (
	"log"

	"wedding-bot/internal/config"
)

func debugLogf(format string, args ...interface{}) {
	if config.IsDebug() {
		log.Printf(format, args...)
	}
}
