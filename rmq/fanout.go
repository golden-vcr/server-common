package rmq

import amqp "github.com/rabbitmq/amqp091-go"

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

// declareConsumerQueue uses an AMQP client to declare a temporary queue for a consumer
// process, then bind it to the exchange with the given name
func declareConsumerQueue(ch *amqp.Channel, exchange string) (*amqp.Queue, error) {
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
