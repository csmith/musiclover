package matcher

import (
	"testing"

	"github.com/csmith/musiclover/model"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeForMatching(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase conversion",
			input:    "Artist Name",
			expected: "artist name",
		},
		{
			name:     "remove parentheses",
			input:    "Song (Remastered)",
			expected: "song",
		},
		{
			name:     "remove feat",
			input:    "Song feat. Other Artist",
			expected: "song",
		},
		{
			name:     "remove ft",
			input:    "Song ft. Other Artist",
			expected: "song",
		},
		{
			name:     "remove featuring",
			input:    "Song featuring Other Artist",
			expected: "song",
		},
		{
			name:     "remove leading the",
			input:    "The Beatles",
			expected: "beatles",
		},
		{
			name:     "normalize whitespace",
			input:    "Song   With    Spaces",
			expected: "song with spaces",
		},
		{
			name:     "complex example",
			input:    "The Song (Live) feat. Artist",
			expected: "song",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeForMatching(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name     string
		trackA   model.LovedTrack
		trackB   model.LovedTrack
		expected Score
	}{
		{
			name: "track MBID match",
			trackA: model.LovedTrack{
				Track:     "Song",
				Artist:    "Artist",
				TrackMBID: "track-mbid-123",
			},
			trackB: model.LovedTrack{
				Track:     "Different Song",
				Artist:    "Different Artist",
				TrackMBID: "track-mbid-123",
			},
			expected: TrackMBID,
		},
		{
			name: "album and artist MBID match",
			trackA: model.LovedTrack{
				Track:      "Song",
				Artist:     "Artist",
				Album:      "Album",
				AlbumMBID:  "album-mbid-123",
				ArtistMBID: "artist-mbid-123",
			},
			trackB: model.LovedTrack{
				Track:      "Song",
				Artist:     "Artist",
				Album:      "Album",
				AlbumMBID:  "album-mbid-123",
				ArtistMBID: "artist-mbid-123",
			},
			expected: AlbumArtistMBID,
		},
		{
			name: "artist MBID and track name match",
			trackA: model.LovedTrack{
				Track:      "Song Name",
				Artist:     "Artist",
				ArtistMBID: "artist-mbid-123",
			},
			trackB: model.LovedTrack{
				Track:      "Song Name",
				Artist:     "Different Artist",
				ArtistMBID: "artist-mbid-123",
			},
			expected: ArtistMBID,
		},
		{
			name: "exact artist and track match",
			trackA: model.LovedTrack{
				Track:  "Song Name",
				Artist: "Artist Name",
			},
			trackB: model.LovedTrack{
				Track:  "Song Name",
				Artist: "Artist Name",
			},
			expected: ExactMatch,
		},
		{
			name: "exact match case insensitive",
			trackA: model.LovedTrack{
				Track:  "Song Name",
				Artist: "Artist Name",
			},
			trackB: model.LovedTrack{
				Track:  "SONG NAME",
				Artist: "ARTIST NAME",
			},
			expected: ExactMatch,
		},
		{
			name: "fuzzy match with parentheses",
			trackA: model.LovedTrack{
				Track:  "Song",
				Artist: "Artist",
			},
			trackB: model.LovedTrack{
				Track:  "Song (Remastered)",
				Artist: "Artist",
			},
			expected: FuzzyMatch,
		},
		{
			name: "fuzzy match with featuring",
			trackA: model.LovedTrack{
				Track:  "Song",
				Artist: "Artist",
			},
			trackB: model.LovedTrack{
				Track:  "Song feat. Other",
				Artist: "Artist",
			},
			expected: FuzzyMatch,
		},
		{
			name: "fuzzy match with typo",
			trackA: model.LovedTrack{
				Track:  "Song Name",
				Artist: "Artist",
			},
			trackB: model.LovedTrack{
				Track:  "Song Naem",
				Artist: "Artist",
			},
			expected: FuzzyMatch,
		},
		{
			name: "no match - different tracks",
			trackA: model.LovedTrack{
				Track:  "Song One",
				Artist: "Artist",
			},
			trackB: model.LovedTrack{
				Track:  "Completely Different Song",
				Artist: "Artist",
			},
			expected: NoMatch,
		},
		{
			name: "no match - empty fields",
			trackA: model.LovedTrack{
				Track:  "",
				Artist: "",
			},
			trackB: model.LovedTrack{
				Track:  "Song",
				Artist: "Artist",
			},
			expected: NoMatch,
		},
		{
			name: "track MBID takes precedence over exact match",
			trackA: model.LovedTrack{
				Track:     "Song",
				Artist:    "Artist",
				TrackMBID: "mbid-123",
			},
			trackB: model.LovedTrack{
				Track:     "Song",
				Artist:    "Artist",
				TrackMBID: "mbid-123",
			},
			expected: TrackMBID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Match(tt.trackA, tt.trackB)
			assert.Equal(t, tt.expected, result)
		})
	}
}
