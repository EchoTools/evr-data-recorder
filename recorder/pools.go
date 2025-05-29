package recorder

import (
	"bytes"
	"strings"
)

var stringBuilderPool = NewPoolOf(
	func() *strings.Builder {
		return &strings.Builder{}
	},
	func(sb *strings.Builder) {
		sb.Reset() // Reset the builder for reuse
	},
)

var bytesBufferPool = NewPoolOf(func() *bytes.Buffer {
	return bytes.NewBuffer(make([]byte, 0, 64*1024)) // 64KB buffer
},
	func(b *bytes.Buffer) {
		b.Reset() // Reset the buffer for reuse
	},
)
