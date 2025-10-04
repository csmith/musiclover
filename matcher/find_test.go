package matcher

import (
	"testing"

	"github.com/csmith/musiclover/model"
	"github.com/stretchr/testify/assert"
)

func TestFind(t *testing.T) {
	tracks := []model.LovedTrack{
		{
			Track:     "Song One",
			Artist:    "Artist One",
			TrackMBID: "mbid-1",
		},
		{
			Track:      "Song Two",
			Artist:     "Artist Two",
			ArtistMBID: "artist-mbid-2",
		},
		{
			Track:  "Song Three",
			Artist: "Artist Three",
		},
		{
			Track:  "Song Four (Remastered)",
			Artist: "Artist Four",
		},
	}

	tests := []struct {
		name          string
		tracks        []model.LovedTrack
		target        model.LovedTrack
		expectedIndex *int // nil means no match expected
	}{
		{
			name:   "find by track MBID",
			tracks: tracks,
			target: model.LovedTrack{
				Track:     "Different Name",
				Artist:    "Different Artist",
				TrackMBID: "mbid-1",
			},
			expectedIndex: intPtr(0),
		},
		{
			name:   "find by artist MBID and track name",
			tracks: tracks,
			target: model.LovedTrack{
				Track:      "Song Two",
				Artist:     "Wrong Artist",
				ArtistMBID: "artist-mbid-2",
			},
			expectedIndex: intPtr(1),
		},
		{
			name:   "find by exact match",
			tracks: tracks,
			target: model.LovedTrack{
				Track:  "Song Three",
				Artist: "Artist Three",
			},
			expectedIndex: intPtr(2),
		},
		{
			name:   "find by fuzzy match",
			tracks: tracks,
			target: model.LovedTrack{
				Track:  "Song Four",
				Artist: "Artist Four",
			},
			expectedIndex: intPtr(3),
		},
		{
			name:   "no match found",
			tracks: tracks,
			target: model.LovedTrack{
				Track:  "Nonexistent Song",
				Artist: "Nonexistent Artist",
			},
			expectedIndex: nil,
		},
		{
			name:   "empty slice returns nil",
			tracks: []model.LovedTrack{},
			target: model.LovedTrack{
				Track:  "Song",
				Artist: "Artist",
			},
			expectedIndex: nil,
		},
		{
			name: "prefers higher score match",
			tracks: []model.LovedTrack{
				{
					Track:  "Song",
					Artist: "Artist",
				},
				{
					Track:     "Song",
					Artist:    "Artist",
					TrackMBID: "mbid-123",
				},
			},
			target: model.LovedTrack{
				Track:     "Song",
				Artist:    "Artist",
				TrackMBID: "mbid-123",
			},
			expectedIndex: intPtr(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Find(tt.tracks, tt.target)

			if tt.expectedIndex == nil {
				assert.Equal(t, -1, result)
			} else {
				assert.Equal(t, *tt.expectedIndex, result)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
