package entry

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// RunServer blocks while a gRPC server application runs
func RunGRPCServer(ctx context.Context, logger *slog.Logger, s *grpc.Server, bindAddr string, listenPort uint16) {
	// Bind to the configured port and begin listening for TCP connections
	addr := fmt.Sprintf("%s:%d", bindAddr, listenPort)
	listenConfig := net.ListenConfig{}
	lis, err := listenConfig.Listen(ctx, "tcp", addr)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to listen on %s", addr), "error", err)
		os.Exit(1)
	}

	// Kick off a goroutine which calls s.Serve
	var wg errgroup.Group
	wg.Go(func() error { return s.Serve(lis) })

	// Block indefinitely, running the server all the while, until our application-level
	// context is done
	select {
	case <-ctx.Done():
		cancelErr := context.Cause(ctx)
		if cancelErr != nil && cancelErr != ctx.Err() {
			logger.Error("Closing server due to application error", "error", cancelErr)
		} else {
			logger.Info("Application is shutting down cleanly; closing server")
		}
		s.GracefulStop()
	}

	// Block until s.Serve returns so we can ensure that the server is closed
	err = wg.Wait()
	if err != nil {
		logger.Error("Error running server", "error", err)
		os.Exit(1)
	}
	logger.Info("Server closed")
}
