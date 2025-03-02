package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	
	spotify "spotify-cli/src" // Import the spotify package
)

// Global variables to store search results
var (
	lastSearchResults spotify.SearchResults
	lastNewReleases   spotify.NewReleases
	lastPlaylists     spotify.Playlists
)

func loadEnv(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// .env file doesn't exist, that's okay
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		os.Setenv(key, value)
	}
	return scanner.Err()
}

func NewSpotifyClient() (*spotify.SpotifyClient, error) {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set in .env file")
	}

	return &spotify.SpotifyClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}, nil
}

func main() {
	// Load environment variables
	if err := loadEnv(".env"); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	// Create Spotify client
	client, err := NewSpotifyClient()
	if err != nil {
		log.Fatal("Error creating Spotify client:", err)
	}

	// Start the authorization flow
	fmt.Println("Starting Spotify CLI...")
	if err := client.StartAuthFlow(); err != nil {
		log.Fatal("Error during authorization:", err)
	}

	fmt.Println("Successfully authenticated with Spotify!")

	// Start command loop
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\nCommands:")
		fmt.Println("1. search <query> - Search for tracks")
		fmt.Println("2. play <number> - Play track from search results")
		fmt.Println("3. new - Show new releases")
		fmt.Println("4. play-new <number> - Play album from new releases")
		fmt.Println("5. current - Show current track")
		fmt.Println("6. toggle - Play/Pause")
		fmt.Println("7. playlists - List all playlists")
		fmt.Println("8. play-list <number> - Play playlist from list")
		fmt.Println("9. volume <0-100> - Set playback volume")
		fmt.Println("10. repeat - Toggle repeat mode (off/track/context)")
		fmt.Println("11. repeat-mode <mode> - Set repeat mode (off/track/context/song/album/playlist)")
		fmt.Println("12. next - Skip to next track")
		fmt.Println("13. prev - Go back to previous track")
		fmt.Println("14. quit - Exit the program")
		fmt.Print("\nEnter command: ")

		command, _ := reader.ReadString('\n')
		command = strings.TrimSpace(command)

		switch {
		case command == "quit":
			fmt.Println("Goodbye!")
			return
		case command == "new":
			results, err := client.ShowNewReleases()
			if err != nil {
				fmt.Println("Error:", err)
				// Try refreshing the token if it expired
				if err := client.RefreshToken(); err == nil {
					results, err = client.ShowNewReleases()
					if err != nil {
						fmt.Println("Error after token refresh:", err)
						continue
					}
					// If successful after token refresh, update lastNewReleases
					lastNewReleases = results
					continue
				}
				continue
			}
			lastNewReleases = results
		case command == "current":
			err = client.GetCurrentTrack()
			if err != nil {
				fmt.Println("Error:", err)
				// Try refreshing the token if it expired
				if err := client.RefreshToken(); err == nil {
					if err := client.GetCurrentTrack(); err != nil {
						fmt.Println("Error after token refresh:", err)
					}
				}
			}
		case command == "toggle":
			err = client.TogglePlayback()
			if err != nil {
				fmt.Println("Error:", err)
				// Try refreshing the token if it expired
				if err := client.RefreshToken(); err == nil {
					if err := client.TogglePlayback(); err != nil {
						fmt.Println("Error after token refresh:", err)
					}
				}
			}
		case command == "playlists":
			results, err := client.ListPlaylists()
			if err != nil {
				fmt.Println("Error:", err)
				// Try refreshing the token if it expired
				if err := client.RefreshToken(); err == nil {
					results, err = client.ListPlaylists()
					if err != nil {
						fmt.Println("Error after token refresh:", err)
						continue
					}
					// If successful after token refresh, update lastPlaylists
					lastPlaylists = results
					continue
				}
				continue
			}
			lastPlaylists = results
		case strings.HasPrefix(command, "play "):
			numStr := strings.TrimPrefix(command, "play ")
			num, err := strconv.Atoi(numStr)
			if err != nil || num < 1 || num > len(lastSearchResults.Tracks) {
				fmt.Println("Invalid track number")
				continue
			}

			track := lastSearchResults.Tracks[num-1]
			err = client.PlayTrack(track.URI)
			if err != nil {
				fmt.Println("Error:", err)
				// Try refreshing the token if it expired
				if err := client.RefreshToken(); err == nil {
					if err := client.PlayTrack(track.URI); err != nil {
						fmt.Println("Error after token refresh:", err)
					}
				}
			}
		case strings.HasPrefix(command, "play-new "):
			numStr := strings.TrimPrefix(command, "play-new ")
			num, err := strconv.Atoi(numStr)
			if err != nil || num < 1 || num > len(lastNewReleases.Albums) {
				fmt.Println("Invalid album number")
				continue
			}

			album := lastNewReleases.Albums[num-1]
			err = client.PlayAlbum(album.ID)
			if err != nil {
				fmt.Println("Error:", err)
				// Try refreshing the token if it expired
				if err := client.RefreshToken(); err == nil {
					if err := client.PlayAlbum(album.ID); err != nil {
						fmt.Println("Error after token refresh:", err)
					}
				}
			}
		case strings.HasPrefix(command, "play-list "):
			numStr := strings.TrimPrefix(command, "play-list ")
			num, err := strconv.Atoi(numStr)
			if err != nil || num < 1 || num > len(lastPlaylists.Items) {
				fmt.Println("Invalid playlist number")
				continue
			}

			playlist := lastPlaylists.Items[num-1]
			err = client.PlayPlaylist(playlist.ID)
			if err != nil {
				fmt.Println("Error:", err)
				// Try refreshing the token if it expired
				if err := client.RefreshToken(); err == nil {
					if err := client.PlayPlaylist(playlist.ID); err != nil {
						fmt.Println("Error after token refresh:", err)
					}
				}
			}
		case strings.HasPrefix(command, "volume "):
			volStr := strings.TrimPrefix(command, "volume ")
			vol, err := strconv.Atoi(strings.TrimSpace(volStr))
			if err != nil {
				fmt.Println("Error: Please provide a valid volume number between 0 and 100")
				continue
			}
			err = client.SetVolume(vol)
			if err != nil {
				fmt.Println("Error:", err)
				// Try refreshing the token if it expired
				if err := client.RefreshToken(); err == nil {
					if err := client.SetVolume(vol); err != nil {
						fmt.Println("Error after token refresh:", err)
					}
				}
			}
		case strings.HasPrefix(command, "search "):
			query := strings.TrimPrefix(command, "search ")
			results, err := client.SearchTracks(query)
			if err != nil {
				fmt.Println("Error:", err)
				// Try refreshing the token if it expired
				if err := client.RefreshToken(); err == nil {
					results, err = client.SearchTracks(query)
					if err != nil {
						fmt.Println("Error after token refresh:", err)
						continue
					}
					// If successful after token refresh, update lastSearchResults
					lastSearchResults = results
					continue
				}
				continue
			}
			lastSearchResults = results
		case command == "repeat":
			err = client.ToggleRepeat()
			if err != nil {
				fmt.Println("Error:", err)
				// Try refreshing the token if it expired
				if err := client.RefreshToken(); err == nil {
					if err := client.ToggleRepeat(); err != nil {
						fmt.Println("Error after token refresh:", err)
					}
				}
			}
		case strings.HasPrefix(command, "repeat-mode "):
			mode := strings.TrimPrefix(command, "repeat-mode ")
			err = client.SetRepeatMode(mode)
			if err != nil {
				fmt.Println("Error:", err)
				// Try refreshing the token if it expired
				if err := client.RefreshToken(); err == nil {
					if err := client.SetRepeatMode(mode); err != nil {
						fmt.Println("Error after token refresh:", err)
					}
				}
			}
		case command == "next":
			err = client.SkipToNext()
			if err != nil {
				fmt.Println("Error:", err)
				// Try refreshing the token if it expired
				if err := client.RefreshToken(); err == nil {
					if err := client.SkipToNext(); err != nil {
						fmt.Println("Error after token refresh:", err)
					}
				}
			}
		case command == "prev":
			err = client.SkipToPrevious()
			if err != nil {
				fmt.Println("Error:", err)
				// Try refreshing the token if it expired
				if err := client.RefreshToken(); err == nil {
					if err := client.SkipToPrevious(); err != nil {
						fmt.Println("Error after token refresh:", err)
					}
				}
			}
		default:
			fmt.Println("Unknown command")
		}
	}
}
