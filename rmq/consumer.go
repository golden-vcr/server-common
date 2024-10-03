package rmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

// HandlerFunc describes a function that your application defines in order to handle
// events of a specific type consumed from a queue: a nil return value indicates that
// the message was handled successfully (or ignored) and should be acknowledged; any
// non-nil error will cause the consumer to halt
type HandlerFunc[T any] func(ctx context.Context, logger *slog.Logger, ev *T) error

// Consumer encapsulates the state necessary to run a long-lived consumer process that
// receives message from a queue
type Consumer struct {
	ctx        context.Context
	logger     *slog.Logger
	receiver   Receiver
	deliveries <-chan amqp.Delivery
}

// NewConsumer ensures that the necessary queues/exchanges/etc. are created and bound
// for this queue, then prepares a Consumer that can be used to receive messages from
// the queue by calling rmq.RunConsumer in a goroutine. You MUST call Close() on the
// consumer when finished with it.
func (d *QueueDeclaration) NewConsumer(ctx context.Context, logger *slog.Logger, conn *amqp.Connection) (*Consumer, error) {
	// Prepare a root logger for this consumer which will identify the queue name
	logger = logger.With("queueName", d.Name, "queueType", d.Type)

	// Create a amqp.Channel which we'll used to declare the required AMQP primitives, but
	// which will also live as long as our Consumer does
	ch, err := conn.Channel()
	if err != nil {
		logger.Error("Failed to open AQMP channel", "error", err)
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Prepare an rmq.Receiver which wraps our AMQP channel and initializes the
	// appropriate AMQP queues/exchanges etc. based on our queue type
	receiver, err := d.initReceiver(ch)
	if err != nil {
		ch.Close()
		logger.Error("Failed to initialize receiver", "error", err)
		return nil, fmt.Errorf("failed to initialize receiver: %w", err)
	}

	// Start receiving: this calls Consume on our amqp.Channel, sending an amqp.Delivery
	// messages to the resulting Go channel each time a message is sent to the queue for
	// us to receive
	deliveries, err := receiver.Recv(ctx)
	if err != nil {
		receiver.Close()
		logger.Error("Recv failed", "error", err)
		return nil, fmt.Errorf("failed to initialize recv channel: %w", err)
	}

	// Our state is initialized, the caller can begin receiving by passing their new
	// Consumer, along with a HandlerFunc callback of the appropriate event type, to the
	// RunConsumer function
	logger.Info("Consumer ready to receive")
	return &Consumer{
		ctx:        ctx,
		logger:     logger,
		receiver:   receiver,
		deliveries: deliveries,
	}, nil
}

// Close ensures that the underlying AMQP channel is closed once the consumer is no
// longer needed
func (c *Consumer) Close() {
	if c.receiver != nil {
		c.receiver.Close()
	}
}

// RunConsumer block indefinitely for as long as its receiver channel is open,
// processing each delivery by parsing its payload to the appropriate Event type T, then
// allowing the provided handler function to respond to each message, serially. If any
// error occurs in message-handling, immediately halts and returns an error, without
// acknowleding the current message. Returns nil if the deliveries channel closes and no
// more messages remain.
func RunConsumer[T any](c *Consumer, f HandlerFunc[T]) error {
	// Handle deliveries one-at-a-time as long as they're arriving
	for d := range c.deliveries {
		// Deserialize the JSON payload to an event struct of the appropriate type
		var ev T
		if err := json.Unmarshal(d.Body, &ev); err != nil {
			c.logger.Error("Failed to unmarshal message body to event", "messageBody", d.Body, "error", err)
			return err
		}

		// Call our user-provided handler function to respond to the event
		logger := c.logger.With("queueEvent", ev)
		if err := f(c.ctx, logger, &ev); err != nil {
			logger.Error("Failed to handle event", "error", err)
			return err
		}

		// Our handler function completed without error, so we can acknowledge the event and
		// we're done
		if err := d.Ack(false); err != nil {
			logger.Error("Failed to acknowledge event", "error", err)
			return err
		}
		logger.Info("Event handled successfully and acknowledged")
	}
	c.logger.Info("RunConsumer finished; deliveries channel is closed")
	return nil
}
