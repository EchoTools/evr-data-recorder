package recorder

import (
	"context"
	"time"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type FrameData struct {
	Timestamp      time.Time
	SessionData    []byte
	PlayerBoneData []byte
}

func (f FrameData) SessionUUID() string {
	// Assuming SessionData contains a field "session_uuid" in JSON format
	var meta SessionMeta
	if err := json.Unmarshal(f.SessionData, &meta); err != nil {
		return ""
	}
	if meta.SessionUUID != "" {
		return meta.SessionUUID
	}
	return ""
}

type FrameWriter interface {
	Context() context.Context
	WriteFrame(*FrameData) error
	Close()
	IsStopped() bool
}

type FrameReader interface {
	Context() context.Context
	ReadFrame() (*FrameData, error)
	Close()
}
