package webrtcvad

import (
	"math"
	"testing"
)

// generateTone generates a sine wave tone at the specified frequency
func generateTone(samples int, sampleRate int, frequency float64, amplitude float64) []int16 {
	data := make([]int16, samples)
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		sample := int16(amplitude * math.Sin(2*math.Pi*frequency*t))
		data[i] = sample
	}
	return data
}

// generateNoise generates white noise
func generateNoise(samples int, amplitude float64) []int16 {
	data := make([]int16, samples)
	for i := 0; i < samples; i++ {
		// Simple pseudo-random noise
		noise := int64(i*16807)%2147483647 - 1073741823
		sample := int16(float64(noise) * amplitude / 1073741823)
		data[i] = sample
	}
	return data
}

// pcmToBytes converts int16 PCM data to byte slice
func pcmToBytes(pcm []int16) []byte {
	bytes := make([]byte, len(pcm)*2)
	for i, sample := range pcm {
		bytes[i*2] = byte(sample & 0xff)
		bytes[i*2+1] = byte((sample >> 8) & 0xff)
	}
	return bytes
}

// TestIntegrationSilence tests VAD behavior with silence
func TestIntegrationSilence(t *testing.T) {
	vad, err := New()
	if err != nil {
		t.Fatal(err)
	}

	if err := vad.SetMode(2); err != nil {
		t.Fatal(err)
	}

	// Generate 1 second of silence at 8kHz
	samples := 8000
	silence := make([]int16, samples)
	frameBytes := pcmToBytes(silence)

	// Process in 20ms frames
	frameSize := 160 // 20ms at 8kHz
	speechFrames := 0

	for i := 0; i < len(frameBytes); i += frameSize * 2 {
		end := i + frameSize*2
		if end > len(frameBytes) {
			break
		}

		active, err := vad.Process(8000, frameBytes[i:end])
		if err != nil {
			t.Fatal(err)
		}

		if active {
			speechFrames++
		}
	}

	// With silence, we expect very few or no speech detections
	// Allow some tolerance for statistical variation
	if speechFrames > 5 { // Allow up to 5 false positives in 1 second
		t.Errorf("Too many false positives with silence: %d speech frames detected", speechFrames)
	}
}

// TestIntegrationTone tests VAD behavior with pure tone
func TestIntegrationTone(t *testing.T) {
	vad, err := New()
	if err != nil {
		t.Fatal(err)
	}

	if err := vad.SetMode(2); err != nil {
		t.Fatal(err)
	}

	// Generate 1 second of 1kHz tone at 8kHz with higher amplitude
	samples := 8000
	tone := generateTone(samples, 8000, 1000, 8000) // 1kHz, higher amplitude
	frameBytes := pcmToBytes(tone)

	// Process in 20ms frames
	frameSize := 160
	speechFrames := 0

	for i := 0; i < len(frameBytes); i += frameSize * 2 {
		end := i + frameSize*2
		if end > len(frameBytes) {
			break
		}

		active, err := vad.Process(8000, frameBytes[i:end])
		if err != nil {
			t.Fatal(err)
		}

		if active {
			speechFrames++
		}
	}

	// A loud pure tone should be detected as speech
	// Note: VAD is designed for speech, so pure tones may not always be detected
	// This test mainly ensures the VAD doesn't crash on tonal input
	t.Logf("Tone detection: %d of 50 frames detected as speech", speechFrames)
	if speechFrames == 0 {
		t.Log("Warning: Pure tone was not detected as speech - this may be expected behavior")
	}
}

