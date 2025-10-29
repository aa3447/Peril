package pubsub

import (
	"fmt"

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
	defer channel.Close()

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
