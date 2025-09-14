package recorder

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

const (
	zipFileChunkSize = 2 * 1024 * 1024 // 2MB chunk size for zip file writing
)

var ErrSessionUUIDChanged = errors.New("session UUID changed")

// WriterStrategy defines the interface for writing frame data.
type WriterStrategy interface {
	WriteFrame(frame *FrameData) error
	Flush() error
	Close() error
}

// FrameDataLogSession manages the session and delegates writing to a WriterStrategy.
type FrameDataLogSession struct {
	sync.Mutex
	ctx         context.Context
	ctxCancelFn context.CancelFunc
	logger      *zap.Logger

	filePath   string
	outgoingCh chan *FrameData
	sessionID  string
	stopped    bool

	writer WriterStrategy
}

func (e *FrameDataLogSession) Context() context.Context {
	return e.ctx
}

func NewFrameDataLogSession(ctx context.Context, logger *zap.Logger, filePath string, sessionID string, writer WriterStrategy) *FrameDataLogSession {
	ctx, cancel := context.WithCancel(ctx)
	return &FrameDataLogSession{
		ctx:         ctx,
		ctxCancelFn: cancel,
		logger:      logger,
		filePath:    filePath,
		outgoingCh:  make(chan *FrameData, 1000),
		sessionID:   sessionID,
		writer:      writer,
	}
}

func (fw *FrameDataLogSession) ProcessFrames() error {
	byteCount := 0

OuterLoop:
	for {
		select {
		case frame := <-fw.outgoingCh:
			fw.Lock()
			if fw.stopped {
				fw.Unlock()
				break OuterLoop
			}
			sessionID, err := fw.extractSessionUUID(frame.SessionData)
			if err != nil {
				fw.logger.Error("Failed to extract session UUID from frame",
					zap.String("data", string(frame.SessionData)),
					zap.Error(err))
				fw.Unlock()
				break OuterLoop
			}
			if sessionID != fw.sessionID {
				fw.logger.Debug("Session UUID changed, stopping frame processing",
					zap.String("old_session_id", fw.sessionID),
					zap.String("new_session_id", sessionID),
				)
				fw.Unlock()
				break OuterLoop
			}
			if err := fw.writer.WriteFrame(frame); err != nil {
				fw.logger.Error("Failed to write frame", zap.Error(err))
				fw.Unlock()
				break OuterLoop
			}
			byteCount += len(frame.SessionData) + len(frame.PlayerBoneData)
			fw.Unlock()
		case <-fw.ctx.Done():
			break OuterLoop
		}
	}

	fw.Close()

	fw.Lock()
	defer fw.Unlock()
	if err := fw.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %v", err)
	}
	if err := fw.writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}

	fw.logger.Info("Replay file written",
		zap.String("file_path", fw.filePath),
		zap.Int("byte_count", byteCount),
	)
	return nil
}

func (fw *FrameDataLogSession) WriteFrame(frame *FrameData) error {
	if fw.IsStopped() {
		return fmt.Errorf("frame writer is stopped")
	}
	select {
	case fw.outgoingCh <- frame:
		return nil
	case <-fw.ctx.Done():
		return fmt.Errorf("context cancelled, cannot write frame: %w", fw.ctx.Err())
	default:
		return fmt.Errorf("outgoing channel is full, cannot write frame")
	}
}

func (fw *FrameDataLogSession) Close() {
	fw.ctxCancelFn()
	fw.Lock()
	if fw.stopped {
		fw.Unlock()
		return
	}
	fw.stopped = true
	fw.Unlock()
}

func (fw *FrameDataLogSession) IsStopped() bool {
	fw.Lock()
	defer fw.Unlock()
	return fw.stopped
}

func (fw *FrameDataLogSession) extractSessionUUID(sessionData []byte) (string, error) {
	response := SessionMeta{}
	if err := json.Unmarshal(sessionData, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return response.SessionUUID, nil
}
