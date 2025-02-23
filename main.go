package main

import (
	"context"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/fx"
	"log"
)

func main() {
	app := fx.New(
		RabbitModule,
		fx.Invoke(
			InitRabbitQueues,
			PreDownloadProcess,
			DownloadProcess,
			DetectProcess,
		),
	)
	app.Run()
}

// PreDownloadProcess Reads the video id and masterURL from a file and send to the download queue
func PreDownloadProcess(lc fx.Lifecycle, ch *amqp.Channel, conn *amqp.Connection) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ShouldPublishChannel := make(chan Message)
			log.Println("Starting PreDownloader")
			go func() {
				err := PreDownloader(ShouldPublishChannel)
				if err != nil {
					return
				}
			}()
			go func() {
				for msg := range ShouldPublishChannel {
					err := PublishMessage(ch, msg, "download_queue")
					if err == nil {
						log.Printf("Message published to the download queue with id : %s", msg.ID)
						continue
					}
					if err != nil {
						log.Printf("error in publishing to the download_queue for video : %s\n", msg.ID)
					}

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

// DownloadProcess Reads the video id and masterURL from the download queue and download the video
// Then Publish the video id, masterURL and filepath to the detect queue
func DownloadProcess(lc fx.Lifecycle, ch *amqp.Channel, conn *amqp.Connection) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ShouldPublishChannel := make(chan Message, 3)
			log.Println("Starting Downloader")
			go func() {
				msgs, err := Consume(ch, "download_queue")
				if err != nil {
					log.Println("error consuming messages from download_queue", err)
					return
				}
				YTDLPDownloader(msgs, ShouldPublishChannel, ch)
			}()
			go func() {
				for msg := range ShouldPublishChannel {
					err := PublishMessage(ch, msg, "detect_queue")
					if err == nil {
						log.Printf("Message published to the detect queue with id : %s", msg.ID)
						continue
					}
					if err != nil {
						log.Printf("error in publishing to the detect_queue for video : %s\n", msg.ID)
					}
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Stopping Downloader")
			log.Println("Closing channel")
			ch.Close()
			log.Println("Closing connection")
			conn.Close()
			return nil
		},
	})
}

// DetectProcess reads from the detect_queue, detect the spots and write to an CSV file with video_id and filepath
func DetectProcess(lc fx.Lifecycle, ch *amqp.Channel, conn *amqp.Connection) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("Starting Detection")
			go func() {
				msgs, err := Consume(ch, "detect_queue")
				if err != nil {
					log.Println("error consuming messages from download_queue", err)
					return
				}
				AiDetection(msgs)
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Stopping Detection")
			log.Println("Closing channel")
			ch.Close()
			log.Println("Closing connection")
			conn.Close()
			return nil
		},
	})
}

func InitRabbitQueues(ch *amqp.Channel) error {
	err := DeclareQueue(ch, "download_queue", 19_800_000)
	if err != nil {
		return fmt.Errorf("failed to declare download_queue: %w", err)
	}
	err = DeclareQueue(ch, "detect_queue", 19_800_000)
	if err != nil {
		return fmt.Errorf("failed to declare detect_queue: %w", err)
	}
	return nil
}
