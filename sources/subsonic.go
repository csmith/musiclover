package sources

import (
	"log/slog"
	"net/http"
	"strconv"
	"sync"

	"github.com/csmith/musiclover/matcher"
	"github.com/csmith/musiclover/model"
	"github.com/supersonic-app/go-subsonic/subsonic"
)

// Subsonic is a source that retrieves loved tracks from a Subsonic server
type Subsonic struct {
	BaseURL    string
	Username   string
	Password   string
	ClientName string

	mu          sync.Mutex
	client      *subsonic.Client
	artistMBIDs map[string]string
	albumMBIDs  map[string]string
	allSongs    []*subsonic.Child
}

// LovedTracks retrieves starred tracks from the Subsonic server
func (s *Subsonic) LovedTracks() ([]model.LovedTrack, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	slog.Debug("Retrieving starred tracks", "source", "subsonic")

	starred, err := client.GetStarred2(nil)
	if err != nil {
		return nil, err
	}

	slog.Debug("Retrieved starred tracks", "count", len(starred.Song), "source", "subsonic")

	// Get artist MBIDs
	artistMBIDs, err := s.getArtistMBIDs(client)
	if err != nil {
		return nil, err
	}

	// Get album MBIDs
	albumMBIDs, err := s.getAlbumMBIDs(client)
	if err != nil {
		return nil, err
	}

	// Convert to LovedTrack format
	tracks := s.childToLovedTrack(starred.Song, artistMBIDs, albumMBIDs)
	return tracks, nil
}

// Love stars tracks on the Subsonic server
func (s *Subsonic) Love(tracks []model.LovedTrack) error {
	songs, err := s.findSongs(tracks)
	if err != nil {
		return err
	}

	if len(songs) == 0 {
		return nil
	}

	client, err := s.getClient()
	if err != nil {
		return err
	}

	var songIDs []string
	for _, song := range songs {
		songIDs = append(songIDs, song.ID)
	}

	return client.Star(subsonic.StarParameters{
		SongIDs: songIDs,
	})
}

// Unlove unstars tracks on the Subsonic server
func (s *Subsonic) Unlove(tracks []model.LovedTrack) error {
	songs, err := s.findSongs(tracks)
	if err != nil {
		return err
	}

	if len(songs) == 0 {
		return nil
	}

	client, err := s.getClient()
	if err != nil {
		return err
	}

	var songIDs []string
	for _, song := range songs {
		songIDs = append(songIDs, song.ID)
	}

	return client.Unstar(subsonic.StarParameters{
		SongIDs: songIDs,
	})
}

// getClient lazily connects to the Subsonic server
func (s *Subsonic) getClient() (*subsonic.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		return s.client, nil
	}

	client := &subsonic.Client{
		Client:     http.DefaultClient,
		BaseUrl:    s.BaseURL,
		User:       s.Username,
		ClientName: s.ClientName,
	}

	if s.Password != "" {
		if err := client.Authenticate(s.Password); err != nil {
			return nil, err
		}
	}

	s.client = client
	return s.client, nil
}

// getArtistMBIDs retrieves all artist MBIDs from the Subsonic server
func (s *Subsonic) getArtistMBIDs(client *subsonic.Client) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.artistMBIDs != nil {
		return s.artistMBIDs, nil
	}

	slog.Debug("Retrieving artist MBIDs", "source", "subsonic")

	artists, err := client.GetArtists(nil)
	if err != nil {
		return nil, err
	}

	mbids := make(map[string]string)
	for _, index := range artists.Index {
		for _, artist := range index.Artist {
			if artist.MusicBrainzId != "" {
				mbids[artist.ID] = artist.MusicBrainzId
			}
		}
	}

	slog.Debug("Retrieved artist MBIDs", "count", len(mbids), "source", "subsonic")
	s.artistMBIDs = mbids
	return mbids, nil
}

// getAlbumMBIDs retrieves all album MBIDs from the Subsonic server
func (s *Subsonic) getAlbumMBIDs(client *subsonic.Client) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.albumMBIDs != nil {
		return s.albumMBIDs, nil
	}

	slog.Debug("Retrieving album MBIDs", "source", "subsonic")

	mbids := make(map[string]string)
	offset := 0
	const batchSize = 500

	for {
		albums, err := client.GetAlbumList("alphabeticalByName", map[string]string{
			"size":   strconv.Itoa(batchSize),
			"offset": strconv.Itoa(offset),
		})
		if err != nil {
			return nil, err
		}

		if len(albums) == 0 {
			break
		}

		for _, album := range albums {
			if album.MusicBrainzID != "" {
				mbids[album.ID] = album.MusicBrainzID
			}
		}

		if len(albums) < batchSize {
			break
		}
		offset += batchSize
	}

	slog.Debug("Retrieved album MBIDs", "count", len(mbids), "source", "subsonic")
	s.albumMBIDs = mbids
	return mbids, nil
}

// getAllSongs retrieves all songs from the Subsonic server
func (s *Subsonic) getAllSongs(client *subsonic.Client) ([]*subsonic.Child, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.allSongs != nil {
		return s.allSongs, nil
	}

	slog.Debug("Retrieving all songs", "source", "subsonic")

	var allSongs []*subsonic.Child
	offset := 0
	const batchSize = 500

	for {
		results, err := client.Search3("", map[string]string{
			"songCount":   strconv.Itoa(batchSize),
			"songOffset":  strconv.Itoa(offset),
			"artistCount": "0",
			"albumCount":  "0",
		})
		if err != nil {
			return nil, err
		}

		if len(results.Song) == 0 {
			break
		}

		allSongs = append(allSongs, results.Song...)

		if len(results.Song) < batchSize {
			break
		}
		offset += batchSize
	}

	slog.Debug("Retrieved all songs", "count", len(allSongs), "source", "subsonic")
	s.allSongs = allSongs
	return allSongs, nil
}

// childToLovedTrack converts Subsonic Children to LovedTracks
func (s *Subsonic) childToLovedTrack(songs []*subsonic.Child, artistMBIDs, albumMBIDs map[string]string) []model.LovedTrack {
	tracks := make([]model.LovedTrack, 0, len(songs))
	for _, song := range songs {
		tracks = append(tracks, model.LovedTrack{
			Track:      song.Title,
			Artist:     song.Artist,
			ArtistMBID: artistMBIDs[song.ArtistID],
			Album:      song.Album,
			AlbumMBID:  albumMBIDs[song.AlbumID],
			TrackMBID:  song.MusicBrainzID,
		})
	}
	return tracks
}

// findSongs searches for songs by metadata and returns matching Child records
func (s *Subsonic) findSongs(tracks []model.LovedTrack) ([]*subsonic.Child, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	allSongs, err := s.getAllSongs(client)
	if err != nil {
		return nil, err
	}

	artistMBIDs, err := s.getArtistMBIDs(client)
	if err != nil {
		return nil, err
	}

	albumMBIDs, err := s.getAlbumMBIDs(client)
	if err != nil {
		return nil, err
	}

	candidates := s.childToLovedTrack(allSongs, artistMBIDs, albumMBIDs)
	var songs []*subsonic.Child
	for _, track := range tracks {
		matchIndex := matcher.Find(candidates, track)
		if matchIndex == -1 {
			slog.Warn("Song not found", "artist", track.Artist, "track", track.Track, "source", "subsonic")
			continue
		}

		songs = append(songs, allSongs[matchIndex])
	}
	return songs, nil
}

var _ model.Source = &Subsonic{}
