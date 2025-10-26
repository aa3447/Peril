package main

import (
	"fmt"
	"os"
	"os/signal"

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

	fmt.Println("Connected to RabbitMQ successfully.")

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt)
	<-osSignal

	fmt.Print("\nShutting down Peril server...\n")
}
