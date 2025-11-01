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
	warKey := routing.WarRecognitionsPrefix + ".*"

	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilDirect, pauseQueueName, routing.PauseKey, pubsub.QueueTypeTransient, handlerPause(gameState))
	if err != nil{
		fmt.Printf("Failed to subscribe to pause messages: %v\n", err)
		return
	}
	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilTopic, moveQueueName, moveQueueKey, pubsub.QueueTypeTransient, handlerMove(gameState, connection))
	if err != nil{
		fmt.Printf("Failed to subscribe to move messages: %v\n", err)
		return
	}
	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilTopic, "war", warKey, pubsub.QueueTypeDurable, handlerWar(gameState))
	if err != nil{
		fmt.Printf("Failed to subscribe to war messages: %v\n", err)
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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState)(pubsub.AnkType){
	return func(ps routing.PlayingState)(pubsub.AnkType){
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, connection *amqp.Connection) func(gamelogic.ArmyMove)(pubsub.AnkType){
	return func(am gamelogic.ArmyMove)(pubsub.AnkType){
		defer fmt.Print("> ")
		moveOutCome := gs.HandleMove(am)
		switch moveOutCome{
			case gamelogic.MoveOutComeSafe:
				return pubsub.Ack
			case gamelogic.MoveOutcomeMakeWar:
				warKey := routing.WarRecognitionsPrefix + "." + am.Player.Username
				
				channel, err := connection.Channel()
				if err != nil{
					fmt.Printf("Failed to open a channel: %v\n", err)
					return pubsub.NackRequeue
				}
				defer channel.Close()
				
				warDec := gamelogic.RecognitionOfWar{
					Attacker: am.Player,
					Defender: gs.GetPlayerSnap(),
				}
				err = pubsub.PublishJSON(channel, routing.ExchangePerilTopic, warKey, warDec)
				if err != nil{
					fmt.Printf("failed to publish playing state: %v", err)
					return pubsub.NackRequeue
				}
				
				return pubsub.Ack
			default:
				fmt.Println("Move failed:", moveOutCome)
				return pubsub.NackDiscard
		}	
	}
}

func handlerWar(gs *gamelogic.GameState) func(gamelogic.RecognitionOfWar)(pubsub.AnkType){
	return func(row gamelogic.RecognitionOfWar)(pubsub.AnkType){
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(row)
		switch outcome{
			case gamelogic.WarOutcomeNotInvolved:
				return pubsub.NackRequeue
			case gamelogic.WarOutcomeNoUnits:
				return pubsub.NackDiscard
			case gamelogic.WarOutcomeOpponentWon:
				fmt.Printf("You lost the war against %s. Better luck next time!\n", winner)
				return pubsub.Ack
			case gamelogic.WarOutcomeYouWon:
				fmt.Printf("Congratulations! You won the war against %s!\n", loser)
				return pubsub.Ack
			case gamelogic.WarOutcomeDraw:
				fmt.Println("The war ended in a draw. No one wins!")
				return pubsub.Ack
			default:
				fmt.Println("Unknown war outcome")
				return pubsub.NackDiscard
		}
	}
}

