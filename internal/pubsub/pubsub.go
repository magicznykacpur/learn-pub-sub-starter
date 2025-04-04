package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"

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

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(val)
	if err != nil {
		return err
	}

	return ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{ContentType: "application/gob", Body: buffer.Bytes()},
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
	queueTable := amqp.Table{}
	queueTable["x-dead-letter-exchange"] = "peril_dlx"

	if simpleQueueType == transient {
		queue, err = channel.QueueDeclare(queueName, true, true, false, false, queueTable)
		if err != nil {
			return nil, amqp.Queue{}, err
		}
	}

	if simpleQueueType == nonTransient {
		queue, err = channel.QueueDeclare(queueName, true, false, false, false, nil)
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

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType queueType,
	handler func(T) AckType,
) error {
	return subscribe(conn, exchange, queueName, key, simpleQueueType, handler, unmarshalJSON)
}

func SubscribeGOB[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType queueType,
	handler func(T) AckType,
) error {
	return subscribe(conn, exchange, queueName, key, simpleQueueType, handler, unmarshalGOB)
}

func unmarshalGOB[T any](toUnmarshal []byte) (T, error) {
	var body T
	buffer := bytes.NewBuffer(toUnmarshal)
	decoder := gob.NewDecoder(buffer)

	err := decoder.Decode(&body)
	if err != nil {
		return body, err
	}

	return body, nil
}

func unmarshalJSON[T any](toUnmarshal []byte) (T, error) {
	var body T

	err := json.Unmarshal(toUnmarshal, &body)
	if err != nil {
		return body, err
	}

	return body, nil
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType queueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	channel, queue, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return err
	}

	messages, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for message := range messages {
			body, err := unmarshaller(message.Body)
			if err != nil {
				fmt.Printf("cannot unmarshall delivery message body: %v", err)
			}

			ackType := handler(body)

			if ackType == Ack {
				message.Ack(false)
			}

			if ackType == NackRequeue {
				message.Nack(false, true)
			}

			if ackType == NackDiscard {
				message.Nack(false, false)
			}
		}
	}()

	return nil
}
