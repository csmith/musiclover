package sources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/csmith/musiclover/model"
)

// ListenBrainz is a source that retrieves loved tracks from ListenBrainz
type ListenBrainz struct {
	Token    string
	Username string
}

type listenBrainzFeedbackResponse struct {
	Feedback   []listenBrainzFeedback `json:"feedback"`
	Offset     int                    `json:"offset"`
	Count      int                    `json:"count"`
	TotalCount int                    `json:"total_count"`
}

type listenBrainzFeedback struct {
	RecordingMBID string `json:"recording_mbid"`
	Score         int    `json:"score"`
}

type listenBrainzRecordingFeedback struct {
	RecordingMBID string `json:"recording_mbid"`
	Score         int    `json:"score"`
}

// LovedTracks retrieves loved tracks from ListenBrainz
func (lb *ListenBrainz) LovedTracks() ([]model.LovedTrack, error) {
	slog.Debug("Retrieving loved tracks", "source", "listenbrainz")

	var allTracks []model.LovedTrack
	offset := 0
	const pageSize = 100

	for {
		tracks, totalCount, err := lb.fetchLovedTracksPage(offset, pageSize)
		if err != nil {
			return nil, err
		}

		allTracks = append(allTracks, tracks...)

		if offset+len(tracks) >= totalCount {
			break
		}
		offset += len(tracks)
	}

	slog.Debug("Retrieved loved tracks", "count", len(allTracks), "source", "listenbrainz")
	return allTracks, nil
}

// fetchLovedTracksPage fetches a single page of loved tracks with retry logic
func (lb *ListenBrainz) fetchLovedTracksPage(offset, count int) ([]model.LovedTrack, int, error) {
	const maxRetries = 3
	url := fmt.Sprintf("https://api.listenbrainz.org/1/feedback/user/%s/get-feedback?score=1&offset=%d&count=%d", lb.Username, offset, count)

	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, 0, err
		}

		req.Header.Set("Authorization", fmt.Sprintf("Token %s", lb.Token))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			sleepDuration := lb.getSleepDuration(resp)
			slog.Warn("Rate limited (429), retrying", "attempt", attempt+1, "sleep_seconds", sleepDuration.Seconds(), "source", "listenbrainz")
			time.Sleep(sleepDuration)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, 0, fmt.Errorf("ListenBrainz API error: %s - %s", resp.Status, string(body))
		}

		var feedbackResp listenBrainzFeedbackResponse
		if err := json.NewDecoder(resp.Body).Decode(&feedbackResp); err != nil {
			return nil, 0, err
		}

		var tracks []model.LovedTrack
		for _, feedback := range feedbackResp.Feedback {
			tracks = append(tracks, model.LovedTrack{
				TrackMBID: feedback.RecordingMBID,
			})
		}

		time.Sleep(1 * time.Second)
		return tracks, feedbackResp.TotalCount, nil
	}

	return nil, 0, fmt.Errorf("ListenBrainz: max retries exceeded due to rate limiting")
}

// Love marks tracks as loved on ListenBrainz
func (lb *ListenBrainz) Love(tracks []model.LovedTrack) error {
	return lb.submitFeedback(tracks, 1)
}

// Unlove removes loved status from tracks on ListenBrainz
func (lb *ListenBrainz) Unlove(tracks []model.LovedTrack) error {
	return lb.submitFeedback(tracks, 0)
}

// submitFeedback submits feedback for tracks to ListenBrainz
func (lb *ListenBrainz) submitFeedback(tracks []model.LovedTrack, score int) error {
	if len(tracks) == 0 {
		return nil
	}

	for _, track := range tracks {
		if track.TrackMBID == "" {
			slog.Warn("Skipping track without MBID", "artist", track.Artist, "title", track.Track, "source", "listenbrainz")
			continue
		}

		feedback := listenBrainzRecordingFeedback{
			RecordingMBID: track.TrackMBID,
			Score:         score,
		}

		jsonData, err := json.Marshal(feedback)
		if err != nil {
			return err
		}

		if err := lb.submitSingleFeedback(jsonData); err != nil {
			return err
		}
	}

	return nil
}

// submitSingleFeedback submits a single feedback request, retrying on 429
func (lb *ListenBrainz) submitSingleFeedback(jsonData []byte) error {
	const maxRetries = 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest("POST", "https://api.listenbrainz.org/1/feedback/recording-feedback", bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", fmt.Sprintf("Token %s", lb.Token))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			sleepDuration := lb.getSleepDuration(resp)
			slog.Warn("Rate limited (429), retrying", "attempt", attempt+1, "sleep_seconds", sleepDuration.Seconds(), "source", "listenbrainz")
			time.Sleep(sleepDuration)
			continue
		}

		lb.handleRateLimit(resp)

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("ListenBrainz API error: %s - %s", resp.Status, string(body))
		}

		time.Sleep(1 * time.Second)
		return nil
	}

	return fmt.Errorf("ListenBrainz: max retries exceeded due to rate limiting")
}

// getSleepDuration calculates sleep duration from rate limit headers
func (lb *ListenBrainz) getSleepDuration(resp *http.Response) time.Duration {
	resetInStr := resp.Header.Get("X-RateLimit-Reset-In")
	if resetInStr != "" {
		if resetIn, err := strconv.Atoi(resetInStr); err == nil {
			return time.Duration(resetIn+5) * time.Second
		}
	}
	return 10 * time.Second
}

// handleRateLimit checks rate limit headers and sleeps if necessary
func (lb *ListenBrainz) handleRateLimit(resp *http.Response) {
	remainingStr := resp.Header.Get("X-RateLimit-Remaining")
	resetInStr := resp.Header.Get("X-RateLimit-Reset-In")
	limit := resp.Header.Get("X-RateLimit-Limit")

	if remainingStr == "" {
		return
	}

	remaining, err := strconv.Atoi(remainingStr)
	if err != nil {
		return
	}

	if remaining <= 1 {
		resetIn, err := strconv.Atoi(resetInStr)
		if err != nil {
			slog.Warn("Rate limit low but couldn't parse reset time", "remaining", remaining, "limit", limit, "source", "listenbrainz")
			return
		}

		sleepDuration := time.Duration(resetIn+5) * time.Second
		slog.Warn("Rate limit low, sleeping", "remaining", remaining, "limit", limit, "duration_seconds", sleepDuration.Seconds(), "source", "listenbrainz")
		time.Sleep(sleepDuration)
	}
}

var _ model.Source = &ListenBrainz{}
