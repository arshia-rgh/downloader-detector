package main

import (
	"encoding/json"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"os"
	"os/exec"
)

func AiDetection(msgs <-chan amqp.Delivery) {
	detectionSemaphore := make(chan struct{}, 3)
	for msg := range msgs {
		go func() {
			detectionSemaphore <- struct{}{}
			defer func() {
				<-detectionSemaphore
			}()

			var message Message
			err := json.Unmarshal(msg.Body, &message)
			if err != nil {
				fmt.Println("Error unmarshaling JSON:", err)
				msg.Nack(false, true)
				return
			}
			log.Println("received message in detect_queue, id : ", message.ID)

			cmd := exec.Command(
				"./ai/.venv/bin/python3",
				"./ai/internal_find_exact.py",
				"--id", message.ID,
				"--path", message.FilePath,
				"--topk", "100",
			)

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err = cmd.Run()
			if err != nil {
				fmt.Printf("Error running Python script: %s", err)
				msg.Nack(false, true)
				return
			}
			log.Println("Watermarks detected successfully and wrote the result to the csv file")
			msg.Ack(false)
		}()

	}
}

/*
Get messages from `detection_queue` and detect the spots using AI
Write the detected spots to an Excel file with the video_id, filepath and the detected spots
*/
