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
	log.Println("Starting Peril server...")

	channel, _, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, routing.GameLogSlug, "game_logs.*", 1)
	if err != nil {
		log.Printf("couldn't declare and bind channel and queue: %v", err)
	}

	gamelogic.PrintServerHelp()

	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}

		if input[0] == "pause" {
			fmt.Println("Sending pause message...")
			playingState := routing.PlayingState{IsPaused: true}
			pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, playingState)
		}

		if input[0] == "resume" {
			fmt.Println("Sending resume message...")
			playingState := routing.PlayingState{IsPaused: false}
			pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, playingState)
		}

		if input[0] == "quit" {
			fmt.Println("Exitting the server...")
			break
		}

		if !slices.Contains([]string{"pause", "resume", "quit"}, input[0]) {
			fmt.Printf("Command does not exist: %s", input[0])
			continue
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Peril server gracefully stopped.")
}
