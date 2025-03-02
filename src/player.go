package spotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

var lastPlaylists []Playlist

func convertURItoURL(uri string) string {
	// Convert spotify:track:xxx to https://open.spotify.com/track/xxx
	parts := strings.Split(uri, ":")
	if len(parts) != 3 {
		return uri
	}
	return fmt.Sprintf("https://open.spotify.com/%s/%s", parts[1], parts[2])
}

// ListPlaylists lists the user's playlists
func (c *SpotifyClient) ListPlaylists() (Playlists, error) {
	var results Playlists
	
	reqURL := "https://api.spotify.com/v1/me/playlists?limit=20"
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return results, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return results, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return results, fmt.Errorf("request failed with status %s", resp.Status)
	}

	var playlistsResponse struct {
		Items []Playlist `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&playlistsResponse); err != nil {
		return results, fmt.Errorf("error parsing response: %v", err)
	}

	results.Items = playlistsResponse.Items

	// Display the results
	fmt.Println("\n\033[1;36m╔══════════════════════════════════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[1;36m║\033[0m \033[1;33mYour Playlists:\033[0m                                                        \033[1;36m║\033[0m")
	fmt.Println("\033[1;36m╠══════════════════════════════════════════════════════════════════════════╣\033[0m")

	for i, playlist := range results.Items {
		fmt.Printf("\033[1;36m║\033[0m \033[1;32m%2d.\033[0m %-70s \033[1;36m║\033[0m\n", i+1, truncateString(playlist.Name, 70))
		if playlist.Owner.DisplayName != "" {
			fmt.Printf("\033[1;36m║\033[0m     \033[1;90mBy:\033[0m %-69s \033[1;36m║\033[0m\n", truncateString(playlist.Owner.DisplayName, 69))
		}
		fmt.Printf("\033[1;36m║\033[0m     \033[1;90mTracks:\033[0m %-66d \033[1;36m║\033[0m\n", playlist.Tracks.Total)
		if i < len(results.Items)-1 {
			fmt.Println("\033[1;36m║\033[0m                                                                          \033[1;36m║\033[0m")
		}
	}

	fmt.Println("\033[1;36m╚══════════════════════════════════════════════════════════════════════════╝\033[0m")
	return results, nil
}

func (c *SpotifyClient) PlayPlaylist(playlistID string) error {
	// Construct the URI if it's not already in the correct format
	uri := playlistID
	if !strings.HasPrefix(uri, "spotify:playlist:") {
		uri = "spotify:playlist:" + playlistID
	}

	// Try to get device ID and start playback
	var deviceID string
	for i := 0; i < 5; i++ {
		// Get available devices
		req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me/player/devices", nil)
		if err != nil {
			return fmt.Errorf("error creating request: %v", err)
		}

		req.Header.Add("Authorization", "Bearer "+c.AccessToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("error making request: %v", err)
		}

		var devices struct {
			Devices []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"devices"`
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err := json.Unmarshal(body, &devices); err != nil {
			return fmt.Errorf("error parsing devices: %v", err)
		}

		// Try to find a browser device
		for _, device := range devices.Devices {
			if device.Type == "Computer" || device.Type == "Web Player" {
				deviceID = device.ID
				break
			}
		}

		if deviceID != "" {
			break
		}
		time.Sleep(2 * time.Second)
	}

	// If no device found, open the web player
	if deviceID == "" {
		// Convert Spotify URI to web URL
		parts := strings.Split(uri, ":")
		if len(parts) != 3 {
			return fmt.Errorf("invalid Spotify URI format")
		}
		webURL := fmt.Sprintf("https://open.spotify.com/%s/%s", parts[1], parts[2])

		cmd := exec.Command("cmd", "/c", "start", "firefox.exe", webURL)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to open Firefox: %v", err)
		}
		time.Sleep(3 * time.Second)
		return nil
	}

	// Try to start playback
	requestBody := map[string]interface{}{
		"context_uri": uri,
		"device_id":   deviceID,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("error creating request body: %v", err)
	}

	req, err := http.NewRequest("PUT", "https://api.spotify.com/v1/me/player/play", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Playing playlist: %s\n", playlistID)
	return nil
}

// PlayAlbum plays an album by its ID
func (c *SpotifyClient) PlayAlbum(albumID string) error {
	// Construct the URI if it's not already in the correct format
	uri := albumID
	if !strings.HasPrefix(uri, "spotify:album:") {
		uri = "spotify:album:" + albumID
	}

	// Get available devices
	reqURL := "https://api.spotify.com/v1/me/player/devices"
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("error creating devices request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making devices request: %v", err)
	}
	defer resp.Body.Close()

	var deviceResp struct {
		Devices []struct {
			ID     string `json:"id"`
			Active bool   `json:"is_active"`
		} `json:"devices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return fmt.Errorf("error parsing device response: %v", err)
	}

	// Find active device
	var deviceID string
	for _, device := range deviceResp.Devices {
		if device.Active {
			deviceID = device.ID
			break
		}
	}

	// If no active device found, use the first available one
	if deviceID == "" && len(deviceResp.Devices) > 0 {
		deviceID = deviceResp.Devices[0].ID
	}

	// Prepare play request
	playURL := "https://api.spotify.com/v1/me/player/play"
	if deviceID != "" {
		playURL += "?device_id=" + deviceID
	}

	// Create play request body
	playBody := map[string]interface{}{
		"context_uri": uri,
	}

	playJSON, err := json.Marshal(playBody)
	if err != nil {
		return fmt.Errorf("error creating play request body: %v", err)
	}

	playReq, err := http.NewRequest("PUT", playURL, bytes.NewBuffer(playJSON))
	if err != nil {
		return fmt.Errorf("error creating play request: %v", err)
	}

	playReq.Header.Add("Authorization", "Bearer "+c.AccessToken)
	playReq.Header.Add("Content-Type", "application/json")

	playResp, err := http.DefaultClient.Do(playReq)
	if err != nil {
		return fmt.Errorf("error making play request: %v", err)
	}
	defer playResp.Body.Close()

	if playResp.StatusCode >= 400 {
		body, _ := io.ReadAll(playResp.Body)
		return fmt.Errorf("play request failed with status %s: %s", playResp.Status, body)
	}

	fmt.Printf("Playing album: %s\n", albumID)
	return nil
}

// Helper function to truncate strings that are too long and add ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
