package entry

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/exp/slog"
	"golang.org/x/sync/errgroup"
)

// RunServer blocks while an HTTP server application runs
func RunServer(a Application, handler http.Handler, bindAddr string, listenPort int) {
	// Prepare an http.Server with reasonable default config, using our provided handler
	addr := fmt.Sprintf("%s:%d", bindAddr, listenPort)
	server := &http.Server{
		Addr:     addr,
		Handler:  Middleware(a.Log())(handler),
		ErrorLog: NewErrorLog(*a.Log()),
	}

	// Kick off a goroutine which calls server.ListenAndServe()
	a.Log().Info("Now listening", "bindAddr", bindAddr, "listenPort", listenPort)
	var wg errgroup.Group
	wg.Go(server.ListenAndServe)

	// If our application-level context is closed, abort
	select {
	case <-a.Context().Done():
		a.Log().Info("Received signal; closing server")
		server.Shutdown(context.Background())
	}

	// Otherwise, block until ListenAndServe returns
	err := wg.Wait()
	if err == http.ErrServerClosed {
		a.Log().Info("Server closed")
	} else {
		a.Fail("error running server", err)
	}
}

// NewErrorLog adapts an slog.Logger to the simpler log.Logger interface used by
// http.Server's ErrorLog field
func NewErrorLog(s slog.Logger) *log.Logger {
	w := errorLogWriter{s}
	return log.New(w, "", 0)
}

// errorLog is an implementation of io.Writer that handles http server errors by writing
// them to an underlying slog.Logger
type errorLogWriter struct {
	slog.Logger
}

func (w errorLogWriter) Write(data []byte) (int, error) {
	w.Logger.Error("http.Server error", "error", string(data))
	return len(data), nil
}
