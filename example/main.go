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

	vad, err := webrtcvad.New()
	if err != nil {
		log.Fatal(err)
	}

	if err := vad.SetMode(2); err != nil {
		log.Fatal(err)
	}

	// Use 10ms frames: frameSamples samples per frame, 2 bytes per 16-bit sample.
	frameSamples := rate / 100
	frameSize := frameSamples * 2

	if ok := vad.ValidRateAndFrameLength(rate, frameSamples); !ok {
		log.Fatal("invalid rate or frame length")
	}

	var isActive bool
	var offset int // in samples

	report := func() {
		t := time.Duration(offset) * time.Second / time.Duration(rate)
		fmt.Printf("isActive = %v, t = %v\n", isActive, t)
	}

	data := buf.Data
	for i := 0; i+frameSamples <= len(data); i += frameSamples {
		// Convert the int samples to a 16-bit little-endian byte frame.
		frame := make([]byte, frameSize)
		for j := 0; j < frameSamples; j++ {
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

		offset += frameSamples
	}

	report()
}
