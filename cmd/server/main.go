package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

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

	channel, err := conn.Channel()
	if err != nil {
		log.Printf("couldn't create a channel")
	}

	playingState := routing.PlayingState{IsPaused: true}
	pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, playingState)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Peril server gracefully stopped.")
}
