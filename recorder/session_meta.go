package recorder

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type SessionMeta struct {
	SessionUUID    string `json:"sessionid"`
	GameStatus     string `json:"game_status"`
	MatchType      string `json:"match_type"`
	MapName        string `json:"map_name"`
	IsPrivateMatch bool   `json:"private_match"`
}

func GetSessionMeta(baseURL string) (r SessionMeta, err error) {
	client := &http.Client{
		Timeout: 3 * time.Second, // Overall request timeout
		Transport: &http.Transport{
			ExpectContinueTimeout: 1 * time.Second,
			DialContext: (&net.Dialer{
				Timeout: 1 * time.Second,
			}).DialContext,
		},
	}
	resp, err := client.Get(EndpointSession(baseURL))
	if err != nil {
		return r, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		// Active session found, proceed to read metadata
	case http.StatusNotFound:
		// There is no active session, return empty SessionMeta
		return r, nil
	default:
		// Unexpected status code, return an error
		return r, fmt.Errorf("received non-OK response: %d", resp.StatusCode)
	}
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return r, fmt.Errorf("failed to read response body: %v", err)
	}
	response := SessionMeta{}
	if err := json.Unmarshal(buf, &response); err != nil {
		return r, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	if response.SessionUUID == "" {
		return r, fmt.Errorf("session UUID is empty in response: %s", string(buf))
	}
	return response, nil
}
