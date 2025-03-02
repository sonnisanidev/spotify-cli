package spotify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (c *SpotifyClient) TogglePlayback() error {
	// Get current playback state
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me/player", nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		// No active device, try to play something to activate the player
		fmt.Println("No active device found. Try playing a track first.")
		return nil
	}

	var result struct {
		IsPlaying bool `json:"is_playing"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	// Toggle playback state
	endpoint := "play"
	if result.IsPlaying {
		endpoint = "pause"
	}

	req, err = http.NewRequest("PUT", "https://api.spotify.com/v1/me/player/"+endpoint, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("request failed with status %s", resp.Status)
	}

	return nil
}

func (c *SpotifyClient) SetVolume(volume int) error {
	if volume < 0 || volume > 100 {
		return fmt.Errorf("volume must be between 0 and 100")
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.spotify.com/v1/me/player/volume?volume_percent=%d", volume), nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("request failed with status %s", resp.Status)
	}

	return nil
}

func (c *SpotifyClient) GetCurrentTrack() error {
	// Get full player state which includes repeat and shuffle information
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me/player", nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		fmt.Println("\n\033[1;31m╔══════════════════════════════════════════════╗")
		fmt.Println("║  No track currently playing                  ║")
		fmt.Println("╚══════════════════════════════════════════════╝\033[0m")
		return nil
	}

	var result struct {
		Item struct {
			Name     string   `json:"name"`
			Artists  []Artist `json:"artists"`
			Album    Album    `json:"album"`
			Duration int      `json:"duration_ms"`
			URI      string   `json:"uri"`
		} `json:"item"`
		IsPlaying    bool `json:"is_playing"`
		ProgressMs   int  `json:"progress_ms"`
		ShuffleState bool `json:"shuffle_state"`
		RepeatState  string `json:"repeat_state"`
		Device struct {
			Name         string `json:"name"`
			Type         string `json:"type"`
			VolumePercent int    `json:"volume_percent"`
		} `json:"device"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	// Format status and progress information
	status := "\033[1;32m▶ Playing\033[0m"
	if !result.IsPlaying {
		status = "\033[1;31m⏸ Paused\033[0m"
	}

	// Calculate progress bar (30 chars wide)
	progressPercent := float64(result.ProgressMs) / float64(result.Item.Duration)
	progressBarWidth := 30
	progressChars := int(progressPercent * float64(progressBarWidth))
	
	progressBar := "["
	for i := 0; i < progressBarWidth; i++ {
		if i < progressChars {
			progressBar += "█"
		} else {
			progressBar += "░"
		}
	}
	progressBar += "]"
	
	// Format time as MM:SS
	progressTime := fmt.Sprintf("%d:%02d", result.ProgressMs/60000, (result.ProgressMs/1000)%60)
	totalTime := fmt.Sprintf("%d:%02d", result.Item.Duration/60000, (result.Item.Duration/1000)%60)
	
	// Format shuffle and repeat state
	shuffleState := "Off"
	if result.ShuffleState {
		shuffleState = "On"
	}
	
	// Format repeat state with color and description
	var repeatStateDisplay string
	switch result.RepeatState {
	case "off":
		repeatStateDisplay = "\033[1;31mOff\033[0m (No repeat)"
	case "track":
		repeatStateDisplay = "\033[1;32mTrack\033[0m (Repeat current song)"
	case "context":
		repeatStateDisplay = "\033[1;32mContext\033[0m (Repeat playlist/album)"
	default:
		repeatStateDisplay = result.RepeatState
	}

	// Create a visually appealing display
	fmt.Println("\n\033[1;36m╔══════════════════════════════════════════════════════════════════════════╗\033[0m")
	fmt.Printf("\033[1;36m║\033[0m %-74s \033[1;36m║\033[0m\n", status)
	fmt.Println("\033[1;36m╠══════════════════════════════════════════════════════════════════════════╣\033[0m")
	fmt.Printf("\033[1;36m║\033[0m \033[1;33mTrack:\033[0m %-68s \033[1;36m║\033[0m\n", truncateString(result.Item.Name, 68))
	fmt.Printf("\033[1;36m║\033[0m \033[1;33mArtist:\033[0m %-67s \033[1;36m║\033[0m\n", truncateString(formatArtists(result.Item.Artists), 67))
	fmt.Printf("\033[1;36m║\033[0m \033[1;33mAlbum:\033[0m %-68s \033[1;36m║\033[0m\n", truncateString(result.Item.Album.Name, 68))
	fmt.Println("\033[1;36m╠══════════════════════════════════════════════════════════════════════════╣\033[0m")
	fmt.Printf("\033[1;36m║\033[0m %-74s \033[1;36m║\033[0m\n", progressBar)
	fmt.Printf("\033[1;36m║\033[0m %-74s \033[1;36m║\033[0m\n", fmt.Sprintf("%s / %s", progressTime, totalTime))
	fmt.Println("\033[1;36m╠══════════════════════════════════════════════════════════════════════════╣\033[0m")
	fmt.Printf("\033[1;36m║\033[0m \033[1;33mDevice:\033[0m %-67s \033[1;36m║\033[0m\n", fmt.Sprintf("%s (%s)", result.Device.Name, result.Device.Type))
	fmt.Printf("\033[1;36m║\033[0m \033[1;33mVolume:\033[0m %-67s \033[1;36m║\033[0m\n", fmt.Sprintf("%d%%", result.Device.VolumePercent))
	fmt.Printf("\033[1;36m║\033[0m \033[1;33mShuffle:\033[0m %-66s \033[1;36m║\033[0m\n", shuffleState)
	fmt.Printf("\033[1;36m║\033[0m \033[1;33mRepeat:\033[0m %-67s \033[1;36m║\033[0m\n", repeatStateDisplay)
	fmt.Println("\033[1;36m╚══════════════════════════════════════════════════════════════════════════╝\033[0m")

	return nil
}

