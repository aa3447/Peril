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

	go exitFromOSSignal()
	gameState := gamelogic.NewGameState(username)
	pauseQueueName := routing.PauseKey + "." + username
	moveQueueName := routing.ArmyMovesPrefix + "." + username
	moveQueueKey := routing.ArmyMovesPrefix + ".*"

	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilDirect, pauseQueueName, routing.PauseKey, pubsub.QueueTypeTransient, handlerPause(gameState))
	if err != nil{
		fmt.Printf("Failed to subscribe to pause messages: %v\n", err)
		return
	}
	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilTopic, moveQueueName, moveQueueKey, pubsub.QueueTypeTransient, handlerMove(gameState))
	if err != nil{
		fmt.Printf("Failed to subscribe to move messages: %v\n", err)
		return
	}
	channel, err := connection.Channel()
	if err != nil{
		fmt.Printf("Failed to open a channel: %v\n", err)
		return
	}
	defer channel.Close()
	
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
					moveStruct, err := gameState.CommandMove(commands)
					if err != nil{
						fmt.Println("Failed to move unit:", err)
						continue
					}
					err = pubsub.PublishJSON(channel, routing.ExchangePerilTopic, moveQueueKey, moveStruct)
					if err != nil{
						fmt.Printf("failed to publish playing state: %v", err)
						continue
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
	fmt.Print("\nShutting down Peril client...\n")
	os.Exit(0)
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState){
	return func(ps routing.PlayingState){
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove){
	return func(am gamelogic.ArmyMove){
		defer fmt.Print("> ")
		gs.HandleMove(am)
	}
}

