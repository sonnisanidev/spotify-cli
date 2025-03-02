package spotify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// SearchResults holds the search results
type SearchResults struct {
	Tracks []Track
}

// NewReleases holds the new releases
type NewReleases struct {
	Albums []Album
}

// Playlists holds the user's playlists
type Playlists struct {
	Items []Playlist
}

func (c *SpotifyClient) SearchTracks(query string) (SearchResults, error) {
	var results SearchResults
	
	// URL encode the query
	encodedQuery := url.QueryEscape(query)
	reqURL := fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=track&limit=10", encodedQuery)
	
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

	var searchResponse struct {
		Tracks struct {
			Items []Track `json:"items"`
		} `json:"tracks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return results, fmt.Errorf("error parsing response: %v", err)
	}

	results.Tracks = searchResponse.Tracks.Items

	// Display the results
	fmt.Println("\n\033[1;36m╔══════════════════════════════════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[1;36m║\033[0m \033[1;33mSearch Results:\033[0m                                                        \033[1;36m║\033[0m")
	fmt.Println("\033[1;36m╠══════════════════════════════════════════════════════════════════════════╣\033[0m")

	for i, track := range results.Tracks {
		fmt.Printf("\033[1;36m║\033[0m \033[1;32m%2d.\033[0m %-70s \033[1;36m║\033[0m\n", i+1, truncateString(track.Name, 70))
		fmt.Printf("\033[1;36m║\033[0m     \033[1;90mArtist:\033[0m %-66s \033[1;36m║\033[0m\n", truncateString(formatArtists(track.Artists), 66))
		fmt.Printf("\033[1;36m║\033[0m     \033[1;90mAlbum:\033[0m %-67s \033[1;36m║\033[0m\n", truncateString(track.Album.Name, 67))
		if i < len(results.Tracks)-1 {
			fmt.Println("\033[1;36m║\033[0m                                                                          \033[1;36m║\033[0m")
		}
	}

	fmt.Println("\033[1;36m╚══════════════════════════════════════════════════════════════════════════╝\033[0m")
	return results, nil
}

// ShowNewReleases displays new album releases
func (c *SpotifyClient) ShowNewReleases() (NewReleases, error) {
	var results NewReleases
	
	reqURL := "https://api.spotify.com/v1/browse/new-releases?limit=10"
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

	var newReleasesResponse struct {
		Albums struct {
			Items []Album `json:"items"`
		} `json:"albums"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&newReleasesResponse); err != nil {
		return results, fmt.Errorf("error parsing response: %v", err)
	}

	results.Albums = newReleasesResponse.Albums.Items

	// Display the results
	fmt.Println("\n\033[1;36m╔══════════════════════════════════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[1;36m║\033[0m \033[1;33mNew Releases:\033[0m                                                          \033[1;36m║\033[0m")
	fmt.Println("\033[1;36m╠══════════════════════════════════════════════════════════════════════════╣\033[0m")

	for i, album := range results.Albums {
		fmt.Printf("\033[1;36m║\033[0m \033[1;32m%2d.\033[0m %-70s \033[1;36m║\033[0m\n", i+1, truncateString(album.Name, 70))
		fmt.Printf("\033[1;36m║\033[0m     \033[1;90mArtist:\033[0m %-66s \033[1;36m║\033[0m\n", truncateString(formatArtists(album.Artists), 66))
		if i < len(results.Albums)-1 {
			fmt.Println("\033[1;36m║\033[0m                                                                          \033[1;36m║\033[0m")
		}
	}

	fmt.Println("\033[1;36m╚══════════════════════════════════════════════════════════════════════════╝\033[0m")
	return results, nil
}
