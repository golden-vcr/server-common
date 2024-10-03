package rmq

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
