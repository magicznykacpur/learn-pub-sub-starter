package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	log.Println("Peril server gracefully stopped.")
}
