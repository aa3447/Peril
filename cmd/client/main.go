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
	_, _, err = pubsub.DeclareAndBindQueue(connection, routing.ExchangePerilDirect,queueName,routing.PauseKey,pubsub.QueueTypeTransient)
	if err != nil{
		fmt.Printf("Failed to declare and bind queue: %v\n", err)
		return
	}

	go exitFromOSSignal()
	gameState := gamelogic.NewGameState(username)
	for {
		commands := gamelogic.GetInput()
		lenCommands := len(commands)
		if lenCommands > 0 {
			switch commands[0]{
				case "spawn":
					if lenCommands < 3{
						fmt.Println("Not enough arguments for spawn command")
						continue
					}
					if gameState.CommandSpawn(commands) != nil{
						fmt.Println("Failed to spawn unit")
					}
				case "move":
					if lenCommands < 3{
						fmt.Println("Not enough arguments for move command")
						continue
					}
					_, err := gameState.CommandMove(commands)
					if err != nil{
						fmt.Println("Failed to move unit:", err)
					}
				case "status":
					gameState.CommandStatus()
				case "help":
					gamelogic.PrintClientHelp()
				case "spam":
					fmt.Println("Spamming not allowed yet!")
				case "quit":
					gamelogic.PrintQuit()
					return
				default:
					fmt.Println("Unknown command")
					gamelogic.PrintClientHelp()
			}
		}
	}
}

func exitFromOSSignal(){
	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt)
	<-osSignal
	fmt.Print("\nShutting down Peril server...\n")
	os.Exit(0)
}
