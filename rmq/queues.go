package rmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// QueueType is an abtraction that identifies one of a handful of use cases for RabbitMQ
// within our platform
type QueueType string

const (
	// QueueTypeFanout identifies a queue used to record events that multiple services'
	// consumer processes may be interested in: using this queue type results in a fanout
	// exchange being created, with each consumer binding its own temporary queue to that
	// exchange
	QueueTypeFanout QueueType = "fanout"

	// QueueTypeWork identifies a queue used to record requests that should be fulfilled
	// by only a single worker process
	QueueTypeWork QueueType = "work"
)

// QueueDeclaration records the canonical details of how a particular queue is to be
// configured
type QueueDeclaration struct {
	Name string
	Type QueueType
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

// NewConsumer ensures that the necessary queues/exchanges/etc. are created and bound
// for this queue, then prepares a Consumer interface that can be used to receive
// messages from the queue
func (d *QueueDeclaration) NewConsumer(conn *amqp.Connection) (Consumer, error) {
	// Create a channel which we'll used to declare the required AMQP primitives, but
	// which will also live as long as our Consumer does
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel for %s queue '%s': %w", d.Type, d.Name, err)
	}

	if d.Type == QueueTypeFanout {
		return d.newFanoutConsumer(ch)
	}
	if d.Type == QueueTypeWork {
		return d.newWorkConsumer(ch)
	}
	return nil, fmt.Errorf("queue '%s' has unrecognized type %s", d.Name, d.Type)
}
