package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"os"
	"os/exec"

	"path/filepath"
	"strings"
)

func YTDLPDownloader(msgs <-chan amqp.Delivery, ResultMSG chan<- Message, ch *amqp.Channel) {
	downloadSemaphore := make(chan struct{}, 3)
	for msg := range msgs {
		go func() {
			downloadSemaphore <- struct{}{}
			defer func() {
				<-downloadSemaphore
			}()

			var message Message
			err := json.Unmarshal(msg.Body, &message)
			if err != nil {
				log.Println("Error unmarshaling JSON:", err)
				msg.Nack(false, true) // Nack the message and requeue it
				return
			}
			log.Println("received message in download_queue, id : ", message.ID)

			dirUUID := uuid.New().String()
			dirName := fmt.Sprintf("%s-%s", dirUUID, message.ID)
			dirPath := filepath.Join("videos", dirName)
			os.MkdirAll(dirPath, os.ModePerm)

			audioFormats, err := ExtractAudioFormats(message.MasterURL)
			if err != nil {
				log.Println("Error extracting audio formats:", err)
				msg.Nack(false, true) // Nack the message and requeue it
				return
			}

			err = DownloadWorker(message.MasterURL, dirPath, audioFormats)
			if err != nil {
				log.Println("Error downloading:", err)
				msg.Nack(false, true) // Nack the message and requeue it
				os.RemoveAll(dirPath)
				return
			}
			log.Println("Download completed for video:", message.ID)
			files, err := os.ReadDir(dirPath)
			if err != nil {
				log.Println("error reading dir", err)
				msg.Nack(false, true)
				os.RemoveAll(dirPath)
				return
			}
			for _, file := range files {
				if !file.IsDir() {
					path := filepath.Join(dirPath, file.Name())
					ResultMSG <- Message{
						ID:        message.ID,
						MasterURL: message.MasterURL,
						FilePath:  path,
					}
				}
			}
			msg.Ack(false)
		}()
	}
}

func DownloadWorker(masterURL, dirPath string, audioFormats []string) error {
	log.Printf("startig download with %s audio formats", audioFormats)
	outputTemplate := fmt.Sprintf("%s/%%(height)sp-%%(format_id)s.%%(ext)s", dirPath)
	args := []string{"--all-subs", "--write-subs", "--write-auto-subs", "--sub-langs", "all", "-o", outputTemplate, masterURL}

	if len(audioFormats) == 1 { // means the video has audio
		args = append([]string{"-f", fmt.Sprintf("%s,%s", audioFormats[0], "240p")}, args...) // download requested quality + audios
	} else if len(audioFormats) == 2 {
		args = append([]string{"-f", fmt.Sprintf("%s,%s,%s", audioFormats[0], audioFormats[1], "240p")}, args...) // download requested quality + audios
	} else {
		args = append([]string{"-f", "240p"}, args...) // download requested quality ( doesnt have any audios)
	}

	cmd := exec.Command("yt-dlp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error running yt-dlp: %w", err)
	}
	return nil
}

func ExtractAudioFormats(masterURL string) ([]string, error) {
	// List available formats
	listCmd := exec.Command("yt-dlp", "-F", masterURL)
	output, err := listCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error listing formats: %w", err)
	}

	// Check for the audios
	// Check for the audios
	var audioFormats []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "audio") || strings.Contains(line, "audio") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				audioFormats = append(audioFormats, parts[0])
			}
		}
		if len(audioFormats) >= 2 {
			break
		}
	}
	return audioFormats, nil
}

func CheckAndExtractDownloadedFilesData(dir, masterURL string) (string, map[int][2]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, fmt.Errorf("error reading directory: %w", err)
	}

	var videoFilePath string
	audioFiles := make(map[int][2]string)

	for i, file := range files {
		if !file.IsDir() {
			fileInfo, err := file.Info()
			if err != nil {
				return "", nil, fmt.Errorf("error getting file information: %w", err)
			}

			filePath := filepath.Join(dir, fileInfo.Name())
			//fileSize := fmt.Sprintf("%d", fileInfo.Size())

			if strings.Contains(fileInfo.Name(), "240p") {
				videoFilePath = filePath
			} else {
				// Extract the relevant part of the file name
				fileName := fileInfo.Name()
				if strings.HasSuffix(fileName, ".mp4") {
					fileName = fileName[:len(fileName)-4] // Remove the .mp4 extension
				}
				if strings.HasPrefix(fileName, "NAp-") {
					fileName = fileName[4:] // Remove the NAp- prefix
				}

				// Use the ExtractAudioLanguage function
				language, err := ExtractAudioLanguage(masterURL, fileName)
				if err != nil {
					return "", nil, fmt.Errorf("error extracting audio language: %w", err)
				}

				audioFiles[i] = [2]string{filePath, language}

			}

		}
	}
	return videoFilePath, audioFiles, nil
}

func ExtractAudioLanguage(masterURL, audioName string) (string, error) {
	// List available formats
	listCmd := exec.Command("yt-dlp", "-F", masterURL)
	output, err := listCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error listing formats: %w", err)
	}

	// Check for the specified audio name
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, audioName) {
			parts := strings.Fields(line)
			// Extract the language from the MORE column
			moreInfo := strings.Join(parts[7:], " ")
			if strings.Contains(moreInfo, "[en]") {
				return "en", nil
			} else if strings.Contains(moreInfo, "[fa]") {
				return "fa", nil
			} else {
				// Extract other languages
				start := strings.Index(moreInfo, "[")
				end := strings.Index(moreInfo, "]")
				if start != -1 && end != -1 && end > start {
					return moreInfo[start+1 : end], nil
				}
			}
		}
	}
	return "unknown", nil
}
