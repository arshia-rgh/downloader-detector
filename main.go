package main

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/fx"
	"log"
)

func main() {
	app := fx.New(
		fx.Provide(
			RabbitURL,
			InitRabbit,
		),
		fx.Invoke(
			PreDownloadProcess,
			DownloadProcess,
			DetectProcess,
		),
	)
	app.Run()
}

func PreDownloadProcess(lc fx.Lifecycle, ch *amqp.Channel, conn *amqp.Connection) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("Starting PreDownloader")
			go func() {
				err := PreDownloader(ch, "download_queue")
				if err != nil {
					return
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Stopping PreDownloader")
			log.Println("Closing channel")
			ch.Close()
			log.Println("Closing connection")
			conn.Close()
			return nil
		},
	})
}

func DownloadProcess(lc fx.Lifecycle, ch *amqp.Channel, conn *amqp.Connection) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("Starting Downloader")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Stopping Downloader")
			return nil
		},
	})
}

func DetectProcess(lc fx.Lifecycle, ch *amqp.Channel, conn *amqp.Connection) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("Starting Detection")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Stopping Detection")
			return nil
		},
	})
}
