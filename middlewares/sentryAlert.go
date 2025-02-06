package middlewares

import (
	"net/http"

	"github.com/getsentry/sentry-go"
)

// ExampleMiddleware demonstrates how to extract the Sentry hub from the context.
func SentryAlertMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		if hub != nil {
			// hub.Scope().SetTag("example_middleware", "active")
			hub.CaptureMessage("Alert from SentryAlertMiddleware")
		}
		next.ServeHTTP(w, r)
	})
}
