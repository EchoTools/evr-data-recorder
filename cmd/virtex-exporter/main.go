package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/echotools/nevr-common/v4/gen/go/apigame"
	"github.com/echotools/nevr-common/v4/gen/go/rtapi"
	"github.com/echotools/nevrcap/pkg/codecs"
	"github.com/echotools/nevrcap/pkg/processing"
)

var version string = "v1.0.0"

// VirtexBone represents a single bone in the Virtex format
type VirtexBone struct {
	Rotation    VirtexVector4 `json:"Rotation"`
	Translation VirtexVector4 `json:"Translation"`
	Scale3D     VirtexVector4 `json:"Scale3D"`
	Parameters  [4]float32    `json:"Parameters"`
}

// VirtexVector4 represents a 4D vector split into XY and ZW
type VirtexVector4 struct {
	XY [2]float32 `json:"XY"`
	ZW [2]float32 `json:"ZW"`
}

// VirtexBonesData represents the bones section
type VirtexBonesData struct {
	UserBones []VirtexBone `json:"User_Bones"`
}

// VirtexResponse represents the complete response format
type VirtexResponse struct {
	Data struct {
		Session        *apigame.SessionResponse `json:"Session"`
		Bones          VirtexBonesData          `json:"Bones"`
		StreamTimecode string                   `json:"StreamTimecode"`
		StreamLink     string                   `json:"StreamLink"`
	} `json:"Data"`
}

type VirtexServer struct {
	mode string // "live" or "replay"

	// Live mode fields
	baseURL    string
	httpClient *http.Client

	// Replay mode fields
	replayFile string
	replayLoop bool

	// Shared state
	mu           sync.RWMutex
	currentFrame *rtapi.LobbySessionStateFrame
	isPlaying    bool
	streamLink   string
	bindAddr     string
}

func main() {
	var (
		mode       = flag.String("mode", "live", "Mode: 'live' or 'replay'")
		source     = flag.String("source", "", "Source: host:port for live mode, or file path for replay mode")
		bindAddr   = flag.String("bind", "127.0.0.1:8080", "Host:port to bind HTTP server to")
		loop       = flag.Bool("loop", false, "Loop replay continuously (replay mode only)")
		streamLink = flag.String("stream-link", "", "Stream link (e.g., Twitch URL)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nVirtex Stream Server - Streams EchoVR session data in Virtex format\n")
		fmt.Fprintf(os.Stderr, "\nVersion: %s\n", version)
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  Live mode:   %s -mode live -source 192.168.1.100:6721\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Replay mode: %s -mode replay -source recording.echoreplay -loop\n", os.Args[0])
	}

	flag.Parse()

	if *source == "" {
		flag.Usage()
		os.Exit(1)
	}

	server := &VirtexServer{
		mode:       *mode,
		bindAddr:   *bindAddr,
		replayLoop: *loop,
		streamLink: *streamLink,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	switch *mode {
	case "live":
		server.baseURL = "http://" + *source
		server.httpClient = &http.Client{
			Timeout: 3 * time.Second,
			Transport: &http.Transport{
				MaxConnsPerHost:       2,
				DisableCompression:    true,
				MaxIdleConns:          2,
				MaxIdleConnsPerHost:   2,
				IdleConnTimeout:       5 * time.Second,
				TLSHandshakeTimeout:   2 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   2 * time.Second,
					KeepAlive: 5 * time.Second,
				}).DialContext,
			},
		}
		go server.runLiveMode(ctx)

	case "replay":
		server.replayFile = *source
		if _, err := os.Stat(*source); os.IsNotExist(err) {
			log.Fatalf("Replay file does not exist: %s", *source)
		}
		go server.runReplayMode(ctx)

	default:
		log.Fatalf("Invalid mode: %s (must be 'live' or 'replay')", *mode)
	}

	// Setup HTTP handlers
	http.HandleFunc("/", server.handleRoot)
	http.HandleFunc("/stream", server.handleStream)

	log.Printf("Starting Virtex Stream Server on %s", *bindAddr)
	log.Printf("Mode: %s", *mode)
	log.Printf("Source: %s", *source)
	log.Printf("Endpoints:")
	log.Printf("  GET /        - Server info (HTML)")
	log.Printf("  GET /stream  - Virtex format stream data (JSON)")

	if err := http.ListenAndServe(*bindAddr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func (vs *VirtexServer) runLiveMode(ctx context.Context) {
	log.Printf("Starting live mode polling from %s", vs.baseURL)

	processor := processing.New()
	sessionBuffer := bytes.NewBuffer(make([]byte, 0, 64*1024))
	playerBonesBuffer := bytes.NewBuffer(make([]byte, 0, 64*1024))

	ticker := time.NewTicker(100 * time.Millisecond) // Poll at 10Hz
	defer ticker.Stop()

	vs.mu.Lock()
	vs.isPlaying = true
	vs.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sessionBuffer.Reset()
			playerBonesBuffer.Reset()

			var wg sync.WaitGroup
			wg.Add(2)

			// Fetch session data
			go func() {
				defer wg.Done()
				resp, err := vs.httpClient.Get(vs.baseURL + "/session")
				if err != nil {
					log.Printf("Failed to fetch session: %v", err)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					io.Copy(sessionBuffer, resp.Body)
				}
			}()

			// Fetch player bones data
			go func() {
				defer wg.Done()
				resp, err := vs.httpClient.Get(vs.baseURL + "/user_bones")
				if err != nil {
					log.Printf("Failed to fetch user_bones: %v", err)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					io.Copy(playerBonesBuffer, resp.Body)
				}
			}()

			wg.Wait()

			// Process the frame
			if sessionBuffer.Len() > 0 && playerBonesBuffer.Len() > 0 {
				frame, err := processor.ProcessFrame(sessionBuffer.Bytes(), playerBonesBuffer.Bytes(), time.Now())
				if err != nil {
					log.Printf("Failed to process frame: %v", err)
					continue
				}

				vs.mu.Lock()
				vs.currentFrame = frame
				vs.mu.Unlock()
			}
		}
	}
}

