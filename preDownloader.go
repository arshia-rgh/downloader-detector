package main

import (
	"encoding/csv"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"os"
)

type FileData struct {
	MasterPlaylist string
	VideoID        string
}

func ReadCSV() ([]FileData, error) {
	file, err := os.Open("data.csv")
	if err != nil {
		return nil, fmt.Errorf("error opening csv: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading csv: %w", err)
	}

	var data []FileData
	for _, record := range records[1:] {
		data = append(data, FileData{
			MasterPlaylist: record[24],
			VideoID:        record[0],
		})
	}
	return data, nil

}

func PreDownloader(ch *amqp.Channel, queueName string) error {
	data, err := ReadCSV()
	if err != nil {
		return fmt.Errorf("error reading csv: %w", err)
	}

	for _, d := range data {
		err := PublishMessage(ch, Message{
			ID:        d.VideoID,
			MasterURL: d.MasterPlaylist,
		}, queueName)
		if err != nil {
			return fmt.Errorf("error publishing message: %w", err)
		}
		log.Println("Published message with video id: ", d.VideoID)
	}
	return nil
}

/*
Read from csv file and publish to the download_queue ( masterPlaylist, video_id)
*/
