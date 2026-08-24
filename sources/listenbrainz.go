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
	Token     string
	Username  string
	UserAgent string
}

// httpClient is shared by all ListenBrainz requests so that connections to
// the API can be reused rather than reopened for every request.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
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

// fetchLovedTracksPage fetches a single page of loved tracks
func (lb *ListenBrainz) fetchLovedTracksPage(offset, count int) ([]model.LovedTrack, int, error) {
	url := fmt.Sprintf("https://api.listenbrainz.org/1/feedback/user/%s/get-feedback?score=1&offset=%d&count=%d", lb.Username, offset, count)

	body, err := lb.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}

	var feedbackResp listenBrainzFeedbackResponse
	if err := json.Unmarshal(body, &feedbackResp); err != nil {
		return nil, 0, err
	}

	var tracks []model.LovedTrack
	for _, feedback := range feedbackResp.Feedback {
		tracks = append(tracks, model.LovedTrack{
			TrackMBID: feedback.RecordingMBID,
		})
	}

	return tracks, feedbackResp.TotalCount, nil
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

// submitSingleFeedback submits a single feedback request
func (lb *ListenBrainz) submitSingleFeedback(jsonData []byte) error {
	_, err := lb.doRequest(http.MethodPost, "https://api.listenbrainz.org/1/feedback/recording-feedback", jsonData)
	return err
}

// doRequest performs a request against the ListenBrainz API, sleeping based
// on the rate limit headers of each response and retrying with backoff for
// as long as we are being rate limited. It returns the body of a successful
// response.
func (lb *ListenBrainz) doRequest(method, url string, body []byte) ([]byte, error) {
	const maxAttempts = 10

	backoff := 10 * time.Second
	for attempt := 1; ; attempt++ {
		var reqBody io.Reader
		if body != nil {
			reqBody = bytes.NewReader(body)
		}

		req, err := http.NewRequest(method, url, reqBody)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", fmt.Sprintf("Token %s", lb.Token))
		req.Header.Set("User-Agent", lb.userAgent())
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}

		switch {
		case resp.StatusCode == http.StatusOK:
			time.Sleep(nextRequestDelay(resp))
			return respBody, nil

		case resp.StatusCode == http.StatusTooManyRequests:
			if attempt >= maxAttempts {
				return nil, fmt.Errorf("ListenBrainz: still rate limited after %d attempts", attempt)
			}
			delay := retryDelay(resp, backoff)
			backoff = min(2*backoff, 5*time.Minute)
			slog.Warn("Rate limited (429), backing off", "attempt", attempt, "max_attempts", maxAttempts, "sleep_seconds", delay.Seconds(), "source", "listenbrainz")
			time.Sleep(delay)

		default:
			return nil, fmt.Errorf("ListenBrainz API error: %s - %s", resp.Status, string(respBody))
		}
	}
}

// userAgent returns the User-Agent to send, falling back to a default if
// none was configured.
func (lb *ListenBrainz) userAgent() string {
	if lb.UserAgent != "" {
		return lb.UserAgent
	}
	return "musiclover/dev"
}

// retryDelay returns how long to back off after a 429 response: the window
// reset time reported by the server if available, or the given fallback
// (which callers grow exponentially between attempts) otherwise.
func retryDelay(resp *http.Response, fallback time.Duration) time.Duration {
	if resetIn, ok := headerInt(resp, "X-RateLimit-Reset-In"); ok {
		return time.Duration(resetIn+5) * time.Second
	}
	return fallback
}

// nextRequestDelay returns how long to wait before the next request, based
// on the rate limit headers of the previous response. Normally this is a
// steady one request per second, but if the current window is nearly
// exhausted we wait for it to reset rather than risking a 429.
func nextRequestDelay(resp *http.Response) time.Duration {
	const baseDelay = 1 * time.Second

	if remaining, ok := headerInt(resp, "X-RateLimit-Remaining"); ok && remaining <= 2 {
		if resetIn, ok := headerInt(resp, "X-RateLimit-Reset-In"); ok {
			return max(time.Duration(resetIn+1)*time.Second, baseDelay)
		}
	}
	return baseDelay
}

// headerInt reads an integer-valued header, reporting whether it was present
// and parseable.
func headerInt(resp *http.Response, name string) (value int, ok bool) {
	value, err := strconv.Atoi(resp.Header.Get(name))
	return value, err == nil
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

var _ model.Source = &ListenBrainz{}
