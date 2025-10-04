package matcher

import (
	"strings"

	"github.com/agnivade/levenshtein"
	"github.com/csmith/musiclover/model"
)

// Score represents the quality of a match between two tracks
type Score int

const (
	NoMatch         Score = 0
	FuzzyMatch      Score = 1
	ExactMatch      Score = 2
	ArtistMBID      Score = 3
	AlbumArtistMBID Score = 4
	TrackMBID       Score = 5
)

const maxLevenshteinDistance = 3

// Match compares two LovedTracks and returns a score indicating match quality
func Match(a, b model.LovedTrack) Score {
	// Best match: track MBID
	if a.TrackMBID != "" && b.TrackMBID != "" && a.TrackMBID == b.TrackMBID {
		return TrackMBID
	}

	// Album + Artist MBID match
	if a.AlbumMBID != "" && b.AlbumMBID != "" && a.AlbumMBID == b.AlbumMBID &&
		a.ArtistMBID != "" && b.ArtistMBID != "" && a.ArtistMBID == b.ArtistMBID {
		return AlbumArtistMBID
	}

	// Artist MBID + track name match
	if a.ArtistMBID != "" && b.ArtistMBID != "" && a.ArtistMBID == b.ArtistMBID &&
		a.Track != "" && b.Track != "" && strings.EqualFold(a.Track, b.Track) {
		return ArtistMBID
	}

	// Exact artist and track name match
	if a.Artist != "" && b.Artist != "" && a.Track != "" && b.Track != "" &&
		strings.EqualFold(a.Artist, b.Artist) && strings.EqualFold(a.Track, b.Track) {
		return ExactMatch
	}

	// Fuzzy match on artist + track name
	if a.Artist != "" && b.Artist != "" && a.Track != "" && b.Track != "" {
		aKey := normalizeForMatching(a.Artist) + "|" + normalizeForMatching(a.Track)
		bKey := normalizeForMatching(b.Artist) + "|" + normalizeForMatching(b.Track)
		distance := levenshtein.ComputeDistance(aKey, bKey)
		if distance <= maxLevenshteinDistance {
			return FuzzyMatch
		}
	}

	return NoMatch
}

func normalizeForMatching(s string) string {
	s = strings.ToLower(s)

	// Remove anything in parentheses
	for {
		start := strings.Index(s, "(")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], ")")
		if end == -1 {
			break
		}
		s = s[:start] + s[start+end+1:]
	}

	// Remove anything after feat/ft/featuring
	for _, sep := range []string{" feat.", " feat ", " ft.", " ft ", " featuring "} {
		if idx := strings.Index(s, sep); idx != -1 {
			s = s[:idx]
		}
	}

	// Clean up whitespace
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")

	// Remove "the " from the start
	s = strings.TrimPrefix(s, "the ")

	return s
}
