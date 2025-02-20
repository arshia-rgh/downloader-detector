package main

import (
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
)

type RabbitMQURL string

type Message struct {
	ID        string `json:"id"`
	MasterURL string `json:"master_url"`
}

func Getenv(key, defaultValue string, optional bool) (string, error) {
	value := os.Getenv(key)
	if value != "" {
		return value, nil
	}
	if defaultValue != "" {
		return defaultValue, nil
	}
	if optional {
		return "", nil
	}
	return "", fmt.Errorf("%s is required", key)
}

func RabbitURL() RabbitMQURL {
	url, err := Getenv("RABBIT_URL", "amqp://guest:guest@localhost:5672/", false)
	if err != nil {
		panic(err)
	}
	return RabbitMQURL(url)
}

func InitRabbit(rabbitURL RabbitMQURL) (*amqp.Connection, *amqp.Channel, error) {
	connection, err := amqp.Dial(string(rabbitURL))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}
	ch, err := connection.Channel()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open a channel: %v", err)
	}
	return connection, ch, nil
}

func DeclareQueue(ch *amqp.Channel) error {

}

func PublishMessage(ch *amqp.Channel, message Message) error {

}

func Consume(ch *amqp.Channel) <-chan amqp.Delivery {

}
