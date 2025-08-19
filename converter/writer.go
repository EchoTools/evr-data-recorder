package converter

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/echotools/evr-data-recorder/v3/recorder"
)

// NEVRFrame represents a frame in the NEVR capture format
type NEVRFrame struct {
	Timestamp      time.Time       `json:"timestamp"`
	SessionData    json.RawMessage `json:"session_data"`
	PlayerBoneData json.RawMessage `json:"player_bone_data"`
}

// NEVRCapture represents the complete NEVR capture file format
type NEVRCapture struct {
	Version    string      `json:"version"`
	CreatedAt  time.Time   `json:"created_at"`
	SourceFile string      `json:"source_file"`
	Frames     []NEVRFrame `json:"frames"`
}

// NEVRWriter writes .nevrcap files
type NEVRWriter struct {
	filename   string
	sourceFile string
	frames     []NEVRFrame
}

// NewNEVRWriter creates a new writer for .nevrcap files
func NewNEVRWriter(filename, sourceFile string) *NEVRWriter {
	return &NEVRWriter{
		filename:   filename,
		sourceFile: sourceFile,
		frames:     make([]NEVRFrame, 0),
	}
}

// WriteFrame adds a frame to the capture
func (w *NEVRWriter) WriteFrame(frame *recorder.FrameData) error {
	// Try to parse session data as JSON, fallback to raw bytes
	var sessionData json.RawMessage
	if json.Valid(frame.SessionData) {
		sessionData = json.RawMessage(frame.SessionData)
	} else {
		// If not valid JSON, encode as string
		sessionJSON, err := json.Marshal(string(frame.SessionData))
		if err != nil {
			return fmt.Errorf("failed to encode session data: %w", err)
		}
		sessionData = json.RawMessage(sessionJSON)
	}

	// Try to parse bone data as JSON, fallback to raw bytes
	var boneData json.RawMessage
	if json.Valid(frame.PlayerBoneData) {
		boneData = json.RawMessage(frame.PlayerBoneData)
	} else {
		// If not valid JSON, encode as string
		boneJSON, err := json.Marshal(string(frame.PlayerBoneData))
		if err != nil {
			return fmt.Errorf("failed to encode bone data: %w", err)
		}
		boneData = json.RawMessage(boneJSON)
	}

	nevrFrame := NEVRFrame{
		Timestamp:      frame.Timestamp,
		SessionData:    sessionData,
		PlayerBoneData: boneData,
	}

	w.frames = append(w.frames, nevrFrame)
	return nil
}

// Close writes all frames to the file
func (w *NEVRWriter) Close() error {
	capture := NEVRCapture{
		Version:    "1.0",
		CreatedAt:  time.Now().UTC(),
		SourceFile: w.sourceFile,
		Frames:     w.frames,
	}

	file, err := os.Create(w.filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", w.filename, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print for readability
	if err := encoder.Encode(capture); err != nil {
		return fmt.Errorf("failed to encode NEVR capture: %w", err)
	}

	return nil
}

// FrameCount returns the number of frames written
func (w *NEVRWriter) FrameCount() int {
	return len(w.frames)
}