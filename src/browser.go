package spotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Browser type to use for playback
type Browser string

const (
	Firefox Browser = "firefox"
	Chrome  Browser = "chrome"
)

// GetPreferredBrowser returns the preferred browser from environment or default to Firefox
func GetPreferredBrowser() Browser {
	browser := os.Getenv("SPOTIFY_PREFERRED_BROWSER")
	if strings.EqualFold(browser, "chrome") {
		return Chrome
	}
	return Firefox
}

// Convert Spotify URI to web URL
func convertURItoURL(uri string) string {
	parts := strings.Split(uri, ":")
	if len(parts) < 3 {
		return "https://open.spotify.com"
	}

	resourceType := parts[1]
	resourceID := parts[2]
	return fmt.Sprintf("https://open.spotify.com/%s/%s", resourceType, resourceID)
}

func (c *SpotifyClient) PlayTrack(uri string) error {
	// First, try to play the track via the Spotify API
	err := c.playTrackViaAPI(uri)
	if err == nil {
		return nil
	}

	// If API playback fails, fall back to browser playback
	fmt.Printf("API playback failed: %v\nFalling back to browser playback...\n", err)

	// Convert Spotify URI to web URL
	webURL := convertURItoURL(uri)
	
	// Get preferred browser
	browser := GetPreferredBrowser()
	fmt.Printf("Opening %s in %s...\n", webURL, browser)

	// Try to play in the browser
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Use PowerShell to find and activate the browser
		var psScript string
		
		if browser == Chrome {
			psScript = fmt.Sprintf(`
				$url = "%s"
				
				# Try to navigate Chrome
				$chromeProcess = Get-Process chrome -ErrorAction SilentlyContinue
				if ($chromeProcess) {
					# Try to activate the Chrome window
					$wshell = New-Object -ComObject wscript.shell
					if ($wshell.AppActivate('Chrome')) {
						Write-Host "Chrome window activated"
						# Now we'll use the clipboard to pass the URL
						Set-Clipboard -Value $url
						# Simulate Ctrl+L to focus address bar
						$wshell.SendKeys('^l')
						Start-Sleep -Milliseconds 100
						# Enter the URL
						$wshell.SendKeys($url)
						Start-Sleep -Milliseconds 100
						# Press Enter
						$wshell.SendKeys('{ENTER}')
					} else {
						# Couldn't activate, open in new tab
						Start-Process "chrome.exe" -ArgumentList "$url"
					}
				} else {
					# Chrome is not running, start it
					Start-Process "chrome.exe" "$url"
				}
			`, webURL)
		} else {
			// Default to Firefox
			psScript = fmt.Sprintf(`
				$url = "%s"
				
				# Try to navigate Firefox
				$firefoxProcess = Get-Process firefox -ErrorAction SilentlyContinue
				if ($firefoxProcess) {
					# Try to activate the Firefox window
					$wshell = New-Object -ComObject wscript.shell
					if ($wshell.AppActivate('Firefox')) {
						Write-Host "Firefox window activated"
						# Now we'll use the clipboard to pass the URL
						Set-Clipboard -Value $url
						# Simulate Ctrl+L to focus address bar
						$wshell.SendKeys('^l')
						Start-Sleep -Milliseconds 100
						# Enter the URL
						$wshell.SendKeys($url)
						Start-Sleep -Milliseconds 100
						# Press Enter
						$wshell.SendKeys('{ENTER}')
					} else {
						# Couldn't activate, open in new tab
						Start-Process "firefox.exe" -ArgumentList "-new-tab", "$url"
					}
				} else {
					# Firefox is not running, start it
					Start-Process "firefox.exe" "$url"
				}
			`, webURL)
		}

		cmd = exec.Command("powershell", "-Command", psScript)
	} else {
		// For non-Windows systems
		if browser == Chrome {
			cmd = exec.Command("google-chrome", webURL)
		} else {
			cmd = exec.Command("firefox", webURL)
		}
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error controlling %s: %v", browser, err)
	}

	return nil
}

func (c *SpotifyClient) playTrackViaAPI(uri string) error {
	// First, check for available devices
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me/player/devices", nil)
	if err != nil {
		return fmt.Errorf("error creating devices request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making devices request: %v", err)
	}
	defer resp.Body.Close()

	var deviceResult struct {
		Devices []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Type   string `json:"type"`
			Active bool   `json:"is_active"`
		} `json:"devices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&deviceResult); err != nil {
		return fmt.Errorf("error parsing devices response: %v", err)
	}

	// Check if we have any devices
	if len(deviceResult.Devices) == 0 {
		return fmt.Errorf("no available Spotify devices found")
	}

	// First try to find an active device
	var deviceID string
	for _, device := range deviceResult.Devices {
		if device.Active {
			deviceID = device.ID
			fmt.Printf("Using active device: %s\n", device.Name)
			break
		}
	}

	// If no active device, look for any device with preference for web player
	if deviceID == "" {
		for _, device := range deviceResult.Devices {
			if strings.Contains(strings.ToLower(device.Name), "web player") {
				deviceID = device.ID
				fmt.Printf("Using Web Player device: %s\n", device.Name)
				break
			}
		}
	}

	// If still no device, use the first available one
	if deviceID == "" && len(deviceResult.Devices) > 0 {
		deviceID = deviceResult.Devices[0].ID
		fmt.Printf("Using device: %s\n", deviceResult.Devices[0].Name)
	}

	// If we found a device, play the track on it
	if deviceID != "" {
		// Determine if this is a track or album URI
		uriParts := strings.Split(uri, ":")
		isAlbum := false
		if len(uriParts) >= 2 && uriParts[1] == "album" {
			isAlbum = true
			fmt.Println("Detected album URI, will play entire album")
		}

		// Create appropriate request body based on URI type
		var playBody map[string]interface{}
		if isAlbum {
			// For albums, use context_uri
			playBody = map[string]interface{}{
				"context_uri": uri,
			}
		} else {
			// For tracks, use uris array
			playBody = map[string]interface{}{
				"uris": []string{uri},
			}
		}

		playJSON, err := json.Marshal(playBody)
		if err != nil {
			return fmt.Errorf("error marshaling play request: %v", err)
		}

		playURL := "https://api.spotify.com/v1/me/player/play"
		if deviceID != "" {
			playURL += "?device_id=" + deviceID
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

		if playResp.StatusCode != http.StatusNoContent && playResp.StatusCode != http.StatusOK {
			// Read the error response for better debugging
			errorBody, _ := io.ReadAll(playResp.Body)
			return fmt.Errorf("play request failed with status %s: %s", playResp.Status, string(errorBody))
		}

		if isAlbum {
			fmt.Println("Playing album in Spotify player")
		} else {
			fmt.Println("Playing track in Spotify player")
		}
		return nil
	}

	return fmt.Errorf("no suitable device found for playback")
}
