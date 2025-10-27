package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")

	connectionURL := "amqp://guest:guest@localhost:5672/"
	connection , err := amqp.Dial(connectionURL)
	if err != nil {
		fmt.Printf("Failed to connect to RabbitMQ: %v\n", err)
		return
	}
	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		fmt.Printf("Failed to open a channel: %v\n", err)
		return
	}
	defer channel.Close()

	fmt.Println("Connected to RabbitMQ successfully.")

	playState := routing.PlayingState{
		IsPaused: true,
	}
	err = pubsub.PublishJSON(channel,routing.ExchangePerilDirect, routing.PauseKey, playState)
	if err != nil{
		fmt.Printf("Failed to publish playing state: %v\n", err)
		return
	}

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt)
	<-osSignal

	fmt.Print("\nShutting down Peril server...\n")
}
