package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/alex-savin/go-webrtcvad"
)

// generateTestAudio generates synthetic audio for demonstration
func generateTestAudio(samples int, sampleRate int, frequency float64, durationMs int) []byte {
	data := make([]int16, samples)

	for i := 0; i < samples; i++ {
		if i < (sampleRate*durationMs)/1000 { // First part with tone
			t := float64(i) / float64(sampleRate)
			sample := int16(8000 * math.Sin(2*math.Pi*frequency*t))
			data[i] = sample
		} else { // Rest silence
			data[i] = 0
		}
	}

	// Convert to bytes
	bytes := make([]byte, len(data)*2)
	for i, sample := range data {
		bytes[i*2] = byte(sample & 0xff)
		bytes[i*2+1] = byte((sample >> 8) & 0xff)
	}
	return bytes
}

func streamingExample() {
	// Create a streaming VAD with 20ms frames at 8kHz
	stream, err := webrtcvad.NewStreamVAD(8000, 20)
	if err != nil {
		log.Fatal(err)
	}

	// Set aggressive mode for demonstration
	if err := stream.SetMode(2); err != nil {
		log.Fatal(err)
	}

	// Generate 2 seconds of test audio: 1 second tone + 1 second silence
	totalSamples := 16000                                          // 2 seconds at 8kHz
	audioData := generateTestAudio(totalSamples, 8000, 1000, 1000) // 1kHz tone for 1 second

	fmt.Println("Processing streaming audio with VAD...")
	fmt.Println("Audio: 1 second of 1kHz tone followed by 1 second of silence")
	fmt.Println("Frame | Speech Detected")
	fmt.Println("------|----------------")

	frameCount := 0
	totalSpeechFrames := 0

	// Process audio in chunks (simulating real-time streaming)
	chunkSize := 640 // 80ms chunks (4 frames worth)
	for i := 0; i < len(audioData); i += chunkSize {
		end := i + chunkSize
		if end > len(audioData) {
			end = len(audioData)
		}

		chunk := audioData[i:end]
		results, err := stream.Process(chunk)
		if err != nil {
			log.Fatal(err)
		}

		// Report results for each frame
		for _, speechDetected := range results {
			frameCount++
			status := "NO"
			if speechDetected {
				status = "YES"
				totalSpeechFrames++
			}
			fmt.Printf("%5d | %s\n", frameCount, status)
		}

		// Small delay to simulate real-time processing
		time.Sleep(10 * time.Millisecond)
	}

	// Flush any remaining buffered data
	finalResults, err := stream.Flush()
	if err != nil {
		log.Fatal(err)
	}

	for _, speechDetected := range finalResults {
		frameCount++
		status := "NO"
		if speechDetected {
			status = "YES"
			totalSpeechFrames++
		}
		fmt.Printf("%5d | %s (flushed)\n", frameCount, status)
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("Total frames processed: %d\n", frameCount)
	fmt.Printf("Speech frames detected: %d\n", totalSpeechFrames)
	fmt.Printf("Speech ratio: %.1f%%\n", float64(totalSpeechFrames)/float64(frameCount)*100)

	// Expected: ~50 frames of speech (1 second / 20ms) + some hysteresis
	if totalSpeechFrames < 40 {
		fmt.Println("Warning: Less speech detected than expected")
	}
}

// Uncomment to run as standalone program:
// func main() {
//     streamingExample()
// }
