package web

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

var dashboardLogger = log.New(os.Stderr, "[dashboard] ", log.LstdFlags|log.Lmicroseconds)

func logInfof(format string, args ...any) {
	dashboardLogger.Printf("INFO "+format, args...)
}

func logErrorf(format string, args ...any) {
	dashboardLogger.Printf("ERROR "+format, args...)
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response does not implement http.Hijacker")
	}
	return hj.Hijack()
}

func (w *loggingResponseWriter) Flush() {
	if fl, ok := w.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

func withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip noisy endpoints to keep dashboard logs useful.
		// Keep the original writer untouched for WS/Hijacker compatibility.
		if skipRequestLog(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		lw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lw, r)
		dur := time.Since(start).Round(time.Millisecond)
		logInfof("api method=%s path=%s status=%d dur=%s remote=%s ua=%q",
			r.Method, r.URL.RequestURI(), lw.status, dur, r.RemoteAddr, r.UserAgent(),
		)
	})
}

func skipRequestLog(path string) bool {
	switch path {
	case "/ws", "/api/overview", "/api/meta/status", "/api/meta/refresh":
		return true
	default:
		return false
	}
}
