package rmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Producer can send arbitrary, JSON-formatted messages to a single message queue
type Producer interface {
	Send(ctx context.Context, jsonData []byte) error
}

// NewProducer initializes a Producer from an AMQP client connection, configuring it to
// send messages to an exchange wtih the given name
func NewProducer(conn *amqp.Connection, exchange string) (Producer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}
	defer ch.Close()

	if err := declareFanoutExchange(ch, exchange); err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &producer{
		conn:     conn,
		exchange: exchange,
	}, nil
}

// producer is a concrete implementation that uses AMQP under the hood, sending messages
// to a named exchange
type producer struct {
	conn     *amqp.Connection
	exchange string
}

func (p *producer) Send(ctx context.Context, jsonData []byte) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}
	mandatory := false
	immediate := false
	return ch.PublishWithContext(ctx, p.exchange, "", mandatory, immediate, amqp.Publishing{
		ContentType: "application/json",
		Body:        jsonData,
	})
}
