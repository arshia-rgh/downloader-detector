package main

import (
	"encoding/csv"
	"fmt"
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

func PreDownloader(ResultMSG chan<- Message) error {
	data, err := ReadCSV()
	if err != nil {
		return fmt.Errorf("error reading csv: %w", err)
	}

	for _, d := range data {
		ResultMSG <- Message{
			ID:        d.VideoID,
			MasterURL: d.MasterPlaylist,
		}
	}
	return nil
}

/*
   Read from csv file and publish to the download_queue ( masterPlaylist, video_id)
*/
