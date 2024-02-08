package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golden-vcr/server-common/entry"
	"golang.org/x/exp/slog"
)

// Handler is an HTTP handler that serves a stream of data using Server-Sent Events
type Handler[T any] struct {
	ctx context.Context
	b   bus[T]

	ResolveEventId func(ev T) string
	OnConnect      func(lastEventId string) []T
}

// NewHandler initializes an SSE handler that will read messages from the given channel
// and fan them out to all extant HTTP connections
func NewHandler[T any](ctx context.Context, ch <-chan T) *Handler[T] {
	h := &Handler[T]{
		ctx: ctx,
		b: bus[T]{
			chs: make(map[chan T]struct{}),
		},
	}
	go func() {
		done := false
		for !done {
			select {
			case <-ctx.Done():
				done = true
				h.b.clear()
			case message := <-ch:
				h.b.publish(message)
			}
		}
	}()
	return h
}

// ServeHTTP responds by opening a long-lived HTTP connection to which events will be
// written as the handler receives them, formatted as text/event-stream messages with
// 'data' consisting of a JSON-encoded message payload
func (h *Handler[T]) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	logger := entry.Log(req)

	// If a content-type is explicitly requested, require that it's text/event-stream
	accept := req.Header.Get("accept")
	if accept != "" && accept != "*/*" && !strings.HasPrefix(accept, "text/event-stream") {
		message := fmt.Sprintf("content-type %s is not supported", accept)
		http.Error(res, message, http.StatusBadRequest)
		return
	}

	// Keep the connection alive and open a text/event-stream response body
	res.Header().Set("content-type", "text/event-stream")
	res.Header().Set("cache-control", "no-cache")
	res.Header().Set("connection", "keep-alive")
	res.WriteHeader(http.StatusOK)
	res.(http.Flusher).Flush()

	// If configured to send an initial value immediately upon connect, resolve that
	// value and send it: otherwise send an initial keepalive message to ensure that
	// Cloudflare will kick into action immediately without requiring special
	// configuration rules
	onConnectMessages := []T{}
	if h.OnConnect != nil {
		onConnectMessages = h.OnConnect(req.Header.Get("last-event-id"))
	}
	if len(onConnectMessages) > 0 {
		h.write(res, logger, onConnectMessages...)
	} else {
		res.Write([]byte(":\n\n"))
		res.(http.Flusher).Flush()
	}

	// Open a channel to receive message structs (i.e. any JSON-serializable value that
	// we want to send over our stream) as they're emitted
	ch := make(chan T, 32)
	h.b.register(ch)

	// Send all incoming messages to the client for as long as the connection is open
	logger.Info("Opened SSE connection", "remoteAddr", req.RemoteAddr)
	for {
		select {
		case <-time.After(30 * time.Second):
			res.Write([]byte(":\n\n"))
			res.(http.Flusher).Flush()
		case message := <-ch:
			h.write(res, logger, message)
		case <-h.ctx.Done():
			logger.Info("Server is shutting down; abandoning SSE connection", "remoteAddr", req.RemoteAddr)
			h.b.unregister(ch)
			return
		case <-req.Context().Done():
			logger.Info("Closed SSE connection", "remoteAddr", req.RemoteAddr)
			h.b.unregister(ch)
			return
		}
	}
}

func (h *Handler[T]) write(res http.ResponseWriter, logger *slog.Logger, messages ...T) {
	for _, message := range messages {
		eventId := ""
		if h.ResolveEventId != nil {
			eventId = h.ResolveEventId(message)
		}

		data, err := json.Marshal(message)
		if err != nil {
			logger.Error("Failed to serialize SSE message as JSON", "error", err)
			continue
		}

		if eventId != "" {
			fmt.Fprintf(res, "id: %s\n", eventId)
		}
		fmt.Fprintf(res, "data: %s\n\n", data)
	}
	res.(http.Flusher).Flush()
}
