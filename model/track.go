package model

// LovedTrack represents a loved/starred track with metadata
type LovedTrack struct {
	Track      string
	Artist     string
	Album      string
	TrackMBID  string
	ArtistMBID string
	AlbumMBID  string
}
