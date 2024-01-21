package rmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer can receive AMQP messages from a single message queue
type Consumer interface {
	Close()
	Recv(ctx context.Context) (<-chan amqp.Delivery, error)
}

// NewConsumer initializes a Consumer from an AMQP client connection, configuring it to
// receive messages from an exchange with the given name
func NewConsumer(conn *amqp.Connection, exchange string) (Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	if err := declareFanoutExchange(ch, exchange); err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	q, err := declareConsumerQueue(ch, exchange)
	if err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to declare consumer queue: %w", err)
	}

	return &consumer{
		conn:     conn,
		ch:       ch,
		q:        q,
		exchange: exchange,
	}, nil
}

// producer is a concrete implementation that uses AMQP under the hood, receiving
// messages from a unique queue that is declared (and bound to a named exchange) for the
// lifetime of the consumer procexx
type consumer struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	q        *amqp.Queue
	exchange string
}

func (c *consumer) Close() {
	c.ch.Close()
	c.conn.Close()
}

func (c *consumer) Recv(ctx context.Context) (<-chan amqp.Delivery, error) {
	autoAck := true
	exclusive := false
	noLocal := false
	noWait := false
	return c.ch.ConsumeWithContext(ctx, c.q.Name, "", autoAck, exclusive, noLocal, noWait, nil)
}
