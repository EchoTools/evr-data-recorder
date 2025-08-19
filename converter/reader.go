package converter

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/echotools/evr-data-recorder/v3/recorder"
)

// EchoReplayReader reads .echoreplay files
type EchoReplayReader struct {
	zipReader *zip.ReadCloser
	scanner   *bufio.Scanner
}

// NewEchoReplayReader creates a new reader for .echoreplay files
func NewEchoReplayReader(filename string) (*EchoReplayReader, error) {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file %s: %w", filename, err)
	}

	// Find the data file inside the zip (should have the same name as the zip file)
	var dataFile *zip.File
	for _, file := range zr.File {
		if !file.FileInfo().IsDir() {
			dataFile = file
			break
		}
	}

	if dataFile == nil {
		zr.Close()
		return nil, fmt.Errorf("no data file found in zip archive %s", filename)
	}

	reader, err := dataFile.Open()
	if err != nil {
		zr.Close()
		return nil, fmt.Errorf("failed to open data file in zip: %w", err)
	}

	scanner := bufio.NewScanner(reader)
	return &EchoReplayReader{
		zipReader: zr,
		scanner:   scanner,
	}, nil
}

// ReadFrame reads the next frame from the .echoreplay file
func (r *EchoReplayReader) ReadFrame() (*recorder.FrameData, error) {
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return nil, fmt.Errorf("scanner error: %w", err)
		}
		return nil, io.EOF
	}

	line := r.scanner.Text()
	parts := strings.Split(line, "\t")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid line format, expected 3 parts but got %d: %s", len(parts), line)
	}

	// Parse timestamp
	timestamp, err := time.Parse("2006/01/02 15:04:05.000", parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp %s: %w", parts[0], err)
	}

	return &recorder.FrameData{
		Timestamp:      timestamp,
		SessionData:    []byte(parts[1]),
		PlayerBoneData: []byte(parts[2]),
	}, nil
}

// Close closes the reader
func (r *EchoReplayReader) Close() error {
	if r.zipReader != nil {
		return r.zipReader.Close()
	}
	return nil
}

// ReadAllFrames reads all frames from the file
func (r *EchoReplayReader) ReadAllFrames() ([]*recorder.FrameData, error) {
	var frames []*recorder.FrameData
	for {
		frame, err := r.ReadFrame()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}