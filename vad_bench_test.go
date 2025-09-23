package webrtcvad

import (
	"testing"
)

// BenchmarkProcess benchmarks VAD processing for different frame sizes and sample rates
func BenchmarkProcess(b *testing.B) {
	vad, err := New()
	if err != nil {
		b.Fatal(err)
	}

	if err := vad.SetMode(2); err != nil {
		b.Fatal(err)
	}

	// Test different frame sizes and sample rates
	benchmarks := []struct {
		name       string
		sampleRate int
		frameSize  int // in samples
	}{
		{"8kHz_10ms", 8000, 80},
		{"8kHz_20ms", 8000, 160},
		{"8kHz_30ms", 8000, 240},
		{"16kHz_10ms", 16000, 160},
		{"16kHz_20ms", 16000, 320},
		{"16kHz_30ms", 16000, 480},
		{"32kHz_10ms", 32000, 320},
		{"32kHz_20ms", 32000, 640},
		{"32kHz_30ms", 32000, 960},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Create test audio frame (silence)
			frame := make([]byte, bm.frameSize*2) // 16-bit = 2 bytes per sample

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := vad.Process(bm.sampleRate, frame)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkProcessWithSpeech benchmarks processing with simulated speech-like audio
func BenchmarkProcessWithSpeech(b *testing.B) {
	vad, err := New()
	if err != nil {
		b.Fatal(err)
	}

	if err := vad.SetMode(2); err != nil {
		b.Fatal(err)
	}

	// Create a frame with some variation (simulated speech)
	frameSize := 160 // 20ms at 8kHz
	frame := make([]byte, frameSize*2)

	// Fill with some variation instead of silence
	for i := 0; i < frameSize; i++ {
		sample := int16((i % 100) * 100) // Simple pattern
		frame[i*2] = byte(sample & 0xff)
		frame[i*2+1] = byte((sample >> 8) & 0xff)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := vad.Process(8000, frame)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSetMode benchmarks mode changes
func BenchmarkSetMode(b *testing.B) {
	vad, err := New()
	if err != nil {
		b.Fatal(err)
	}

	modes := []int{0, 1, 2, 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mode := modes[i%len(modes)]
		if err := vad.SetMode(mode); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidRateAndFrameLength benchmarks validation function
func BenchmarkValidRateAndFrameLength(b *testing.B) {
	vad, err := New()
	if err != nil {
		b.Fatal(err)
	}

	validCombos := []struct {
		rate   int
		length int
	}{
		{8000, 80}, {8000, 160}, {8000, 240},
		{16000, 160}, {16000, 320}, {16000, 480},
		{32000, 320}, {32000, 640}, {32000, 960},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		combo := validCombos[i%len(validCombos)]
		vad.ValidRateAndFrameLength(combo.rate, combo.length)
	}
}

// BenchmarkNew benchmarks VAD instance creation
func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		vad, err := New()
		if err != nil {
			b.Fatal(err)
		}
		// Note: In a real benchmark, you'd want to clean up the VAD instance
		// but for simplicity and to focus on creation time, we skip cleanup
		_ = vad
	}
}
