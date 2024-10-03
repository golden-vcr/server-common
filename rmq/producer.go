package rmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Producer is the interface your application uses to send messages to the associated
// queue
type Producer interface {
	// Send accepts an instance of any JSON-serializable type and produces it to the queue
	// in JSON format
	Send(ctx context.Context, data interface{}) error
}

// NewProducer ensures that the necessary queues/exchanges/etc. are created and bound
// for this queue, then prepares a Producer interface that can be used to send messages
// to the queue
func (d *QueueDeclaration) NewProducer(conn *amqp.Connection) (Producer, error) {
	// Create a channel so we can declare the required AMQP primitives: channels are
	// short-lived, so this one gets closed once this function call completes; subsequent
	// sends will open their own channels
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel for %s queue '%s': %w", d.Type, d.Name, err)
	}
	defer ch.Close()

	if d.Type == QueueTypeFanout {
		return d.newFanoutProducer(conn, ch)
	}
	if d.Type == QueueTypeWork {
		return d.newWorkProducer(conn, ch)
	}
	return nil, fmt.Errorf("queue '%s' has unrecognized type %s", d.Name, d.Type)
}