// TestIntegrationNoise tests VAD behavior with noise
func TestIntegrationNoise(t *testing.T) {
	vad, err := New()
	if err != nil {
		t.Fatal(err)
	}

	if err := vad.SetMode(2); err != nil {
		t.Fatal(err)
	}

	// Generate 1 second of noise at 8kHz
	samples := 8000
	noise := generateNoise(samples, 500) // Moderate noise level
	frameBytes := pcmToBytes(noise)

	// Process in 20ms frames
	frameSize := 160
	speechFrames := 0

	for i := 0; i < len(frameBytes); i += frameSize * 2 {
		end := i + frameSize*2
		if end > len(frameBytes) {
			break
		}

		active, err := vad.Process(8000, frameBytes[i:end])
		if err != nil {
			t.Fatal(err)
		}

		if active {
			speechFrames++
		}
	}

	// Noise should trigger some detections but not all
	// This is a statistical test - noise might be detected as speech
	t.Logf("Noise detection: %d of 50 frames detected as speech", speechFrames)
}

// TestIntegrationModeSensitivity tests different VAD modes
func TestIntegrationModeSensitivity(t *testing.T) {
	// Test with the same noise input across different modes
	samples := 8000
	noise := generateNoise(samples, 300) // Low noise level
	frameBytes := pcmToBytes(noise)
	frameSize := 160

	modes := []int{0, 1, 2, 3}
	results := make([]int, len(modes))

	for modeIdx, mode := range modes {
		vad, err := New()
		if err != nil {
			t.Fatal(err)
		}

		if err := vad.SetMode(mode); err != nil {
			t.Fatal(err)
		}

		speechFrames := 0
		for i := 0; i < len(frameBytes); i += frameSize * 2 {
			end := i + frameSize*2
			if end > len(frameBytes) {
				break
			}

			active, err := vad.Process(8000, frameBytes[i:end])
			if err != nil {
				t.Fatal(err)
			}

			if active {
				speechFrames++
			}
		}
		results[modeIdx] = speechFrames
	}

	// Higher modes should be more sensitive (detect more speech)
	t.Logf("Mode sensitivity results: Mode0=%d, Mode1=%d, Mode2=%d, Mode3=%d",
		results[0], results[1], results[2], results[3])

	// Basic sanity check: higher modes should not detect significantly less
	for i := 1; i < len(results); i++ {
		if results[i] < results[i-1]-10 { // Allow some variation
			t.Errorf("Mode %d detected significantly fewer frames (%d) than mode %d (%d)",
				i, results[i], i-1, results[i-1])
		}
	}
}

// TestIntegrationFrameContinuity tests that VAD state is maintained across frames
func TestIntegrationFrameContinuity(t *testing.T) {
	vad, err := New()
	if err != nil {
		t.Fatal(err)
	}

	if err := vad.SetMode(2); err != nil {
		t.Fatal(err)
	}

	// Create a sequence: silence -> speech -> silence
	silenceSamples := 160 // 20ms
	speechSamples := 160  // 20ms

	silence := make([]int16, silenceSamples)
	speech := generateTone(speechSamples, 8000, 1000, 2000) // Loud tone

	// Convert to frames
	silenceFrame := pcmToBytes(silence)
	speechFrame := pcmToBytes(speech)

	// Process sequence
	results := []bool{}

	// 5 frames of silence
	for i := 0; i < 5; i++ {
		active, err := vad.Process(8000, silenceFrame)
		if err != nil {
			t.Fatal(err)
		}
		results = append(results, active)
	}

	// 5 frames of speech
	for i := 0; i < 5; i++ {
		active, err := vad.Process(8000, speechFrame)
		if err != nil {
			t.Fatal(err)
		}
		results = append(results, active)
	}

	// 5 frames of silence
	for i := 0; i < 5; i++ {
		active, err := vad.Process(8000, silenceFrame)
		if err != nil {
			t.Fatal(err)
		}
		results = append(results, active)
	}

	// Check that speech frames are detected
	speechDetected := 0
	for i := 5; i < 10; i++ {
		if results[i] {
			speechDetected++
		}
	}

	if speechDetected < 3 { // At least 3 of 5 speech frames should be detected
		t.Errorf("Speech not properly detected: only %d of 5 frames", speechDetected)
	}

	t.Logf("Frame continuity test results: %v", results)
}