func (vs *VirtexServer) runReplayMode(ctx context.Context) {
	log.Printf("Starting replay mode from file: %s", vs.replayFile)

	ext := strings.ToLower(filepath.Ext(vs.replayFile))

	for {
		vs.mu.Lock()
		vs.isPlaying = true
		vs.mu.Unlock()

		var err error
		switch ext {
		case ".echoreplay":
			err = vs.playEchoReplayFile()
		case ".nevrcap":
			err = vs.playNevrCapFile()
		default:
			log.Fatalf("Unsupported file format: %s (must be .echoreplay or .nevrcap)", ext)
		}

		if err != nil {
			log.Printf("Error playing file: %v", err)
		}

		vs.mu.Lock()
		vs.isPlaying = false
		vs.mu.Unlock()

		if !vs.replayLoop {
			log.Printf("Playback finished")
			break
		}

		log.Printf("Looping playback...")
		time.Sleep(1 * time.Second)
	}
}

func (vs *VirtexServer) playEchoReplayFile() error {
	reader, err := codecs.NewEchoReplayReader(vs.replayFile)
	if err != nil {
		return fmt.Errorf("failed to open echo replay file: %w", err)
	}
	defer reader.Close()

	var lastTimestamp time.Time

	for {
		frame, err := reader.ReadFrame()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to read frame: %w", err)
		}

		// Calculate delay for 1x playback speed
		if !lastTimestamp.IsZero() {
			delay := frame.GetTimestamp().AsTime().Sub(lastTimestamp)
			if delay > 0 && delay < 10*time.Second {
				time.Sleep(delay)
			}
		}
		lastTimestamp = frame.GetTimestamp().AsTime()

		// Update current frame
		vs.mu.Lock()
		vs.currentFrame = frame
		vs.mu.Unlock()
	}

	return nil
}

func (vs *VirtexServer) playNevrCapFile() error {
	reader, err := codecs.NewNevrCapReader(vs.replayFile)
	if err != nil {
		return fmt.Errorf("failed to open nevrcap file: %w", err)
	}
	defer reader.Close()

	var lastTimestamp time.Time

	for {
		frame, err := reader.ReadFrame()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to read frame: %w", err)
		}

		// Calculate delay for 1x playback speed
		if !lastTimestamp.IsZero() {
			delay := frame.GetTimestamp().AsTime().Sub(lastTimestamp)
			if delay > 0 && delay < 10*time.Second {
				time.Sleep(delay)
			}
		}
		lastTimestamp = frame.GetTimestamp().AsTime()

		// Update current frame
		vs.mu.Lock()
		vs.currentFrame = frame
		vs.mu.Unlock()
	}

	return nil
}

