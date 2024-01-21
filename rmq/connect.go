package rmq

import (
	"fmt"
	"net/url"
)

// FormatConnectionString builds a URI that will permit an AMQP client to connect to the
// RabbitMQ server described by the provided config values
func FormatConnectionString(host string, port int, vhost, user, password string) string {
	urlencodedPassword := url.QueryEscape(password)
	return fmt.Sprintf("amqp://%s:%s@%s:%d/%s", user, urlencodedPassword, host, port, vhost)
}
