package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"slices"
	"syscall"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

const guestUrl = "amqp://guest:guest@localhost:5672/"

func main() {
	log.Println("Connecting to rabbitMq server...")

	conn, err := amqp.Dial(guestUrl)
	if err != nil {
		log.Printf("couldnt dial connection with %s: %v\n", guestUrl, err)
	}
	defer conn.Close()

	log.Println("Connection to rabbitMq server successfull!")
	log.Println("Starting Peril client...")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Printf("couldn't get client welcome message: %v", err)
	}

	channel, _, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, username),
		routing.PauseKey,
		0,
	)
	if err != nil {
		log.Printf("couldn't declare and bind channel: %v", err)
	}

	gameState := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, username),
		routing.PauseKey,
		0,
		handlerPause(gameState),
	)
	if err != nil {
		log.Printf("couldn't subscribe to %s: %v", routing.ExchangePerilDirect, err)
	}

	pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username),
		fmt.Sprintf("%s.*", routing.ArmyMovesPrefix),
		0,
		handlerMove(gameState),
	)
	if err != nil {
		log.Printf("couldn't subscribe to %s: %v", routing.ExchangePerilTopic, err)
	}

	for {
		input := gamelogic.GetInput()

		if input[0] == "spawn" {
			if len(input) != 3 {
				fmt.Println("invalid command arguments: spawn <location> <type>")
				continue
			}

			err := gameState.CommandSpawn(input)
			if err != nil {
				fmt.Printf("couldn't spawn unit: %v\n", err)
			}
		}

		if input[0] == "move" {
			if len(input) != 3 {
				fmt.Println("invalid command arguments: move <location> <to>")
			}

			armyMove, err := gameState.CommandMove(input)
			if err != nil {
				fmt.Printf("couldn't move: %v\n", err)
			}

			err = pubsub.PublishJSON(
				channel,
				routing.ExchangePerilTopic,
				fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username),
				armyMove,
			)
			if err != nil {
				fmt.Printf("coudln't publish move: %v", err)
			}

			fmt.Println("move was published successfully")
		}

		if input[0] == "status" {
			gameState.CommandStatus()
		}

		if input[0] == "help" {
			gamelogic.PrintClientHelp()
		}

		if input[0] == "spam" {
			fmt.Println("Spamming is not allowed yet!")
		}

		if input[0] == "quit" {
			gamelogic.PrintQuit()
			break
		}

		if !slices.Contains([]string{"spawn", "move", "status", "help", "spam", "quit"}, input[0]) {
			fmt.Printf("Unknown command: %s\n", input[0])
			continue
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Peril client gracefully stopped.")
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Printf("> ")
		gs.HandlePause(ps)
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) {
	return func(am gamelogic.ArmyMove) {
		defer fmt.Printf("> ")
		gs.HandleMove(am)
	}
}
