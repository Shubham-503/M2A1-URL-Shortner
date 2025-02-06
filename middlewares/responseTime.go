package middlewares

import (
	"net/http"
	"time"
)

type timedResponseWriter struct {
	http.ResponseWriter
	start       time.Time
	wroteHeader bool
}

func (t *timedResponseWriter) WriteHeader(statusCode int) {
	if !t.wroteHeader {
		elapsed := time.Since(t.start)
		t.ResponseWriter.Header().Set("X-Response-Time", elapsed.String())
		t.wroteHeader = true
	}
	t.ResponseWriter.WriteHeader(statusCode)
}

func (t *timedResponseWriter) Write(b []byte) (int, error) {
	if !t.wroteHeader {
		t.WriteHeader(http.StatusOK)
	}
	return t.ResponseWriter.Write(b)
}

func ResponseTimeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		tw := &timedResponseWriter{
			ResponseWriter: w,
			start:          start,
		}
		next.ServeHTTP(tw, r)
	})
}
