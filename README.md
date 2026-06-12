# Go WebRTC VAD

[![Go Reference](https://pkg.go.dev/badge/github.com/alex-savin/go-webrtcvad.svg)](https://pkg.go.dev/github.com/alex-savin/go-webrtcvad)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Go (cgo) wrapper around Google's WebRTC Voice Activity Detector (VAD) — the same C sources used by [py-webrtcvad](https://github.com/wiseman/py-webrtcvad).

A VAD classifies a piece of audio data as being voiced or unvoiced. It can be useful for telephony and speech recognition.

## Overview

A VAD classifies a piece of audio data as being voiced or unvoiced. It is highly beneficial for telephony systems (like FreeSWITCH) and speech recognition pipelines, particularly when dealing with real-time audio streams.

The VAD originally developed for the WebRTC project by Google is one of the best available, being:

- **Fast** - Optimized for real-time processing
- **Modern** - Based on current state-of-the-art algorithms  
- **Free** - Open source with MIT license

## Requirements

- **Go 1.26 or newer**
- **cgo enabled with a working C compiler** (e.g. `gcc` or `clang`). The WebRTC
  VAD sources are vendored and compiled with the package, so you do **not** need
  a system WebRTC installation — but cgo must be available (`CGO_ENABLED=1`,
  which is the default for native builds).

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

// Reset clears the buffer and the VAD's internal state so the same
// StreamVAD can be reused for an independent stream. The configured
// aggressiveness mode is preserved.
if err := stream.Reset(); err != nil {
    log.Fatal(err)
}
```

See `example/streaming/main.go` for a complete, runnable example (`go run ./example/streaming`).

## API Reference

### Supported audio formats

Input must be mono, 16-bit little-endian signed PCM. The following sample rate
and frame duration combinations are supported:

| Sample rate | 10 ms | 20 ms | 30 ms |
|-------------|-------|-------|-------|
| 8000 Hz     | 80    | 160   | 240   |
| 16000 Hz    | 160   | 320   | 480   |
| 32000 Hz    | 320   | 640   | 960   |

(values are samples per frame; multiply by 2 for bytes).

### `VAD` — frame-based API

- `New() (*VAD, error)` — create a VAD instance.
- `(*VAD) SetMode(mode int) error` — set aggressiveness (0 = quality, 3 = very aggressive).
- `(*VAD) Process(rate int, frame []byte) (bool, error)` — classify one frame; returns `true` for speech.
- `(*VAD) Reset() error` — reinitialize internal state (reverts to the default mode).
- `(*VAD) ValidRateAndFrameLength(rate, frameLength int) bool` — validate a rate/frame-length pair.

### `StreamVAD` — streaming API

- `NewStreamVAD(sampleRate, frameDuration int) (*StreamVAD, error)` — `frameDuration` in ms (10/20/30).
- `(*StreamVAD) SetMode(mode int) error`
- `(*StreamVAD) Process(audioData []byte) ([]bool, error)` — buffers input, returns one decision per complete frame.
- `(*StreamVAD) Flush() ([]bool, error)` — process any remaining buffered data (padded with silence).
- `(*StreamVAD) Reset() error` — clear the buffer and reset state, preserving the configured mode.

`VAD` and `StreamVAD` are **not safe for concurrent use** by multiple goroutines.

## Performance Benchmarks

The library includes comprehensive benchmark tests to measure performance across different configurations. Run benchmarks with:

```bash
go test -bench=. -benchmem
```

### Benchmark Results

Processing a single frame is allocation-free. Measured on an Apple M4 Max,
reported as the median of 10 runs (`go test -bench=. -benchmem -benchtime=2s
-count=10`, summarized with [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)):

| Sample rate | 10 ms | 20 ms | 30 ms |
|-------------|-------|-------|-------|
| 8000 Hz     | ~210 ns/op  | ~435 ns/op  | ~688 ns/op  |
| 16000 Hz    | ~442 ns/op  | ~896 ns/op  | ~1426 ns/op |
| 32000 Hz    | ~889 ns/op  | ~1896 ns/op | ~2968 ns/op |

Run-to-run variation was ±1–6%. All `Process` benchmarks report **0 B/op,
0 allocs/op**. Cost scales with the number of samples per frame, so processing
time roughly doubles with each step up in sample rate or frame duration.

### Real-time performance

What matters in practice is how the per-frame cost compares to the amount of
audio in each frame. Even the heaviest configuration processes audio thousands
of times faster than real time:

| Config        | Time / frame | Audio / frame | Faster than real time |
|---------------|--------------|---------------|-----------------------|
| 8kHz / 20ms   | ~435 ns      | 20 ms         | ~46,000×              |
| 32kHz / 30ms  | ~2968 ns     | 30 ms         | ~10,100×              |

In other words, classifying a frame takes a tiny fraction of the time that frame
represents. Combined with zero per-frame allocations, a single core can sustain
thousands of concurrent audio streams before VAD becomes a bottleneck — making
this suitable for real-time telephony and streaming workloads.

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