func (c *SpotifyClient) ToggleRepeat() error {
	// Get current playback state to determine current repeat mode
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me/player", nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		// No active device
		fmt.Println("\033[1;31mNo active device found. Try playing a track first.\033[0m")
		return nil
	}

	var result struct {
		RepeatState string `json:"repeat_state"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	// Display current state
	currentState := result.RepeatState
	fmt.Println("\n\033[1;36m╔══════════════════════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[1;36m║                      REPEAT MODE STATUS                       ║\033[0m")
	fmt.Println("\033[1;36m╠══════════════════════════════════════════════════════════════╣\033[0m")
	
	switch currentState {
	case "off":
		fmt.Println("\033[1;36m║\033[0m  Current mode: \033[1;31mOFF\033[0m                                         \033[1;36m║\033[0m")
	case "track":
		fmt.Println("\033[1;36m║\033[0m  Current mode: \033[1;32mTRACK\033[0m                                       \033[1;36m║\033[0m")
	case "context":
		fmt.Println("\033[1;36m║\033[0m  Current mode: \033[1;32mCONTEXT\033[0m                                     \033[1;36m║\033[0m")
	}

	// Determine next repeat state
	// Cycle through: off -> track -> context -> off
	var nextState string
	switch result.RepeatState {
	case "off":
		nextState = "track"
	case "track":
		nextState = "context"
	case "context":
		nextState = "off"
	default:
		nextState = "off"
	}

	fmt.Printf("\033[1;36m║\033[0m  Changing to: \033[1;33m%-48s\033[1;36m ║\033[0m\n", strings.ToUpper(nextState))
	fmt.Println("\033[1;36m╚══════════════════════════════════════════════════════════════╝\033[0m")

	// Set the new repeat state
	req, err = http.NewRequest("PUT", "https://api.spotify.com/v1/me/player/repeat?state="+nextState, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("request failed with status %s", resp.Status)
	}

	return nil
}

func (c *SpotifyClient) SetRepeatMode(mode string) error {
	// Validate the mode
	validModes := map[string]string{
		"off":     "off",
		"track":   "track",
		"song":    "track",    // alias for track
		"context": "context",
		"album":   "context",  // alias for context
		"playlist": "context", // alias for context
	}
	
	spotifyMode, valid := validModes[strings.ToLower(mode)]
	if !valid {
		return fmt.Errorf("invalid repeat mode: %s. Valid modes are: off, track, song, context, album, playlist", mode)
	}
	
	// Set the repeat state
	req, err := http.NewRequest("PUT", "https://api.spotify.com/v1/me/player/repeat?state="+spotifyMode, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("request failed with status %s", resp.Status)
	}

	// Display the new repeat state with clear text descriptions
	fmt.Println("\n\033[1;36m╔══════════════════════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[1;36m║                      REPEAT MODE SET                          ║\033[0m")
	fmt.Println("\033[1;36m╠══════════════════════════════════════════════════════════════╣\033[0m")
	
	switch spotifyMode {
	case "off":
		fmt.Println("\033[1;36m║\033[0m  Current mode: \033[1;31mOFF\033[0m                                         \033[1;36m║\033[0m")
		fmt.Println("\033[1;36m║\033[0m  Songs will play in sequence without repeating                \033[1;36m║\033[0m")
	case "track":
		fmt.Println("\033[1;36m║\033[0m  Current mode: \033[1;32mTRACK\033[0m                                       \033[1;36m║\033[0m")
		fmt.Println("\033[1;36m║\033[0m  Current song will repeat continuously                        \033[1;36m║\033[0m")
	case "context":
		fmt.Println("\033[1;36m║\033[0m  Current mode: \033[1;32mCONTEXT\033[0m                                     \033[1;36m║\033[0m")
		fmt.Println("\033[1;36m║\033[0m  Current playlist or album will repeat after finishing        \033[1;36m║\033[0m")
	}
	
	fmt.Println("\033[1;36m╚══════════════════════════════════════════════════════════════╝\033[0m")

	return nil
}

func (c *SpotifyClient) ShowRepeatMode() error {
	// Get current playback state to determine current repeat mode
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me/player", nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		// No active device
		fmt.Println("\033[1;31mNo active device found. Try playing a track first.\033[0m")
		return nil
	}

	var result struct {
		RepeatState string `json:"repeat_state"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	// Display current state with detailed explanation
	currentState := result.RepeatState
	fmt.Println("\n\033[1;36m╔══════════════════════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[1;36m║                      REPEAT MODE STATUS                       ║\033[0m")
	fmt.Println("\033[1;36m╠══════════════════════════════════════════════════════════════╣\033[0m")
	
	switch currentState {
	case "off":
		fmt.Println("\033[1;36m║\033[0m  Current mode: \033[1;31mOFF\033[0m                                         \033[1;36m║\033[0m")
		fmt.Println("\033[1;36m║\033[0m  Songs will play in sequence without repeating                \033[1;36m║\033[0m")
	case "track":
		fmt.Println("\033[1;36m║\033[0m  Current mode: \033[1;32mTRACK\033[0m                                       \033[1;36m║\033[0m")
		fmt.Println("\033[1;36m║\033[0m  Current song will repeat continuously                        \033[1;36m║\033[0m")
	case "context":
		fmt.Println("\033[1;36m║\033[0m  Current mode: \033[1;32mCONTEXT\033[0m                                     \033[1;36m║\033[0m")
		fmt.Println("\033[1;36m║\033[0m  Current playlist or album will repeat after finishing        \033[1;36m║\033[0m")
	}
	
	fmt.Println("\033[1;36m╚══════════════════════════════════════════════════════════════╝\033[0m")

	return nil
}

func (c *SpotifyClient) SkipToNext() error {
	req, err := http.NewRequest("POST", "https://api.spotify.com/v1/me/player/next", nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		fmt.Println("\n\033[1;32mSkipped to next track\033[0m")
		return nil
	}

	if resp.StatusCode == http.StatusNotFound {
		fmt.Println("\n\033[1;31mNo active device found. Try playing a track first.\033[0m")
		return nil
	}

	return fmt.Errorf("request failed with status %s", resp.Status)
}

func (c *SpotifyClient) SkipToPrevious() error {
	req, err := http.NewRequest("POST", "https://api.spotify.com/v1/me/player/previous", nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		fmt.Println("\n\033[1;32mSkipped to previous track\033[0m")
		return nil
	}

	if resp.StatusCode == http.StatusNotFound {
		fmt.Println("\n\033[1;31mNo active device found. Try playing a track first.\033[0m")
		return nil
	}

	return fmt.Errorf("request failed with status %s", resp.Status)
}
