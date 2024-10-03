package rmq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// declareWorkQueue uses an AMQP client to declare a simple persistent queue that can be
// used to distribute messages to worker processes
func declareWorkQueue(ch *amqp.Channel, name string) (*amqp.Queue, error) {
	durable := false
	autoDelete := false
	exclusive := false
	noWait := false
	q, err := ch.QueueDeclare(name, durable, autoDelete, exclusive, noWait, nil)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// workProducer is an rmq.Producer implementation that publishes messages to a work
// queue
type workProducer struct {
	conn *amqp.Connection
	q    *amqp.Queue
}

func (p *workProducer) Send(ctx context.Context, data interface{}) error {
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

	// Publish directly to the queue, which will choose a single consumer to dispatch the
	// message to
	mandatory := false
	immediate := false
	return ch.PublishWithContext(ctx, "", p.q.Name, mandatory, immediate, amqp.Publishing{
		ContentType: "application/json",
		Body:        jsonData,
	})
}

func (d *QueueDeclaration) newWorkProducer(conn *amqp.Connection, ch *amqp.Channel) (Producer, error) {
	q, err := declareWorkQueue(ch, d.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to declare work queue '%s': %w", d.Name, err)
	}
	return &workProducer{
		conn: conn,
		q:    q,
	}, nil
}

// workReceiver is an rmq.Receiver implementation that contends with other consumers to
// receive messages from a work queue
type workReceiver struct {
	ch *amqp.Channel
	q  *amqp.Queue
}

func (c *workReceiver) Close() {
	c.ch.Close()
}

func (c *workReceiver) Recv(ctx context.Context) (<-chan amqp.Delivery, error) {
	autoAck := false
	exclusive := false
	noLocal := false
	noWait := false
	return c.ch.ConsumeWithContext(ctx, c.q.Name, "", autoAck, exclusive, noLocal, noWait, nil)
}

func (d *QueueDeclaration) newWorkReceiver(ch *amqp.Channel) (Receiver, error) {
	q, err := declareWorkQueue(ch, d.Name)
	if err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to declare work queue '%s': %w", d.Name, err)
	}
	return &workReceiver{
		ch: ch,
		q:  q,
	}, nil
}
