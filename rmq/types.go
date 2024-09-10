package rmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Producer interface {
	Send(ctx context.Context, data interface{}) error
}

type Consumer interface {
	Close()
	Recv(ctx context.Context) (<-chan amqp.Delivery, error)
}
