# Spotify CLI

A command-line interface for controlling Spotify playback, searching for tracks, and managing playlists directly from your terminal.

## Features

- 🔍 Search for tracks on Spotify
- ▶️ Control playback (play, pause, next, previous)
- 🔊 Adjust volume
- 🔄 Toggle repeat modes
- 📋 List and play your playlists
- 🆕 Browse new releases
- 🎵 View current playing track information

## Prerequisites

- Go 1.16 or higher
- A Spotify Premium account
- Spotify Developer credentials

## Installation

### Clone the repository

```bash
git clone https://github.com/yourusername/spotify-cli.git
cd spotify-cli
```

### Build the application

```bash
go build -o spotify-cli
```

## Setup

### 1. Create a Spotify Developer Application

1. Go to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard/)
2. Log in with your Spotify account
3. Click "Create an App"
4. Fill in the app name and description
5. Once created, note down the **Client ID** and **Client Secret**
6. Click "Edit Settings" and add `http://localhost:8888/callback` as a Redirect URI

### 2. Configure Environment Variables

Create a `.env` file in the root directory with the following content:

```
SPOTIFY_CLIENT_ID=your_client_id_here
SPOTIFY_CLIENT_SECRET=your_client_secret_here
SPOTIFY_REDIRECT_URI=http://localhost:8888/callback
```

Replace `your_client_id_here` and `your_client_secret_here` with the values from your Spotify Developer Dashboard.

## Usage

### Run the application

```bash
./spotify-cli
```

On first run, the application will open your browser to authenticate with Spotify. After successful authentication, you'll be redirected back to the application.

### Available Commands

- `search <query>` - Search for tracks
- `play <number>` - Play track from search results
- `new` - Show new releases
- `play-new <number>` - Play album from new releases
- `current` - Show current track
- `toggle` - Play/Pause
- `playlists` - List all playlists
- `play-list <number>` - Play playlist from list
- `volume <0-100>` - Set playback volume
- `repeat` - Toggle repeat mode (off/track/context)
- `repeat-mode <mode>` - Set repeat mode (off/track/context/song/album/playlist)
- `next` - Skip to next track
- `prev` - Go back to previous track
- `quit` - Exit the program

## Project Structure

- `main.go` - Entry point and command handling
- `src/` - Package containing all Spotify functionality
  - `auth.go` - Authentication handling
  - `playback.go` - Playback control functions
  - `search.go` - Search functionality
  - `player.go` - Playlist management
  - `types.go` - Data structures
  - `utils.go` - Utility functions

## Authentication Flow

The application uses the OAuth 2.0 Authorization Code flow:

1. On first run, it opens a browser to authenticate with Spotify
2. After successful authentication, Spotify redirects back with an authorization code
3. The application exchanges this code for access and refresh tokens
4. The refresh token is stored in `.refresh_token` for future sessions
5. If the access token expires, it's automatically refreshed

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgements

- [Spotify Web API](https://developer.spotify.com/documentation/web-api/)
- [Go Programming Language](https://golang.org/)