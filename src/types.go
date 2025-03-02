package spotify

// SpotifyClient handles authentication and API requests
type SpotifyClient struct {
	ClientID     string
	ClientSecret string
	AccessToken  string
}

// Artist represents a Spotify artist
type Artist struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

// Album represents a Spotify album
type Album struct {
	Name    string   `json:"name"`
	URI     string   `json:"uri"`
	ID      string   `json:"id"`
	Artists []Artist `json:"artists"`
}

// Track represents a Spotify track
type Track struct {
	Name     string   `json:"name"`
	URI      string   `json:"uri"`
	Artists  []Artist `json:"artists"`
	Album    Album    `json:"album"`
	Duration int      `json:"duration_ms"`
}

// SearchResult represents a Spotify search result
type SearchResult struct {
	Tracks struct {
		Items []Track `json:"items"`
	} `json:"tracks"`
}

// Playlist represents a Spotify playlist
type Playlist struct {
	Name  string `json:"name"`
	URI   string `json:"uri"`
	ID    string `json:"id"`
	Owner struct {
		DisplayName string `json:"display_name"`
	} `json:"owner"`
	Tracks struct {
		Total int `json:"total"`
	} `json:"tracks"`
}

// NewReleasesResult represents Spotify's new releases
type NewReleasesResult struct {
	Albums struct {
		Items []Album `json:"items"`
	} `json:"albums"`
}

// PlaylistsResult represents a list of Spotify playlists
type PlaylistsResult struct {
	Items []Playlist `json:"items"`
	Total int        `json:"total"`
}
