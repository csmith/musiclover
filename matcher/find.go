package matcher

import "github.com/csmith/musiclover/model"

// Find searches for the best matching track in a slice
// Returns the index of the best match, or -1 if no match is found
func Find(tracks []model.LovedTrack, target model.LovedTrack) int {
	bestIndex := -1
	bestScore := NoMatch

	for i := range tracks {
		score := Match(tracks[i], target)
		if score > bestScore {
			bestScore = score
			bestIndex = i
		}
	}

	return bestIndex
}
