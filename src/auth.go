package spotify

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	mathrand "math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Generate a random string for state parameter
func generateRandomString(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// Fallback to less secure but functional method if crypto/rand fails
		return fallbackRandomString(length)
	}
	return base64.URLEncoding.EncodeToString(b)[:length]
}

// Fallback random string generator using math/rand
func fallbackRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func (c *SpotifyClient) StartAuthFlow() error {
	// Check if we have a refresh token saved
	refreshToken, err := os.ReadFile(".refresh_token")
	if err == nil && len(refreshToken) > 0 {
		// Try to use the refresh token
		if err := c.refreshAccessToken(string(refreshToken)); err == nil {
			return nil
		}
		// If refresh fails, continue with new auth flow
	}

	// Start authorization code flow
	redirectURI := os.Getenv("SPOTIFY_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://localhost:8888/callback"
	}

	// Define the scopes we need
	scopes := []string{
		"user-read-private",
		"user-read-email",
		"user-read-playback-state",
		"user-modify-playback-state",
		"user-read-currently-playing",
		"playlist-read-private",
		"playlist-read-collaborative",
	}

	// Generate a random state value
	state := generateRandomString(16)
	fmt.Printf("Generated state: %s\n", state)

	// Save the state to a temporary file for later verification
	if err := os.WriteFile(".auth_state", []byte(state), 0600); err != nil {
		return fmt.Errorf("error saving auth state: %v", err)
	}

	// Build the authorization URL
	authURL := "https://accounts.spotify.com/authorize"
	params := url.Values{}
	params.Add("client_id", c.ClientID)
	params.Add("response_type", "code")
	params.Add("redirect_uri", redirectURI)
	params.Add("scope", strings.Join(scopes, " "))
	params.Add("state", state)

	authFullURL := authURL + "?" + params.Encode()

	// Open the URL in the browser
	fmt.Printf("Please open the following URL in your browser:\n%s\n", authFullURL)
	fmt.Println("After authorizing, you will be redirected to a URL.")
	fmt.Println("Option 1: Paste the full redirect URL here")
	fmt.Println("Option 2: Type 'manual' to enter the authorization code and state manually")
	fmt.Print("Enter your choice: ")

	// Read the redirected URL from user input
	reader := bufio.NewReader(os.Stdin)
	redirectedURL, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}
	redirectedURL = strings.TrimSpace(redirectedURL)
	
	// Check if the user entered a URL or wants to use manual entry
	if strings.HasPrefix(redirectedURL, "manual") {
		// Manual entry mode
		fmt.Println("Enter the authorization code:")
		code, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading code: %v", err)
		}
		code = strings.TrimSpace(code)
		
		fmt.Println("Enter the state parameter:")
		receivedState, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading state: %v", err)
		}
		receivedState = strings.TrimSpace(receivedState)
		
		// Read the saved state for verification
		savedState, err := os.ReadFile(".auth_state")
		if err != nil {
			return fmt.Errorf("error reading saved auth state: %v", err)
		}
		
		fmt.Printf("Saved state: %s\n", string(savedState))
		fmt.Printf("Received state: %s\n", receivedState)
		
		if receivedState != string(savedState) {
			return fmt.Errorf("state mismatch, possible CSRF attack. This could happen if:\n" +
				"1. The authorization process was interrupted or timed out\n" +
				"2. You're using an old or invalid redirect URL\n" +
				"3. You started multiple authorization processes\n" +
				"Please try again from the beginning.")
		}
		
		// Clean up the state file after verification
		os.Remove(".auth_state")
		
		if code == "" {
			return fmt.Errorf("no authorization code provided")
		}
		
		// Continue with token exchange using the manually entered code
		return c.exchangeCodeForToken(code, redirectURI)
	}
	
	fmt.Printf("Received URL: %s\n", redirectedURL)

	// Parse the URL to extract the authorization code
	parsedURL, err := url.Parse(redirectedURL)
	if err != nil {
		return fmt.Errorf("error parsing redirected URL: %v", err)
	}

	queryParams := parsedURL.Query()
	if queryParams.Get("error") != "" {
		return fmt.Errorf("authorization error: %s", queryParams.Get("error"))
	}

	// Read the saved state for verification
	savedState, err := os.ReadFile(".auth_state")
	if err != nil {
		return fmt.Errorf("error reading saved auth state: %v", err)
	}
	
	receivedState := queryParams.Get("state")
	fmt.Printf("Saved state: %s\n", string(savedState))
	fmt.Printf("Received state: %s\n", receivedState)

	if receivedState != string(savedState) {
		return fmt.Errorf("state mismatch, possible CSRF attack. This could happen if:\n" +
			"1. The authorization process was interrupted or timed out\n" +
			"2. You're using an old or invalid redirect URL\n" +
			"3. You started multiple authorization processes\n" +
			"Please try again from the beginning.")
	}

	// Clean up the state file after verification
	os.Remove(".auth_state")

	code := queryParams.Get("code")
	if code == "" {
		return fmt.Errorf("no authorization code found in the redirected URL")
	}
	
	// Exchange the authorization code for an access token
	return c.exchangeCodeForToken(code, redirectURI)
}

// Exchange authorization code for access token
func (c *SpotifyClient) exchangeCodeForToken(code string, redirectURI string) error {
	tokenURL := "https://accounts.spotify.com/api/token"
	auth := base64.StdEncoding.EncodeToString([]byte(c.ClientID + ":" + c.ClientSecret))

	formData := url.Values{}
	formData.Add("grant_type", "authorization_code")
	formData.Add("code", code)
	formData.Add("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("error creating token request: %v", err)
	}

	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making token request: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading token response: %v", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("error parsing token response: %v", err)
	}

	if result.AccessToken == "" {
		return fmt.Errorf("no access token received: %s", string(body))
	}

	c.AccessToken = result.AccessToken

	// Save the refresh token for future use
	if result.RefreshToken != "" {
		if err := os.WriteFile(".refresh_token", []byte(result.RefreshToken), 0600); err != nil {
			fmt.Printf("Warning: Could not save refresh token: %v\n", err)
		}
	}

	return nil
}

func (c *SpotifyClient) refreshAccessToken(refreshToken string) error {
	tokenURL := "https://accounts.spotify.com/api/token"
	auth := base64.StdEncoding.EncodeToString([]byte(c.ClientID + ":" + c.ClientSecret))

	formData := url.Values{}
	formData.Add("grant_type", "refresh_token")
	formData.Add("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("error creating refresh token request: %v", err)
	}

	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making refresh token request: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("error parsing refresh token response: %v", err)
	}

	if result.AccessToken == "" {
		return fmt.Errorf("no access token received from refresh")
	}

	c.AccessToken = result.AccessToken

	// Save the new refresh token if provided
	if result.RefreshToken != "" {
		if err := os.WriteFile(".refresh_token", []byte(result.RefreshToken), 0600); err != nil {
			fmt.Printf("Warning: Could not save refresh token: %v\n", err)
		}
	}

	return nil
}

func (c *SpotifyClient) RefreshToken() error {
	// Try to read the saved refresh token
	refreshToken, err := os.ReadFile(".refresh_token")
	if err != nil || len(refreshToken) == 0 {
		// If no refresh token, start a new auth flow
		return c.StartAuthFlow()
	}

	// Try to refresh the token
	if err := c.refreshAccessToken(string(refreshToken)); err != nil {
		// If refresh fails, start a new auth flow
		return c.StartAuthFlow()
	}

	return nil
}
