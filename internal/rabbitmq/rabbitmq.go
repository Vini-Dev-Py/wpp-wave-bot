package rabbitmq

import (
	"fmt"

	"github.com/streadway/amqp"
)

// RabbitMQ wraps an AMQP connection and channels
// to publish and consume messages
// Using a single connection for simplicity

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// New creates a connection to RabbitMQ
func New(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}
	return &RabbitMQ{conn: conn, channel: ch}, nil
}

// Close shuts down channel and connection
func (r *RabbitMQ) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}

// Consume returns a delivery channel for the given queue
func (r *RabbitMQ) Consume(queue string) (<-chan amqp.Delivery, error) {
	return r.channel.Consume(queue, "", true, false, false, false, nil)
}

// Publish sends a message to an exchange with routing key
func (r *RabbitMQ) Publish(exchange, key string, body []byte) error {
	return r.channel.Publish(exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
