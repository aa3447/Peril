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

	queueName := "game_logs"
	key := queueName + ".*"
	_, _, err = pubsub.DeclareAndBindQueue(connection, routing.ExchangePerilTopic,queueName,key,pubsub.QueueTypeDurable)
	if err != nil{
		fmt.Printf("Failed to declare and bind queue: %v\n", err)
		return
	}

	fmt.Println("Connected to RabbitMQ successfully.")
	go exitFromOSSignal()
	gamelogic.PrintServerHelp()
	for {
		commands := gamelogic.GetInput()
		if len(commands) > 0 {
			switch commands[0]{
				case "pause":
					fmt.Println("Pausing the game...")
					if publishPlayingState(channel, true) != nil{
						fmt.Println("Failed to publish pause state")
						continue
					}
					fmt.Println("Game paused")
				case "resume":
					fmt.Println("Resuming the game...")
					if publishPlayingState(channel, false) != nil{
						fmt.Println("Failed to publish pause state")
						continue
					}
					fmt.Println("Games resumed")
				case "quit":
					fmt.Println("Quitting the server...")
					return
				default:
					fmt.Println("Unknown command")
					gamelogic.PrintServerHelp()
			}
		}
	}

}

func publishPlayingState(channel *amqp.Channel, isPaused bool) error{
	playState := routing.PlayingState{
		IsPaused: isPaused,
	}
	err := pubsub.PublishJSON(channel,routing.ExchangePerilDirect, routing.PauseKey, playState)
	if err != nil{
		return fmt.Errorf("failed to publish playing state: %v", err)
	}
	return nil
}

func exitFromOSSignal(){
	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt)
	<-osSignal
	fmt.Print("\nShutting down Peril server...\n")
	os.Exit(0)
}