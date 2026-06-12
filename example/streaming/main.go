package main

import (
	"fmt"
	"log"
	"math"

	webrtcvad "github.com/alex-savin/go-webrtcvad"
)

// generateTestAudio generates synthetic audio for demonstration: a tone for the
// first durationMs milliseconds, followed by silence.
func generateTestAudio(samples int, sampleRate int, frequency float64, durationMs int) []byte {
	data := make([]int16, samples)

	for i := 0; i < samples; i++ {
		if i < (sampleRate*durationMs)/1000 { // First part with tone
			t := float64(i) / float64(sampleRate)
			data[i] = int16(8000 * math.Sin(2*math.Pi*frequency*t))
		} else { // Rest silence
			data[i] = 0
		}
	}

	// Convert to 16-bit little-endian bytes.
	bytes := make([]byte, len(data)*2)
	for i, sample := range data {
		bytes[i*2] = byte(sample & 0xff)
		bytes[i*2+1] = byte((sample >> 8) & 0xff)
	}
	return bytes
}

func main() {
	// Create a streaming VAD with 20ms frames at 8kHz.
	stream, err := webrtcvad.NewStreamVAD(8000, 20)
	if err != nil {
		log.Fatal(err)
	}

	// Set aggressive mode for demonstration.
	if err := stream.SetMode(2); err != nil {
		log.Fatal(err)
	}

	// Generate 2 seconds of test audio: 1 second tone + 1 second silence.
	totalSamples := 16000                                          // 2 seconds at 8kHz
	audioData := generateTestAudio(totalSamples, 8000, 1000, 1000) // 1kHz tone for 1 second

	fmt.Println("Processing streaming audio with VAD...")
	fmt.Println("Audio: 1 second of 1kHz tone followed by 1 second of silence")
	fmt.Println("Frame | Speech Detected")
	fmt.Println("------|----------------")

	frameCount := 0
	totalSpeechFrames := 0

	printFrame := func(speechDetected bool, suffix string) {
		frameCount++
		status := "NO"
		if speechDetected {
			status = "YES"
			totalSpeechFrames++
		}
		fmt.Printf("%5d | %s%s\n", frameCount, status, suffix)
	}

	// Process audio in chunks (simulating real-time streaming).
	chunkSize := 640 // 40ms chunks (2 frames worth: 320 samples × 2 bytes)
	for i := 0; i < len(audioData); i += chunkSize {
		end := i + chunkSize
		if end > len(audioData) {
			end = len(audioData)
		}

		results, err := stream.Process(audioData[i:end])
		if err != nil {
			log.Fatal(err)
		}

		for _, speechDetected := range results {
			printFrame(speechDetected, "")
		}
	}

	// Flush any remaining buffered data when the stream ends.
	finalResults, err := stream.Flush()
	if err != nil {
		log.Fatal(err)
	}
	for _, speechDetected := range finalResults {
		printFrame(speechDetected, " (flushed)")
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("Total frames processed: %d\n", frameCount)
	fmt.Printf("Speech frames detected: %d\n", totalSpeechFrames)
	fmt.Printf("Speech ratio: %.1f%%\n", float64(totalSpeechFrames)/float64(frameCount)*100)

	// Expected: ~50 frames of speech (1 second / 20ms) plus some hysteresis.
	if totalSpeechFrames < 40 {
		fmt.Println("Warning: less speech detected than expected")
	}
}
