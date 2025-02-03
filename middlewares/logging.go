package middlewares

import (
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger instances for different log levels
var (
	AuditLogger *log.Logger
	DebugLogger *log.Logger
	ErrorLogger *log.Logger
)

// ininLoggers initializes loggers for audit, debug and error logs.
func initLoggers() {
	// Ensure the log directiories exist
	createLogDir("logs/audit")
	createLogDir("logs/debug")
	createLogDir("logs/error")
}

// createLogDir ensures that provided directory exists.
func createLogDir(dir string) {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Fatalf("Could not create log directory %s: %v", dir, err)
	}

	// Setup Lumberjack for audit logs
	// TODO: Filename path should be from env variable
	auditLog := &lumberjack.Logger{
		Filename:   filepath.Join("logs", "audit", "audit.log"),
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	// Setup Lumberjack for debug logs
	// TODO: Filename path should be from env variable
	debugLog := &lumberjack.Logger{
		Filename:   filepath.Join("logs", "audit", "audit.log"),
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	// Setup Lumberjack for error logs
	// TODO: Filename path should be from env variable
	errorLog := &lumberjack.Logger{
		Filename:   filepath.Join("logs", "audit", "audit.log"),
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	// Create loggers with appropriate prefoxes and flags
	AuditLogger = log.New(auditLog, "AUDIT: ", log.LstdFlags)
	DebugLogger = log.New(debugLog, "DEBUG: ", log.LstdFlags)
	ErrorLogger = log.New(errorLog, "Error: ", log.LstdFlags)
}

func init() {
	initLoggers()
}

// LoggingMiddleware logs audit information for every request.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamp := time.Now().Format(time.RFC3339)
		method := r.Method
		url := r.URL.String()
		userAgent := r.UserAgent()
		ip := getIPAddress(r)

		// Log as an audit log entry
		AuditLogger.Printf("Time: %s | Method: %s | URL: %s | User-Agent: %s | IP: %s", timestamp, method, url, userAgent, ip)

		next.ServeHTTP(w, r)
	})

}

func getIPAddress(r *http.Request) string {
	// Check X-Forwarded-For header for proxies
	xff := r.Header.Get("X-Forwaded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Fallback to RemoteAddr (trim port)
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
