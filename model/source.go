package model

// Source represents a music source that can provide loved tracks
type Source interface {
	LovedTracks() ([]LovedTrack, error)
	Love(tracks []LovedTrack) error
	Unlove(tracks []LovedTrack) error
}
