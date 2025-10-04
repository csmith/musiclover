package matcher

import (
	"sort"

	"github.com/csmith/musiclover/model"
)

type SegmentResult struct {
	Matched []model.LovedTrack
	Missing []model.LovedTrack
	Extra   []model.LovedTrack
}

type matchCandidate struct {
	desiredIndex int
	actualIndex  int
	score        Score
}

// Segment compares desired tracks against actual tracks
func Segment(desired []model.LovedTrack, actual []model.LovedTrack) SegmentResult {
	result := SegmentResult{
		Matched: make([]model.LovedTrack, 0),
		Missing: make([]model.LovedTrack, 0),
		Extra:   make([]model.LovedTrack, 0),
	}

	// Find all possible matches
	var candidates []matchCandidate
	for i, desiredTrack := range desired {
		for j, actualTrack := range actual {
			score := Match(desiredTrack, actualTrack)
			if score != NoMatch {
				candidates = append(candidates, matchCandidate{
					desiredIndex: i,
					actualIndex:  j,
					score:        score,
				})
			}
		}
	}

	// Sort by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Greedy matching: pick best scores first
	matchedDesired := make(map[int]bool)
	matchedActual := make(map[int]bool)

	for _, candidate := range candidates {
		if !matchedDesired[candidate.desiredIndex] && !matchedActual[candidate.actualIndex] {
			matchedDesired[candidate.desiredIndex] = true
			matchedActual[candidate.actualIndex] = true
		}
	}

	// Populate results
	for i, desiredTrack := range desired {
		if matchedDesired[i] {
			result.Matched = append(result.Matched, desiredTrack)
		} else {
			result.Missing = append(result.Missing, desiredTrack)
		}
	}

	for j, actualTrack := range actual {
		if !matchedActual[j] {
			result.Extra = append(result.Extra, actualTrack)
		}
	}

	return result
}
