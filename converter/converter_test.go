package converter

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/echotools/evr-data-recorder/v3/recorder"
)

func TestConversion(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "echoreplay_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock .echoreplay file using the writer from the existing codebase
	testFile := filepath.Join(tmpDir, "test.echoreplay")

	// Create test frame data
	testFrames := []*recorder.FrameData{
		{
			Timestamp:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			SessionData:    []byte(`{"sessionid":"test-session-123","game_status":"active"}`),
			PlayerBoneData: []byte(`{"player1":{"x":1.0,"y":2.0,"z":3.0}}`),
		},
		{
			Timestamp:      time.Date(2024, 1, 1, 12, 0, 1, 0, time.UTC),
			SessionData:    []byte(`{"sessionid":"test-session-123","game_status":"active"}`),
			PlayerBoneData: []byte(`{"player1":{"x":1.1,"y":2.1,"z":3.1}}`),
		},
	}

	// Create a mock echoreplay file using the existing writer
	err = createMockEchoReplayFile(testFile, testFrames)
	if err != nil {
		t.Fatalf("Failed to create mock echoreplay file: %v", err)
	}

	// Test reading the file
	reader, err := NewEchoReplayReader(testFile)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	frames, err := reader.ReadAllFrames()
	if err != nil {
		t.Fatalf("Failed to read frames: %v", err)
	}

	if len(frames) != len(testFrames) {
		t.Fatalf("Expected %d frames, got %d", len(testFrames), len(frames))
	}

	// Test conversion
	outputFile := filepath.Join(tmpDir, "test.nevrcap")
	writer := NewNEVRWriter(outputFile, testFile)

	for _, frame := range frames {
		if err := writer.WriteFrame(frame); err != nil {
			t.Fatalf("Failed to write frame: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Verify the output file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Output file does not exist: %s", outputFile)
	}

	// Test the full conversion function
	options := ConvertOptions{
		RemoveOriginal: false,
		DryRun:         false,
		Verbose:        true,
	}

	err = ConvertFile(testFile, options)
	if err != nil {
		t.Fatalf("Failed to convert file: %v", err)
	}

	// Check that the converted file was created
	expectedFile := filepath.Join(tmpDir, "test.nevrcap")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("Converted file does not exist: %s", expectedFile)
	}
}

// createMockEchoReplayFile creates a mock .echoreplay file for testing
func createMockEchoReplayFile(filename string, frames []*recorder.FrameData) error {
	// We need to create a zip file with the expected format
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Create a file inside the zip with the same name
	baseFilename := filepath.Base(filename)
	zipFile, err := zipWriter.Create(baseFilename)
	if err != nil {
		return err
	}

	// Write frame data in the expected format
	for _, frame := range frames {
		line := frame.Timestamp.UTC().Format("2006/01/02 15:04:05.000") + "\t" +
			string(frame.SessionData) + "\t" +
			string(frame.PlayerBoneData) + "\n"
		_, err := zipFile.Write([]byte(line))
		if err != nil {
			return err
		}
	}

	return nil
}