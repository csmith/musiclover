package sources

import (
	"log/slog"
	"sync"

	"github.com/csmith/musiclover/model"
	"github.com/twoscott/gobble-fm/lastfm"
	"github.com/twoscott/gobble-fm/session"
)

// Lastfm is a source that retrieves loved tracks from Last.fm
type Lastfm struct {
	APIKey   string
	Secret   string
	Username string
	Password string

	mu     sync.Mutex
	client *session.Client
}

// LovedTracks retrieves loved tracks from Last.fm
func (l *Lastfm) LovedTracks() ([]model.LovedTrack, error) {
	client, err := l.getClient()
	if err != nil {
		return nil, err
	}

	slog.Debug("Retrieving loved tracks", "source", "lastfm")

	var tracks []model.LovedTrack
	page := uint(1)

	for {
		lovedTracks, err := client.User.LovedTracks(lastfm.LovedTracksParams{
			User:  l.Username,
			Page:  page,
			Limit: 200,
		})
		if err != nil {
			return nil, err
		}

		for _, track := range lovedTracks.Tracks {
			tracks = append(tracks, model.LovedTrack{
				Track:      track.Title,
				TrackMBID:  track.MBID,
				Artist:     track.Artist.Name,
				ArtistMBID: track.Artist.MBID,
			})
		}

		if page >= uint(lovedTracks.TotalPages) {
			break
		}
		page++
	}

	slog.Debug("Retrieved loved tracks", "count", len(tracks), "source", "lastfm")
	return tracks, nil
}

// Love marks tracks as loved on Last.fm
func (l *Lastfm) Love(tracks []model.LovedTrack) error {
	if len(tracks) == 0 {
		return nil
	}

	client, err := l.getClient()
	if err != nil {
		return err
	}

	for _, track := range tracks {
		var artist, title string

		if track.TrackMBID != "" {
			trackInfo, err := client.Track.InfoByMBID(lastfm.TrackInfoMBIDParams{
				MBID: track.TrackMBID,
			})
			if err == nil {
				artist = trackInfo.Artist.Name
				title = trackInfo.Title
			}
		}

		if artist == "" || title == "" {
			slog.Info("Couldn't use MBID to find Last.fm track, falling back to blind artist/title", "mbid", track.TrackMBID, "artist", track.Artist, "title", track.Track)
			artist = track.Artist
			title = track.Track
		}

		if err := client.Track.Love(artist, title); err != nil {
			return err
		}
	}

	return nil
}

// Unlove removes loved status from tracks on Last.fm
func (l *Lastfm) Unlove(tracks []model.LovedTrack) error {
	if len(tracks) == 0 {
		return nil
	}

	client, err := l.getClient()
	if err != nil {
		return err
	}

	for _, track := range tracks {
		if err := client.Track.Unlove(track.Artist, track.Track); err != nil {
			return err
		}
	}

	return nil
}

// getClient lazily connects to Last.fm
func (l *Lastfm) getClient() (*session.Client, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.client != nil {
		return l.client, nil
	}

	client := session.NewClient(l.APIKey, l.Secret)
	if err := client.Login(l.Username, l.Password); err != nil {
		return nil, err
	}

	l.client = client
	return l.client, nil
}

var _ model.Source = &Lastfm{}
