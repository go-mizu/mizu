package pipeline

import "log"

func logInfof(format string, args ...any) {
	log.Printf("[pipeline] INFO  "+format, args...)
}

func logErrorf(format string, args ...any) {
	log.Printf("[pipeline] ERROR "+format, args...)
}
