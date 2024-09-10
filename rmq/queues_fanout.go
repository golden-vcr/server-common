package rmq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// declareFanoutExchange uses an AMQP client to declare an fanout exchange for a simple
// message queueing scheme in which any number of producers can send messages to a named
// exchange, and any number of consumers can receive messages by binding their own
// temporary queue to that exchange
func declareFanoutExchange(ch *amqp.Channel, exchange string) error {
	durable := true
	autoDelete := false
	internal := false
	noWait := false
	return ch.ExchangeDeclare(exchange, "fanout", durable, autoDelete, internal, noWait, nil)
}

// declareFanoutConsumerQueue uses an AMQP client to declare a temporary queue for a
// consumer process, then bind it to the exchange with the given name
func declareFanoutConsumerQueue(ch *amqp.Channel, exchange string) (*amqp.Queue, error) {
	durable := false
	autoDelete := false
	exclusive := true
	noWait := false
	q, err := ch.QueueDeclare("", durable, autoDelete, exclusive, noWait, nil)
	if err != nil {
		return nil, err
	}

	noWait = false
	if err := ch.QueueBind(q.Name, "", exchange, noWait, nil); err != nil {
		return nil, err
	}
	return &q, nil
}

// fanoutProducer is an rmq.Producer implementation that publishes to the configured
// fanout exchange
type fanoutProducer struct {
	conn     *amqp.Connection
	exchange string
}

func (p *fanoutProducer) Send(ctx context.Context, data interface{}) error {
	// Serialize the message to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Prepare a channel to send our message
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Publish to the fanout exchange, which will ensure that a copy of the message is
	// sent to all consumers which have bound their transient queues to that exchange
	mandatory := false
	immediate := false
	return ch.PublishWithContext(ctx, p.exchange, "", mandatory, immediate, amqp.Publishing{
		ContentType: "application/json",
		Body:        jsonData,
	})
}

func (d *QueueDeclaration) newFanoutProducer(conn *amqp.Connection, ch *amqp.Channel) (Producer, error) {
	if err := declareFanoutExchange(ch, d.Name); err != nil {
		return nil, fmt.Errorf("failed to declare fanout exchange '%s': %w", d.Name, err)
	}
	return &fanoutProducer{
		conn:     conn,
		exchange: d.Name,
	}, nil
}

// fanoutConsumer is an rmq.Consumer implementation that receives messages from a
// temporary queue bound to a fanout exchange
type fanoutConsumer struct {
	ch       *amqp.Channel
	q        *amqp.Queue
	exchange string
}

func (c *fanoutConsumer) Close() {
	c.ch.Close()
}

func (c *fanoutConsumer) Recv(ctx context.Context) (<-chan amqp.Delivery, error) {
	autoAck := false
	exclusive := false
	noLocal := false
	noWait := false
	return c.ch.ConsumeWithContext(ctx, c.q.Name, "", autoAck, exclusive, noLocal, noWait, nil)
}

func (d *QueueDeclaration) newFanoutConsumer(ch *amqp.Channel) (Consumer, error) {
	if err := declareFanoutExchange(ch, d.Name); err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to declare fanout exchange '%s': %w", d.Name, err)
	}
	q, err := declareFanoutConsumerQueue(ch, d.Name)
	if err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to declare consumer queue for fanout exchange '%s': %w", d.Name, err)
	}
	return &fanoutConsumer{
		ch:       ch,
		q:        q,
		exchange: d.Name,
	}, nil
}
