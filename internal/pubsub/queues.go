package pubsub

import (
	"fmt"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int
const (
	QueueTypeDurable   SimpleQueueType = 1
	QueueTypeTransient SimpleQueueType = 2
)

func DeclareAndBindQueue(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error){
	var durable, autoDelete, exclusive bool

	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("failed to open channel: %v", err)
	}

	switch queueType{
		case QueueTypeDurable:
			durable = true
			autoDelete = false
			exclusive = false
		case QueueTypeTransient:
			durable = false
			autoDelete = true
			exclusive = true
		default:
			return nil, amqp.Queue{}, fmt.Errorf("unknown queue type: %v", queueType)
	}

	
	queue, err := channel.QueueDeclare(queueName,durable,autoDelete,exclusive,false,nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("failed to declare queue: %v", err)
	}

	err = channel.QueueBind(queueName,key,exchange,false,nil)
	if err != nil{
		return nil, amqp.Queue{}, fmt.Errorf("failed to bind queue: %v", err)
	}

	return channel, queue, nil
}

// SubscribeJSON sets up a subscription to a queue and processes incoming messages using the provided handler function.
// Declares and binds the queue if it does not already exist.
// Accepts any struct type T for message deserialization.
func SubscribeJSON[T any](
    conn *amqp.Connection,
    exchange,
    queueName,
    key string,
    queueType SimpleQueueType, // an enum to represent "durable" or "transient"
    handler func(T),
) error {
	channel, queue, err := DeclareAndBindQueue(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("failed to declare and bind queue: %v", err)
	}

	msgs, err := channel.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to receive messages: %v", err)
	}

	go func(){
		for msg := range msgs{
			var genericMsgStruct T
			err := json.Unmarshal(msg.Body, &genericMsgStruct)
			if err != nil{
				fmt.Printf("Failed to unmarshal message: %v\n", err)
				msg.Nack(false, false)
				continue
			}
			handler(genericMsgStruct)
			msg.Ack(false)
		}
	}()

	return nil
}