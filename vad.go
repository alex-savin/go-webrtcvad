// Package webrtcvad provides a Go wrapper for the WebRTC Voice Activity Detector (VAD).
// The VAD analyzes audio frames and determines whether they contain speech or not.
// It supports sample rates of 8000, 16000, and 32000 Hz with frame lengths of 10, 20, or 30 ms.
package webrtcvad

//#cgo CFLAGS: -I.
//#include "webrtc/common_audio/vad/include/webrtc_vad.h"
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

// New creates a new VAD instance.
// It initializes the VAD with default settings and returns an error if creation fails.
func New() (*VAD, error) {
	var inst *C.struct_WebRtcVadInst

	ret := C.WebRtcVad_Create(&inst)
	if ret != 0 {
		return nil, errors.New("failed to create VAD")
	}

	vad := &VAD{inst}
	// Free the C instance once the VAD becomes unreachable. The cleanup
	// captures inst rather than vad so it does not keep vad alive itself.
	runtime.AddCleanup(vad, func(inst *C.struct_WebRtcVadInst) {
		C.WebRtcVad_Free(inst)
	}, inst)

	ret = C.WebRtcVad_Init(inst)
	if ret != 0 {
		return nil, errors.New("default mode could not be set")
	}

	return vad, nil
}

// VAD represents a WebRTC Voice Activity Detector instance.
// It is not safe for concurrent use by multiple goroutines.
type VAD struct {
	inst *C.struct_WebRtcVadInst
}

// SetMode sets the VAD operating mode. The mode controls the aggressiveness of the VAD:
//   - 0: Quality mode (least aggressive)
//   - 1: Low bitrate mode
//   - 2: Aggressive mode
//   - 3: Very aggressive mode (most restrictive)
//
// Higher modes reduce false positives but may miss some speech.
// Returns an error if the mode is invalid or the VAD is not initialized.
func (v *VAD) SetMode(mode int) error {
	if mode < 0 || mode > 3 {
		return errors.New("mode must be between 0 and 3")
	}
	ret := C.WebRtcVad_set_mode(v.inst, C.int(mode))
	runtime.KeepAlive(v)
	if ret != 0 {
		return errors.New("mode could not be set")
	}
	return nil
}

// Reset reinitializes the VAD, clearing its internal state. This also resets
// the aggressiveness mode to the default (0); call SetMode again afterwards to
// restore a non-default mode.
func (v *VAD) Reset() error {
	ret := C.WebRtcVad_Init(v.inst)
	runtime.KeepAlive(v)
	if ret != 0 {
		return errors.New("VAD could not be reset")
	}
	return nil
}

// Process analyzes an audio frame and returns true if speech is detected.
// The audio frame must be 16-bit little-endian signed PCM data.
// Supported sample rates: 8000, 16000, 32000 Hz
// Supported frame lengths: 10ms, 20ms, 30ms (corresponding to 80/160/240, 160/320/480, 240/480/720 samples)
// Returns an error if the frame format is invalid or processing fails.
func (v *VAD) Process(fs int, audioFrame []byte) (activeVoice bool, err error) {
	if len(audioFrame)%2 != 0 {
		return false, errors.New("audio frame must be 16-bit little-endian signed PCM (even number of bytes)")
	}
	frameLen := len(audioFrame) / 2
	if !v.ValidRateAndFrameLength(fs, frameLen) {
		return false, errors.New("invalid sample rate or frame length")
	}

	audioFramePtr := (*C.int16_t)(unsafe.Pointer(&audioFrame[0]))

	ret := C.WebRtcVad_Process(v.inst, C.int(fs), audioFramePtr, C.int(frameLen))
	runtime.KeepAlive(v)
	switch ret {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, errors.New("processing error")
	}
}

// ValidRateAndFrameLength checks if the given sample rate and frame length combination is supported.
// Supported sample rates: 8000, 16000, 32000 Hz
// Supported frame lengths correspond to 10ms, 20ms, or 30ms at the given rate.
func (v *VAD) ValidRateAndFrameLength(rate int, frameLength int) bool {
	ret := C.WebRtcVad_ValidRateAndFrameLength(C.int(rate), C.int(frameLength))
	return ret >= 0
}

// StreamVAD provides a streaming interface for continuous audio processing.
// It automatically handles frame buffering and maintains VAD state across calls.
type StreamVAD struct {
	vad        *VAD
	sampleRate int
	frameSize  int // in samples
	mode       int
	buffer     []byte
}

// NewStreamVAD creates a new streaming VAD instance.
// sampleRate should be 8000, 16000, or 32000 Hz.
// frameDuration should be 10, 20, or 30 (milliseconds).
func NewStreamVAD(sampleRate, frameDuration int) (*StreamVAD, error) {
	vad, err := New()
	if err != nil {
		return nil, err
	}

	// Calculate frame size in samples
	frameSize := (sampleRate * frameDuration) / 1000
	if !vad.ValidRateAndFrameLength(sampleRate, frameSize) {
		return nil, errors.New("invalid sample rate or frame duration")
	}

	return &StreamVAD{
		vad:        vad,
		sampleRate: sampleRate,
		frameSize:  frameSize,
		buffer:     make([]byte, 0, frameSize*2), // Pre-allocate with some capacity
	}, nil
}

// SetMode sets the VAD aggressiveness mode (0-3).
func (s *StreamVAD) SetMode(mode int) error {
	if err := s.vad.SetMode(mode); err != nil {
		return err
	}
	s.mode = mode
	return nil
}

// Process adds audio data to the stream and returns VAD decisions for complete frames.
// The audioData should be 16-bit little-endian signed PCM.
// Returns a slice of boolean values, one for each complete frame processed.
// May return an empty slice if not enough data was provided to complete a frame.
func (s *StreamVAD) Process(audioData []byte) ([]bool, error) {
	if len(audioData)%2 != 0 {
		return nil, errors.New("audio data must be 16-bit (even number of bytes)")
	}

	// Add new data to buffer
	s.buffer = append(s.buffer, audioData...)

	results := []bool{}
	bytesPerFrame := s.frameSize * 2

	// Process complete frames
	for len(s.buffer) >= bytesPerFrame {
		frame := s.buffer[:bytesPerFrame]
		s.buffer = s.buffer[bytesPerFrame:]

		active, err := s.vad.Process(s.sampleRate, frame)
		if err != nil {
			return nil, err
		}

		results = append(results, active)
	}

	return results, nil
}

// Flush processes any remaining buffered data and returns final VAD decisions.
// Call this when the audio stream ends to ensure all buffered data is processed.
func (s *StreamVAD) Flush() ([]bool, error) {
	if len(s.buffer) == 0 {
		return []bool{}, nil
	}

	// If we have partial frame data, we need to pad with silence
	bytesPerFrame := s.frameSize * 2
	if len(s.buffer) < bytesPerFrame {
		// Pad with silence
		padding := make([]byte, bytesPerFrame-len(s.buffer))
		s.buffer = append(s.buffer, padding...)
	}

	active, err := s.vad.Process(s.sampleRate, s.buffer)
	if err != nil {
		return nil, err
	}

	// Clear buffer
	s.buffer = s.buffer[:0]

	return []bool{active}, nil
}

// Reset clears the internal buffer and resets the VAD's internal state,
// reapplying the configured aggressiveness mode. Call this to reuse the same
// StreamVAD for an independent audio stream.
func (s *StreamVAD) Reset() error {
	s.buffer = s.buffer[:0]
	if err := s.vad.Reset(); err != nil {
		return err
	}
	// Reset reverts the VAD to the default mode, so reapply the configured one.
	return s.vad.SetMode(s.mode)
}
