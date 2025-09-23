package webrtcvad

import (
	"testing"
)

func TestNew(t *testing.T) {
	vad, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if vad == nil {
		t.Fatal("New() returned nil VAD")
	}
}

func TestSetMode(t *testing.T) {
	vad, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Test valid modes
	for mode := 0; mode <= 3; mode++ {
		err := vad.SetMode(mode)
		if err != nil {
			t.Errorf("SetMode(%d) failed: %v", mode, err)
		}
	}

	// Test invalid modes
	invalidModes := []int{-1, 4, 5, 100}
	for _, mode := range invalidModes {
		err := vad.SetMode(mode)
		if err == nil {
			t.Errorf("SetMode(%d) should have failed but didn't", mode)
		}
	}
}

func TestValidRateAndFrameLength(t *testing.T) {
	vad, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Test valid combinations
	validCombos := []struct {
		rate   int
		length int
	}{
		{8000, 80},   // 10ms
		{8000, 160},  // 20ms
		{8000, 240},  // 30ms
		{16000, 160}, // 10ms
		{16000, 320}, // 20ms
		{16000, 480}, // 30ms
		{32000, 320}, // 10ms
		{32000, 640}, // 20ms
		{32000, 960}, // 30ms
	}

	for _, combo := range validCombos {
		if !vad.ValidRateAndFrameLength(combo.rate, combo.length) {
			t.Errorf("ValidRateAndFrameLength(%d, %d) should be valid", combo.rate, combo.length)
		}
	}

	// Test invalid combinations
	invalidCombos := []struct {
		rate   int
		length int
	}{
		{8000, 79},   // too short
		{8000, 241},  // too long
		{8000, 100},  // not matching frame size
		{44100, 441}, // unsupported rate
		{16000, 80},  // wrong length for rate
	}

	for _, combo := range invalidCombos {
		if vad.ValidRateAndFrameLength(combo.rate, combo.length) {
			t.Errorf("ValidRateAndFrameLength(%d, %d) should be invalid", combo.rate, combo.length)
		}
	}
}

func TestNewStreamVAD(t *testing.T) {
	stream, err := NewStreamVAD(8000, 20)
	if err != nil {
		t.Fatalf("NewStreamVAD() failed: %v", err)
	}
	if stream == nil {
		t.Fatal("NewStreamVAD() returned nil")
	}
}

func TestStreamVADSetMode(t *testing.T) {
	stream, err := NewStreamVAD(8000, 20)
	if err != nil {
		t.Fatalf("NewStreamVAD() failed: %v", err)
	}

	for mode := 0; mode <= 3; mode++ {
		err := stream.SetMode(mode)
		if err != nil {
			t.Errorf("SetMode(%d) failed: %v", mode, err)
		}
	}

	err = stream.SetMode(4)
	if err == nil {
		t.Error("SetMode(4) should have failed")
	}
}

func TestStreamVADProcess(t *testing.T) {
	stream, err := NewStreamVAD(8000, 20) // 20ms frames = 160 samples = 320 bytes
	if err != nil {
		t.Fatalf("NewStreamVAD() failed: %v", err)
	}

	err = stream.SetMode(2)
	if err != nil {
		t.Fatal(err)
	}

	// Test with partial frames
	partialFrame := make([]byte, 160) // Half a frame
	results, err := stream.Process(partialFrame)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("Expected no results with partial frame, got %d", len(results))
	}

	// Add the rest of the frame
	remainingFrame := make([]byte, 160)
	results, err = stream.Process(remainingFrame)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result after complete frame, got %d", len(results))
	}

	// Test with multiple frames
	multipleFrames := make([]byte, 960) // 3 frames
	results, err = stream.Process(multipleFrames)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

func TestStreamVADFlush(t *testing.T) {
	stream, err := NewStreamVAD(8000, 20)
	if err != nil {
		t.Fatalf("NewStreamVAD() failed: %v", err)
	}

	// Add partial frame
	partialFrame := make([]byte, 100)
	results, err := stream.Process(partialFrame)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Error("Should not have results for partial frame")
	}

	// Flush should process the partial frame (padded with silence)
	results, err = stream.Flush()
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result from flush, got %d", len(results))
	}

	// Buffer should be empty after flush
	results, err = stream.Flush()
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Error("Buffer should be empty after flush")
	}
}

func TestStreamVADReset(t *testing.T) {
	stream, err := NewStreamVAD(8000, 20)
	if err != nil {
		t.Fatalf("NewStreamVAD() failed: %v", err)
	}

	// Add some data
	partialFrame := make([]byte, 100)
	stream.Process(partialFrame)

	// Reset should clear buffer
	stream.Reset()

	// Flush should not return anything
	results, err := stream.Flush()
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Error("Reset should clear buffer")
	}
}

func TestStreamVADInvalidParams(t *testing.T) {
	// Invalid sample rate
	_, err := NewStreamVAD(44100, 20)
	if err == nil {
		t.Error("Should fail with invalid sample rate")
	}

	// Invalid frame duration
	_, err = NewStreamVAD(8000, 15)
	if err == nil {
		t.Error("Should fail with invalid frame duration")
	}

	// Valid parameters should work
	stream, err := NewStreamVAD(8000, 20)
	if err != nil {
		t.Fatal(err)
	}
	if stream == nil {
		t.Error("Should succeed with valid parameters")
	}
}
