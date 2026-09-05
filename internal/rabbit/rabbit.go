// Package rabbit wraps amqp091 with a topic exchange and helpers for
// publish/consume used across ingesters and the aggregator.
package rabbit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

const Exchange = "election.events"

// Routing keys.
const (
	KeyPollRaw    = "poll.raw"
	KeyGeoResult  = "geo.result"
	KeyFetchPolls = "cmd.fetch.polls"   // scheduler -> poll-ingester
	KeyFetchGeo   = "cmd.fetch.geo"     // scheduler -> results-ingester
)

type Conn struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// Dial connects with retry (RabbitMQ may still be booting in compose).
func Dial(url string) (*Conn, error) {
	var conn *amqp.Connection
	var err error
	for i := 0; i < 30; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Warn().Err(err).Msg("rabbitmq not ready, retrying")
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("dial amqp: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		return nil, err
	}
	return &Conn{conn: conn, ch: ch}, nil
}

func (c *Conn) Close() {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// Publish marshals v as JSON and publishes with the given routing key.
func (c *Conn) Publish(ctx context.Context, key string, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.ch.PublishWithContext(ctx, Exchange, key, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
	})
}

// Consume binds a durable queue to the given routing keys and returns a delivery channel.
func (c *Conn) Consume(queue string, keys ...string) (<-chan amqp.Delivery, error) {
	q, err := c.ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		if err := c.ch.QueueBind(q.Name, k, Exchange, false, nil); err != nil {
			return nil, err
		}
	}
	if err := c.ch.Qos(20, 0, false); err != nil {
		return nil, err
	}
	return c.ch.Consume(q.Name, "", false, false, false, false, nil)
}
