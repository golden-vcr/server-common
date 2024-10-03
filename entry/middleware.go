package entry

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Middleware injects HTTP response handler logic to facilitate tracing and logging:
// every incoming request will receive an X-Request-Id header (accessible via a context
// value) and a customized slog.Logger instance (also stored in the request context, and
// accessible via entry.Log()), and all requests will be logged
func Middleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate a unique ID for this request, if it doesn't already have one
			requestId := r.Header.Get("x-request-id")
			if requestId == "" {
				requestId = uuid.NewString()
			}

			// Prepare a logger with the relevant details of this request
			reqLogger := logger.With(
				"requestId", requestId,
				"method", r.Method,
				"path", r.URL.Path,
				"remoteAddr", r.RemoteAddr,
			)
			reqLogger.Debug("Handling request")

			// Inject the request ID and logger into the request context, so that HTTP
			// handler functions can pull them out and use them
			ctx := context.WithValue(r.Context(), "x-request-id", requestId)
			ctx = context.WithValue(ctx, "logger", reqLogger)
			r = r.WithContext(ctx)

			// Preemptively set the X-Request-Id response header, so that the request ID
			// will be carried end-to-end
			w.Header().Set("x-request-id", requestId)

			// Wrap our ResponseWriter in a struct that will capture the response code
			// written by the HTTP handler
			recorder := statusRecorder{ResponseWriter: w}

			// Handle the request, measuring how long it takes to execute
			start := time.Now()
			next.ServeHTTP(&recorder, r)
			elapsed := time.Since(start)
			elapsedMilliseconds := float64(elapsed.Nanoseconds()) / float64(1000000)

			// Write a final log message indicating that the request is finished
			level := slog.LevelError
			if recorder.status >= 100 && recorder.status <= 499 {
				level = slog.LevelInfo
			}
			reqLogger.Log(nil, level,
				"Request finished",
				"elapsedMilliseconds", elapsedMilliseconds,
				"status", recorder.status,
			)
		})
	}
}

// Log returns a slog.Logger, guaranteed to be valid, for use within the context of the
// provided request
func Log(r *http.Request) *slog.Logger {
	logger, ok := r.Context().Value("logger").(*slog.Logger)
	if ok && logger != nil {
		return logger
	}
	return slog.Default()
}

// ConveyRequestId checks to see if it's being called in the context of an HTTP request
// with a valid X-Request-Id, and if so, it modified an outgoing HTTP request to carry
// the same request ID as a header
func ConveyRequestId(ctx context.Context, req *http.Request) *http.Request {
	if req.Header.Get("x-request-id") == "" {
		requestId, ok := ctx.Value("x-request-id").(string)
		if ok && requestId != "" {
			req.Header.Set("x-request-id", requestId)
		}
	}
	return req
}

// statusRecorder wraps an http.ResponseWriter in order to intercept and store the HTTP
// status code for the response to a request
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	n, err := r.ResponseWriter.Write(data)
	if err == nil && r.status == 0 {
		r.status = http.StatusOK
	}
	return n, err
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Flush() {
	r.ResponseWriter.(http.Flusher).Flush()
}
