package recorder

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

func NewHTTPFramePoller(ctx context.Context, logger *zap.Logger, client *http.Client, baseURL string, interval time.Duration, session FrameWriter) {

	// Start a goroutine to fetch data from the URLs at the specified interval

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var (
		wg                sync.WaitGroup
		sessionURL        = EndpointSession(baseURL)
		playerBonesURL    = EndpointPlayerBones(baseURL)
		sessionBuffer     = bytes.NewBuffer(make([]byte, 0, 64*1024)) // 64KB buffer
		playerBonesBuffer = bytes.NewBuffer(make([]byte, 0, 64*1024)) // 64KB buffer
	)

	requestCount := 0
	dataWritten := 0

	defer session.Close()

	go func() {
		<-ctx.Done()
		logger.Debug("HTTP frame poller done", zap.Int("request_count", requestCount), zap.Int("data_written", dataWritten))
	}()

	timeoutTimer := time.NewTimer(5 * time.Second)
	for {

		select {
		case <-ctx.Done():
			return
		case <-timeoutTimer.C:
			logger.Debug("HTTP frame poller timeout, stopping", zap.Int("request_count", requestCount), zap.Int("data_written", dataWritten))
			return
		case <-ticker.C:
		}

		wg.Add(2)
		// Reset the buffers
		for url, buf := range map[string]*bytes.Buffer{
			sessionURL:     sessionBuffer,
			playerBonesURL: playerBonesBuffer,
		} {
			buf.Reset()
			requestCount++
			go func() {
				defer wg.Done()
				resp, err := client.Get(url)
				if err != nil {
					logger.Warn("Failed to fetch data from URL", zap.String("url", url), zap.Error(err))
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					logger.Warn("Received non-OK response from URL", zap.String("url", url), zap.Int("status_code", resp.StatusCode))
					// If the response is not OK, we can skip processing this URL
					return
				}

				// Use a buffer to read the response body
				n, err := io.Copy(buf, resp.Body)
				if err != nil {
					logger.Warn("Failed to read response body", zap.String("url", url), zap.Error(err))
					return
				}
				dataWritten += int(n)
			}()
		}

		wg.Wait()

		// Check if the context is done before processing the data
		select {
		case <-ctx.Done():
			return
		default:
		}
		// Create a new FrameData with the fetched data
		frameData := &FrameData{
			Timestamp:      time.Now(),
			SessionData:    sessionBuffer.Bytes(),
			PlayerBoneData: playerBonesBuffer.Bytes(),
		}
		// Write the data to the FrameWriter
		if err := session.WriteFrame(frameData); err != nil {
			logger.Error("Failed to write frame data",
				zap.Error(err))
			continue
		}
		timeoutTimer.Reset(5 * time.Second) // Reset the timer for the next iteration
	}
}
