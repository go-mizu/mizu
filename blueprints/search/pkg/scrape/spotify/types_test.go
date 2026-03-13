package spotify

import "testing"

func TestParseRef(t *testing.T) {
	tests := []struct {
		in       string
		expected string
		entity   string
		id       string
	}{
		{"11dFghVXANMlKmJXsNCbNl", EntityTrack, EntityTrack, "11dFghVXANMlKmJXsNCbNl"},
		{"spotify:album:0tGPJ0bkWOUmH7MEOR77qc", EntityAlbum, EntityAlbum, "0tGPJ0bkWOUmH7MEOR77qc"},
		{"https://open.spotify.com/artist/6sFIWsNpZYqfjUpaCgueju", EntityArtist, EntityArtist, "6sFIWsNpZYqfjUpaCgueju"},
		{"https://open.spotify.com/intl-ja/playlist/37i9dQZF1DXcBWIGoYBM5M?si=abc", EntityPlaylist, EntityPlaylist, "37i9dQZF1DXcBWIGoYBM5M"},
	}

	for _, tt := range tests {
		ref, err := ParseRef(tt.in, tt.expected)
		if err != nil {
			t.Fatalf("ParseRef(%q): %v", tt.in, err)
		}
		if ref.EntityType != tt.entity || ref.ID != tt.id {
			t.Fatalf("ParseRef(%q) = (%s, %s), want (%s, %s)", tt.in, ref.EntityType, ref.ID, tt.entity, tt.id)
		}
	}
}

func TestParseCompactNumber(t *testing.T) {
	tests := map[string]int64{
		"19.3M": 19300000,
		"525K":  525000,
		"344":   344,
	}
	for in, want := range tests {
		if got := parseCompactNumber(in); got != want {
			t.Fatalf("parseCompactNumber(%q) = %d, want %d", in, got, want)
		}
	}
}
