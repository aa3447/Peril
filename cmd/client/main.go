package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	connectionURL := "amqp://guest:guest@localhost:5672/"
	connection , err := amqp.Dial(connectionURL)
	if err != nil {
		fmt.Printf("Failed to connect to RabbitMQ: %v\n", err)
		return
	}
	defer connection.Close()


	username, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Printf("Error during welcome: %v\n", err)
		return
	}

	queueName := routing.PauseKey + "." + username
	_, _, err = pubsub.DeclareAndBindTransient(connection, routing.ExchangePerilDirect,queueName,routing.PauseKey,pubsub.QueueTypeTransient)
	if err != nil{
		fmt.Printf("Failed to declare and bind queue: %v\n", err)
		return
	}

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt)
	<-osSignal
	fmt.Print("\nShutting down Peril server...\n")
	os.Exit(0)
}
