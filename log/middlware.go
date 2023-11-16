package log

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// WithZap is a logger middleware for Chi that implements the [http.Handler] interface.
func WithZap(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		t1 := time.Now()
		defer func() {
			reqLogger := logger.With(
				zap.String("proto", r.Proto),
				zap.String("path", r.URL.Path),
				zap.Duration("took", time.Since(t1)),
				zap.Int("status", ww.Status()),
				zap.Int("size", ww.BytesWritten()),
			)
			reqLogger.Info("request")
		}()
		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}
