package pubsub

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int
const (
	QueueTypeDurable   SimpleQueueType = 1
	QueueTypeTransient SimpleQueueType = 2
)

type AnkType int
const (
	Ack AnkType = 1
	NackRequeue AnkType = 2
	NackDiscard AnkType = 3
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

	args := amqp.Table{
		"x-dead-letter-exchange": routing.ExchangePerilDLX,
	}
	queue, err := channel.QueueDeclare(queueName,durable,autoDelete,exclusive,false,args)
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
    handler func(T)(AnkType),
)  error {
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
		return  fmt.Errorf("failed to receive messages: %v", err) 
	}

	go func(){
		for msg := range msgs{
			var genericMsgStruct T
			err := json.Unmarshal(msg.Body, &genericMsgStruct)
			if err != nil{
				fmt.Printf("Failed to unmarshal message: %v\n", err)
				log.Printf("%v\n", msg.Body)
				msg.Nack(false, false)
				continue
			}
			ank := handler(genericMsgStruct)
			switch ank{
				case Ack:
					msg.Ack(false)
					log.Println("Message processed and acknowledged.")
				case NackRequeue:
					msg.Nack(false, true)
					log.Println("Message processing failed, message requeued.")
				case NackDiscard:
					msg.Nack(false, false)
					log.Println("Message processing failed, message discarded.")
			}
		}
	}()

	return nil
}