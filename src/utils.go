package spotify

import "strings"

func formatArtists(artists []Artist) string {
	names := make([]string, len(artists))
	for i, artist := range artists {
		names[i] = artist.Name
	}
	return strings.Join(names, ", ")
}
