package main

import (
	"encoding/json"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/fx"
	"os"
	"time"
)

var RabbitModule = fx.Module(
	"rabbit",
	fx.Provide(
		fx.Annotate(
			RabbitURL,
			fx.ResultTags(`name:"RABBIT_URL"`),
		),
		fx.Annotate(
			InitRabbit,
			fx.ParamTags(`name:"RABBIT_URL"`),
		),
	),
)

type Message struct {
	ID        string `json:"id"`
	MasterURL string `json:"master_url"`
	FilePath  string `json:"file_path"`
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

func RabbitURL() (string, error) {
	return Getenv("RABBIT_URL", "amqp://guest:guest@localhost:5672/", false)
}

func InitRabbit(rabbitURL string) (*amqp.Connection, *amqp.Channel, error) {
	connection, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}
	ch, err := connection.Channel()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open a channel: %v", err)
	}
	return connection, ch, nil
}

func DeclareQueue(ch *amqp.Channel, queueName string, consumerTimeout time.Duration) error {
	_, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			amqp.ConsumerTimeoutArg: consumerTimeout.Milliseconds(),
		}, // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	return nil
}

func PublishMessage(ch *amqp.Channel, message Message, queueName string) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshalling message: %w", err)
	}

	err = ch.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})
	if err != nil {
		return fmt.Errorf("failed to publish message: %w, id: %s", err, message.ID)
	}

	return nil
}

func Consume(ch *amqp.Channel, queueName string) (<-chan amqp.Delivery, error) {
	err := ch.Qos(3, 0, false) // Prefetch count: 3, Prefetch size: 0, Global: false
	if err != nil {
		return nil, fmt.Errorf("failed to set the qos: %w", err)
	}
	msgs, err := ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to consume: %w", err)
	}
	return msgs, nil
}
