package recorder

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/echotools/nevr-common/gameapi"
	"github.com/echotools/nevr-common/v3/gameapi"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"
)

// NEVRReplayWriterStrategy writes frames to a Zstd-compressed file.
type NEVRReplayWriterStrategy struct {
	file     *os.File
	encoder  *zstd.Encoder
	buf      *bytes.Buffer
	filename string
}

func NewNEVRReplayWriterStrategy(ts time.Time, sessionID string) (*NEVRReplayWriterStrategy, error) {
	currentTime := ts.UTC().Format("2006-01-02_15-04-05")
	filePath := fmt.Sprintf("rec_%s_%s.echoreplay.zst", currentTime, sessionID)

	zf, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd file: %w", err)
	}
	encoder, err := zstd.NewWriter(zf, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		zf.Close()
		return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
	}
	filename := filepath.Base(filePath)
	return &NEVRReplayWriterStrategy{
		file:     zf,
		encoder:  encoder,
		buf:      bytes.NewBuffer(make([]byte, 0, 64*1024)),
		filename: filename,
	}, nil
}

func (z *NEVRReplayWriterStrategy) WriteFrame(frame *FrameData) error {
	sessionResponse := gameapi.SessionResponse{}
	if err := proto.Unmarshal(frame.SessionData, &sessionResponse); err != nil {
		return fmt.Errorf("failed to unmarshal session data: %w", err)
	}
	playerBoneData := gameapi.PlayerBoneData{}
	if err := proto.Unmarshal(frame.PlayerBoneData, &playerBoneData); err != nil {
		return fmt.Errorf("failed to unmarshal player bone data: %w", err)
	}

	dataSize := len(frame.SessionData) + len(frame.PlayerBoneData) + 23 + 2 + 1
	z.buf.Grow(dataSize)
	z.buf.WriteString(frame.Timestamp.UTC().Format("2006/01/02 15:04:05.000"))
	z.buf.WriteByte('\t')
	z.buf.Write(frame.SessionData)
	z.buf.WriteByte('\t')
	z.buf.Write(frame.PlayerBoneData)
	z.buf.WriteByte('\n')
	if z.buf.Len() >= zipFileChunkSize {
		if _, err := z.encoder.Write(z.buf.Bytes()); err != nil {
			return err
		}
		z.buf.Reset()
	}
	return nil
}

func (z *NEVRReplayWriterStrategy) Flush() error {
	if z.buf.Len() > 0 {
		if _, err := z.encoder.Write(z.buf.Bytes()); err != nil {
			return err
		}
		z.buf.Reset()
	}
	return nil
}

func (z *NEVRReplayWriterStrategy) Close() error {
	var err1, err2, err3 error
	if err := z.Flush(); err != nil {
		err1 = err
	}
	if err := z.encoder.Close(); err != nil {
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
