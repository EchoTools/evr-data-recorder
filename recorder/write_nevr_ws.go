package recorder

import (
	"os"

	"github.com/golang/protobuf/proto"
)

func write(frames *[]*GameAPISession) {
	t.Logf("Encoding %d frames to protobuf", len(*frames))
	for _, frame := range *frames {
		// Encode the GameAPISession to protobuf
		protbufEncodedData, err := proto.Marshal(frame)
		if err != nil {
			t.Fatalf("failed to marshal GameAPISession to protobuf: %v", err)
		}
		t.Logf("Encoded GameAPISession to protobuf, size: %d bytes", len(protbufEncodedData))
		// Write the protobuf encoded data to the output file
		if _, err := outputFile.Write(protbufEncodedData); err != nil {
			t.Fatalf("failed to write GameAPISession_protobuf.bin: %v", err)
		}
	}
	// Close the output file
	if err := outputFile.Close(); err != nil {
		t.Fatalf("failed to close output file GameAPISession_protobuf.bin: %v", err)
	}
	// Read the protobuf encoded data back from the file
	protbufEncodedData, err := os.ReadFile("GameAPISession_protobuf.bin")
	if err != nil {
		t.Fatalf("failed to read GameAPISession_protobuf.bin: %v", err)
	}
}
