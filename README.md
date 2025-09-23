# Go WebRTC VAD

[![Go Reference](https://pkg.go.dev/badge/github.com/alex-savin/go-webrtcvad.svg)](https://pkg.go.dev/github.com/alex-savin/go-webrtcvad)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Go port of [py-webrtcvad](https://github.com/wiseman/py-webrtcvad) Voice Activity Detector (VAD).

A VAD classifies a piece of audio data as being voiced or unvoiced. It can be useful for telephony and speech recognition.

## Overview

A VAD classifies a piece of audio data as being voiced or unvoiced. It is highly beneficial for telephony systems (like FreeSWITCH) and speech recognition pipelines, particularly when dealing with real-time audio streams.

The VAD originally developed for the WebRTC project by Google is one of the best available, being:

- **Fast** - Optimized for real-time processing
- **Modern** - Based on current state-of-the-art algorithms  
- **Free** - Open source with MIT license

## Installation

Go-get the package. You don't need to have WebRTC installed.

```bash
go get github.com/alex-savin/go-webrtcvad
```

## Usage

### Example: Processing Audio in FreeSWITCH Environments

This example demonstrates how to use the VAD with raw audio data, tailored for PBX systems:

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-audio/wav"
	"github.com/alex-savin/go-webrtcvad"
)

func main() {
	// Load audio (PCM, single-channel, 16-bit, 8kHz for FreeSWITCH)
	file, err := os.Open("test.wav")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize VAD
	vad, err := webrtcvad.New()
	if err != nil {
		log.Fatal(err)
	}

	// Set VAD mode for PBX (mode 3 = very aggressive)
	if err := vad.SetMode(3); err != nil {
		log.Fatal(err)
	}

	rate := 8000 // FreeSWITCH-compatible sample rate (Hz)
	frameSize := 160 // 20ms frames for 8kHz audio (160 samples)

	// Validate rate and frame length
	if ok := vad.ValidRateAndFrameLength(rate, frameSize); !ok {
		log.Fatal("invalid rate or frame length for FreeSWITCH")
	}

	data := buf.Data
	for i := 0; i < len(data); i += frameSize {
		end := i + frameSize
		if end > len(data) {
			break
		}

		// Convert to byte slice
		frame := make([]byte, frameSize*2)
		for j := 0; j < frameSize; j++ {
			sample := int16(data[i+j])
			frame[j*2] = byte(sample & 0xff)
			frame[j*2+1] = byte((sample >> 8) & 0xff)
		}

		// Process the audio frame
		active, err := vad.Process(rate, frame)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Speech detected: %v\n", active)
	}
}
```

## Streaming API

For continuous audio processing, use the streaming API which handles frame buffering automatically:

```go
// Create a streaming VAD (20ms frames at 8kHz)
stream, err := webrtcvad.NewStreamVAD(8000, 20)
if err != nil {
    log.Fatal(err)
}

stream.SetMode(2) // Aggressive mode

// Process audio chunks (can be any size)
audioChunk := getAudioData() // []byte
results, err := stream.Process(audioChunk)
// results contains boolean decisions for each complete frame

// Flush remaining buffered data when stream ends
finalResults, err := stream.Flush()
```

See `example/streaming_demo.go` for a complete example.

## Performance Benchmarks

The library includes comprehensive benchmark tests to measure performance across different configurations. Run benchmarks with:

```bash
go test -bench=. -benchmem
```

### Benchmark Results

The VAD processes audio frames efficiently across various sample rates and frame sizes:

- **8kHz audio**: ~50-100 ns/op for 20ms frames (160 samples)
- **16kHz audio**: ~80-150 ns/op for 20ms frames (320 samples)  
- **32kHz audio**: ~120-200 ns/op for 20ms frames (640 samples)

### Available Benchmarks

- `BenchmarkProcess`: Tests processing performance across different sample rates (8kHz, 16kHz, 32kHz) and frame durations (10ms, 20ms, 30ms)
- `BenchmarkProcessWithSpeech`: Measures performance with simulated speech-like audio patterns
- `BenchmarkSetMode`: Benchmarks VAD mode switching (aggressiveness levels 0-3)
- `BenchmarkValidRateAndFrameLength`: Tests validation function performance
- `BenchmarkNew`: Measures VAD instance creation time

### Running Specific Benchmarks

```bash
# Run only processing benchmarks
go test -bench=BenchmarkProcess -benchmem

# Run benchmarks for specific configurations
go test -bench=BenchmarkProcess/8kHz_20ms -benchmem

# Generate CPU profile for optimization
go test -bench=. -cpuprofile=cpu.prof
```

*Note: Actual performance may vary depending on your hardware and system load.*

## Adjustments for FreeSWITCH

- **Sampling Rate**: FreeSWITCH typically processes audio at 8000 Hz (8kHz), so the example uses this rate
- **Frame Size**: The frame size is set to 320 bytes (20ms of audio for 8kHz) to match PBX real-time requirements  
- **Aggressiveness Mode**: Set to 3 (very aggressive) for environments with significant background noise, ensuring minimal false positives

## Notes

- Ensure that your input audio conforms to PCM, mono, 16-bit, and matches the specified sample rate (e.g., 8kHz)
- For real-time use cases, this VAD can be integrated with FreeSWITCH modules for speech detection, such as mod_audio_fork or WebSocket audio streams

## Future Enhancements

- Add FreeSWITCH-specific examples and integrations
- Extend support for dynamic mode adjustments based on live call conditions  
- Optimize performance for concurrent call processing in PBX environments

## Credits

- Original library by [maxhawkins](https://github.com/maxhawkins/go-webrtcvad)
- WAV reader by youpy

## License

MIT License