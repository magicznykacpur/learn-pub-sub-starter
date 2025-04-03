package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	bytes, err := json.Marshal(val)
	if err != nil {
		return err
	}

	return ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{ContentType: "application/json", Body: bytes},
	)
}

type queueType int

const (
	transient queueType = iota
	nonTransient
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType queueType,
) (*amqp.Channel, amqp.Queue, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	var queue amqp.Queue

	if simpleQueueType == transient {
		queue, err = channel.QueueDeclare(queueName, true, true, true, false, nil)
		if err != nil {
			return nil, amqp.Queue{}, err
		}
	}

	if simpleQueueType == nonTransient {
		queue, err = channel.QueueDeclare(queueName, false, false, false, false, nil)
		if err != nil {
			return nil, amqp.Queue{}, err
		}
	}

	err = channel.QueueBind(queue.Name, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return channel, queue, nil
}
