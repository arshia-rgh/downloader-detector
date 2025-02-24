package main

import (
	"encoding/json"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func AiDetection(msgs <-chan amqp.Delivery) {
	for msg := range msgs {
		var message Message
		err := json.Unmarshal(msg.Body, &message)
		if err != nil {
			fmt.Println("Error unmarshaling JSON:", err)
			msg.Nack(false, true)
			continue
		}
		log.Println("received message in detect_queue, id : ", message.ID)

		outputFilePath := strings.TrimSuffix(message.FilePath, filepath.Ext(message.FilePath)) + "_edited" + filepath.Ext(message.FilePath)

		err = PrePareFileForDetection(message.FilePath, outputFilePath, "./best_gain_watermark_00.wav", "./best_gain_watermark_01.wav")
		if err != nil {
			fmt.Println("Error preparing file for detection:", err)
			msg.Nack(false, true)
			continue
		}

		cmd := exec.Command(
			"./ai/.venv/bin/python3",
			"./ai/internal_find_exact.py",
			"--id", message.ID,
			"--path", outputFilePath,
			"--topk", "2",
		)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Run()
		if err != nil {
			fmt.Printf("Error running Python script: %s", err)
			msg.Nack(false, true)
			continue
		}
		log.Println("Watermarks detected successfully and wrote the result to the csv file")
		msg.Ack(false)

	}
}

// GetFileDuration returns the duration of the input file in seconds
func GetFileDuration(inputFilePath string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", inputFilePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("error running ffprobe: %w", err)
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing duration: %w", err)
	}
	return duration, nil
}

// PrePareFileForDetection adds two voices to the input file
func PrePareFileForDetection(inputFilePath, outputFilePath, voiceFilePath1, voiceFilePath2 string) error {
	duration, err := GetFileDuration(inputFilePath)
	if err != nil {
		return fmt.Errorf("error getting file duration: %w", err)
	}

	// Calculate the start time for the second voice (last 12 minutes)
	startTimeSecondVoice := duration - 720

	voiceDuration1, err := GetFileDuration(voiceFilePath1)
	if err != nil {
		return fmt.Errorf("error getting voice file duration: %w", err)
	}
	voiceDuration2, err := GetFileDuration(voiceFilePath2)
	if err != nil {
		return fmt.Errorf("error getting voice file duration: %w", err)
	}

	// Command to add voices at the 12th minute and the last 12 minutes
	cmd2 := exec.Command(
		"ffmpeg",
		"-i", inputFilePath,
		"-i", voiceFilePath1,
		"-i", voiceFilePath2,
		"-filter_complex", fmt.Sprintf(
			"[0]volume=enable='between(t,720,%f)':volume=0,volume=enable='between(t,%f,%f)':volume=0[silenced];"+
				"[1]adelay=720000|720000[aud1];"+
				"[2]adelay=%d|%d[aud2];"+
				"[silenced][aud1]amix=inputs=2:duration=first[intermediate];"+
				"[intermediate][aud2]amix=inputs=2:duration=first",
			720+voiceDuration1, startTimeSecondVoice, startTimeSecondVoice+voiceDuration2,
			int(startTimeSecondVoice*1000), int(startTimeSecondVoice*1000)),
		"-c:v", "copy",
		outputFilePath,
	)

	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr

	err = cmd2.Run()
	if err != nil {
		return fmt.Errorf("error running ffmpeg: %w", err)
	}

	log.Println("Voices added successfully at the specified times")
	return nil
}