func (vs *VirtexServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	vs.mu.RLock()
	isPlaying := vs.isPlaying
	hasFrame := vs.currentFrame != nil
	vs.mu.RUnlock()

	w.Header().Set("Content-Type", "text/html")

	html := `<!DOCTYPE html>
<html>
<head>
    <title>Virtex Stream Server</title>
    <style>
        body { font-family: monospace; margin: 20px; }
        .status { background: #f0f0f0; padding: 10px; margin-bottom: 20px; }
        .info { background: #f8f8f8; padding: 10px; }
    </style>
</head>
<body>
    <h1>Virtex Stream Server</h1>
    <div class="status">
        <strong>Mode:</strong> %s<br>
        <strong>Status:</strong> %s<br>
        <strong>Has Frame Data:</strong> %v<br>
        <strong>Source:</strong> %s
    </div>
    <div class="info">
        <h2>Endpoints</h2>
        <ul>
            <li><a href="/stream">/stream</a> - Get current frame in Virtex format (JSON)</li>
        </ul>
    </div>
</body>
</html>`

	status := "Stopped"
	if isPlaying {
		status = "Playing"
	}

	source := vs.baseURL
	if vs.mode == "replay" {
		source = vs.replayFile
	}

	fmt.Fprintf(w, html, vs.mode, status, hasFrame, source)
}

func (vs *VirtexServer) handleStream(w http.ResponseWriter, r *http.Request) {
	vs.mu.RLock()
	frame := vs.currentFrame
	streamLink := vs.streamLink
	vs.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")

	if frame == nil {
		w.WriteHeader(http.StatusNoContent)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "No frame data available",
		})
		return
	}

	response := vs.buildVirtexResponse(frame, streamLink)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(response)
}

func (vs *VirtexServer) buildVirtexResponse(frame *rtapi.LobbySessionStateFrame, streamLink string) *VirtexResponse {
	response := &VirtexResponse{}
	response.Data.Session = frame.GetSession()
	response.Data.StreamLink = streamLink

	// Set stream timecode from frame timestamp
	if frame.GetTimestamp() != nil {
		response.Data.StreamTimecode = frame.GetTimestamp().AsTime().Format(time.RFC3339Nano)
	}

	// Convert PlayerBones to Virtex format
	playerBones := frame.GetPlayerBones()
	if playerBones != nil {
		response.Data.Bones = vs.convertToVirtexBones(playerBones)
	}

	return response
}

func (vs *VirtexServer) convertToVirtexBones(playerBones *apigame.PlayerBonesResponse) VirtexBonesData {
	virtexBones := VirtexBonesData{
		UserBones: make([]VirtexBone, 0),
	}

	if playerBones == nil || playerBones.UserBones == nil {
		return virtexBones
	}

	// Process each player's bones
	for _, userBone := range playerBones.UserBones {
		// Each UserBones contains bone_t (translation) and bone_o (orientation/rotation)
		// The data is packed as: [x, y, z, w] repeated for each bone

		boneT := userBone.GetBoneT()
		boneO := userBone.GetBoneO()

		// Calculate number of bones (each bone has 4 components)
		numBones := len(boneT) / 4
		if numBones > len(boneO)/4 {
			numBones = len(boneO) / 4
		}

		// Convert each bone (typically 22 bones per player)
		for i := 0; i < numBones; i++ {
			bone := VirtexBone{}

			// Extract translation (bone_t)
			if i*4+3 < len(boneT) {
				bone.Translation.XY = [2]float32{boneT[i*4], boneT[i*4+1]}
				bone.Translation.ZW = [2]float32{boneT[i*4+2], boneT[i*4+3]}
			}

			// Extract rotation (bone_o)
			if i*4+3 < len(boneO) {
				bone.Rotation.XY = [2]float32{boneO[i*4], boneO[i*4+1]}
				bone.Rotation.ZW = [2]float32{boneO[i*4+2], boneO[i*4+3]}
			}

			// Set default scale
			bone.Scale3D.XY = [2]float32{1, 1}
			bone.Scale3D.ZW = [2]float32{1, 1}

			// Parameters are zeros by default
			bone.Parameters = [4]float32{0, 0, 0, 0}

			virtexBones.UserBones = append(virtexBones.UserBones, bone)
		}
	}

	return virtexBones
}
