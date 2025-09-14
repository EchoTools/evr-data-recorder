package recorder

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// EchoReplayWriterStrategy writes frames to a zip file.
type EchoReplayWriterStrategy struct {
	file     *os.File
	zw       *zip.Writer
	zipEntry io.Writer
	buf      *bytes.Buffer
	filename string
}

func NewEchoReplayWriterStrategy(filePath string) (*EchoReplayWriterStrategy, error) {
	zf, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip file: %w", err)
	}
	zw := zip.NewWriter(zf)
	zw.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestCompression)
	})
	filename := filepath.Base(filePath)
	file, err := zw.Create(filename)
	if err != nil {
		zf.Close()
		return nil, err
	}
	return &EchoReplayWriterStrategy{
		file:     zf,
		zw:       zw,
		zipEntry: file,
		buf:      bytes.NewBuffer(make([]byte, 0, 64*1024)),
		filename: filename,
	}, nil
}

func (z *EchoReplayWriterStrategy) WriteFrame(frame *FrameData) error {
	dataSize := len(frame.SessionData) + len(frame.PlayerBoneData) + 23 + 2 + 1
	z.buf.Grow(dataSize)
	z.buf.WriteString(frame.Timestamp.UTC().Format("2006/01/02 15:04:05.000"))
	z.buf.WriteByte('\t')
	z.buf.Write(frame.SessionData)
	z.buf.WriteByte('\t')
	z.buf.Write(frame.PlayerBoneData)
	z.buf.WriteByte('\n')
	if z.buf.Len() >= zipFileChunkSize {
		if _, err := z.zipEntry.Write(z.buf.Bytes()); err != nil {
			return err
		}
		z.buf.Reset()
	}
	return nil
}

func (z *EchoReplayWriterStrategy) Flush() error {
	if z.buf.Len() > 0 {
		if _, err := z.zipEntry.Write(z.buf.Bytes()); err != nil {
			return err
		}
		z.buf.Reset()
	}
	return nil
}

func (z *EchoReplayWriterStrategy) Close() error {
	var err1, err2, err3 error
	if err := z.Flush(); err != nil {
		err1 = err
	}
	if err := z.zw.Close(); err != nil {
		err2 = err
	}
	if err := z.file.Close(); err != nil {
		err3 = err
	}
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return err3
}

// You can add more WriterStrategy implementations here, e.g., PlainFileWriterStrategy, etc.

func EchoReplaySessionFilename(ts time.Time, sessionID string) string {
	currentTime := ts.UTC().Format("2006-01-02_15-04-05")
	return fmt.Sprintf("rec_%s_%s.echoreplay", currentTime, sessionID)
}
