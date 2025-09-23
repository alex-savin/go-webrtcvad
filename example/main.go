package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alex-savin/go-webrtcvad"
	"github.com/go-audio/wav"
)

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("usage: example infile.wav")
	}

	filename := flag.Arg(0)

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		log.Fatal("invalid WAV file")
	}

	// Read WAV info
	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		log.Fatal(err)
	}

	format := decoder.Format()
	rate := int(format.SampleRate)
	if format.NumChannels != 1 {
		log.Fatal("expected mono file")
	}
	if rate != 32000 {
		log.Fatal("expected 32kHz file")
	}

	vad, err := webrtcvad.New()
	if err != nil {
		log.Fatal(err)
	}

	if err := vad.SetMode(2); err != nil {
		log.Fatal(err)
	}

	frameSize := 320 * 2 // 20ms at 32kHz = 640 samples = 1280 bytes, but wait
	// For 32kHz, 20ms = 640 samples = 1280 bytes
	// But the original used 320*2 = 640 bytes, which is 320 samples = 10ms at 32kHz
	// Let me check the original: frame := make([]byte, 320*2) for 32kHz, 320 samples * 2 bytes = 640 bytes
	// 320 samples at 32kHz = 10ms

	frameSize = 640 // 320 samples * 2 bytes for 16-bit

	if ok := vad.ValidRateAndFrameLength(rate, frameSize/2); !ok {
		log.Fatal("invalid rate or frame length")
	}

	var isActive bool
	var offset int

	report := func() {
		t := time.Duration(offset) * time.Second / time.Duration(rate) / 2
		fmt.Printf("isActive = %v, t = %v\n", isActive, t)
	}

	data := buf.Data
	for i := 0; i < len(data); i += frameSize / 2 {
		end := i + frameSize/2
		if end > len(data) {
			break
		}

		// Convert int slice to byte slice
		frame := make([]byte, frameSize)
		for j := 0; j < frameSize/2; j++ {
			sample := int16(data[i+j])
			frame[j*2] = byte(sample & 0xff)
			frame[j*2+1] = byte((sample >> 8) & 0xff)
		}

		frameActive, err := vad.Process(rate, frame)
		if err != nil {
			log.Fatal(err)
		}

		if isActive != frameActive || offset == 0 {
			isActive = frameActive
			report()
		}

		offset += frameSize / 2
	}

	report()
}
