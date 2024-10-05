package entry

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func GRPCServerLogging(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Check for an existing x-request-id header; and generate one if not found
		requestId := ""
		if values := metadata.ValueFromIncomingContext(ctx, "x-request-id"); len(values) > 0 {
			requestId = values[0]
		}
		if requestId == "" {
			requestId = uuid.NewString()
		}

		// Get the client IP
		remoteAddr := ""
		if p, ok := peer.FromContext(ctx); ok {
			remoteAddr = p.Addr.String()
		}

		// Prepare a logger with the relevant details of this request
		logger := logger.With(
			"requestId", requestId,
			"grpcMethod", info.FullMethod,
			"remoteAddr", remoteAddr,
		)
		logger.Debug("Handling request")

		// Handle the request, measuring how long it takes to execute
		start := time.Now()
		m, err := handler(context.WithValue(ctx, "logger", logger), req)
		elapsed := time.Since(start)
		elapsedMilliseconds := float64(elapsed.Nanoseconds()) / float64(1000000)

		// Write a final log message indicating that the request is finished, and noting any
		// error that resulted
		logger = logger.With("elapsedMilliseconds", elapsedMilliseconds)
		if err != nil {
			logger = logger.With("error", err)
			if grpcErr, ok := status.FromError(err); ok {
				logger = logger.With("grpcStatusCode", grpcErr.Code().String())
			}
			logger.Error("Request finished with error")
		} else {
			logger.Info("Request finished OK")
		}

		// Pass through the original result value and error unchanged
		return m, err
	}
}

func Logger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value("logger").(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
