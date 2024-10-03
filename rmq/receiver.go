package rmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Receiver wraps an AMQP channel which we've initialized for receiving; it serves as a
// low-level, queue-type-agnostic interface for a consumer to receive messages and clean
// up state when done
type Receiver interface {
	Close()
	Recv(ctx context.Context) (<-chan amqp.Delivery, error)
}

// initReceiver initializes a Receiver on the given channel based on the canonical usage
// pattern for this queue
func (d *QueueDeclaration) initReceiver(ch *amqp.Channel) (Receiver, error) {
	if d.Type == QueueTypeFanout {
		return d.newFanoutReceiver(ch)
	}
	if d.Type == QueueTypeWork {
		return d.newWorkReceiver(ch)
	}
	return nil, fmt.Errorf("queue '%s' has unrecognized type %s", d.Name, d.Type)
}
