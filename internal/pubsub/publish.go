package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error{
	marshaledVal, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %v", err)
	}

	msg := amqp.Publishing{
		ContentType: "application/json",
		Body: marshaledVal,
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, msg)
	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}

	return nil
}
