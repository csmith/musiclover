package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/csmith/envflag/v2"
	"github.com/csmith/musiclover/matcher"
	"github.com/csmith/musiclover/model"
	"github.com/csmith/musiclover/sources"
	"github.com/csmith/slogflags"
)

var (
	subsonicServer   = flag.String("subsonic-server", "", "Subsonic server base address")
	subsonicUsername = flag.String("subsonic-username", "", "Subsonic username")
	subsonicPassword = flag.String("subsonic-password", "", "Subsonic password")

	lastfmKey      = flag.String("lastfm-key", "", "Last.fm API key")
	lastfmSecret   = flag.String("lastfm-secret", "", "Last.fm API secret")
	lastfmUsername = flag.String("lastfm-username", "", "Last.fm username")
	lastfmPassword = flag.String("lastfm-password", "", "Last.fm password")

	listenbrainzToken    = flag.String("listenbrainz-token", "", "ListenBrainz token")
	listenbrainzUsername = flag.String("listenbrainz-username", "", "ListenBrainz username")

	source       = flag.String("source", "", "Source of truth for loved tracks")
	destinations = flag.String("destinations", "", "Comma-separated list of destinations to sync loved tracks to")
	dryRun       = flag.Bool("dry-run", false, "Don't actually do anything, just print the differences in loves")
	removeOther  = flag.Bool("remove-other", false, "Remove tracks that were loved but aren't in the source")
	period       = flag.Duration("period", 0, "Length of time between each update. If zero, will update once and exit.")

	availableSources map[string]model.Source
)

func main() {
	envflag.Parse()
	_ = slogflags.Logger(slogflags.WithSetDefault(true))

	initialiseSources()

	src, err := selectedSource()
	if err != nil {
		slog.Error("Failed to get source", "error", err)
		os.Exit(1)
	}

	dests, err := selectedDestinations()
	if err != nil {
		slog.Error("Failed to get destinations", "error", err)
		os.Exit(1)
	}

	if period.Minutes() < 1 {
		slog.Debug("Period is less than 1 minute, doing a one-shot run")
		run(src, dests)
	} else {
		for {
			run(src, dests)
			slog.Info("Sleeping until next update", "period", period)
			time.Sleep(*period)
		}
	}
}

func run(src model.Source, dests map[string]model.Source) {
	sourceTracks, err := src.LovedTracks()
	if err != nil {
		slog.Error("Failed to get loved tracks from source", "source", *source, "error", err)
		os.Exit(1)
	}

	for name, dest := range dests {
		if err := sync(name, dest, sourceTracks); err != nil {
			slog.Error("Failed to sync to destination", "destination", name, "error", err)
			os.Exit(1)
		}
	}
}

func initialiseSources() {
	availableSources = make(map[string]model.Source)

	if *subsonicServer != "" {
		availableSources["subsonic"] = &sources.Subsonic{
			BaseURL:    *subsonicServer,
			Username:   *subsonicUsername,
			Password:   *subsonicPassword,
			ClientName: "musiclover",
		}
	}

	if *lastfmKey != "" && *lastfmSecret != "" {
		availableSources["lastfm"] = &sources.Lastfm{
			APIKey:   *lastfmKey,
			Secret:   *lastfmSecret,
			Username: *lastfmUsername,
			Password: *lastfmPassword,
		}
	}

	if *listenbrainzToken != "" {
		availableSources["listenbrainz"] = &sources.ListenBrainz{
			Token:    *listenbrainzToken,
			Username: *listenbrainzUsername,
		}
	}
}

func selectedSource() (model.Source, error) {
	if *source == "" {
		return nil, fmt.Errorf("source must be specified")
	}

	src, ok := availableSources[*source]
	if !ok {
		return nil, fmt.Errorf("source not configured or invalid: %s", *source)
	}

	return src, nil
}

func selectedDestinations() (map[string]model.Source, error) {
	if *destinations == "" {
		return nil, fmt.Errorf("destinations must be specified")
	}

	destNames := strings.Split(*destinations, ",")
	for i := range destNames {
		destNames[i] = strings.TrimSpace(destNames[i])
	}

	dests := make(map[string]model.Source)

	for _, destName := range destNames {
		if destName == *source {
			slog.Info("Skipping destination that is the same as source", "destination", destName)
			continue
		}

		dest, ok := availableSources[destName]
		if !ok {
			return nil, fmt.Errorf("destination not configured or invalid: %s", destName)
		}

		dests[destName] = dest
	}

	return dests, nil
}

func sync(name string, dest model.Source, sourceTracks []model.LovedTrack) error {
	destTracks, err := dest.LovedTracks()
	if err != nil {
		return fmt.Errorf("failed to get loved tracks: %w", err)
	}

	segment := matcher.Segment(sourceTracks, destTracks)

	toLove := segment.Missing
	var toUnlove []model.LovedTrack
	if *removeOther {
		toUnlove = segment.Extra
	}

	slog.Info(
		"Calculated differences",
		"destination", name,
		"destination_count", len(destTracks),
		"to_add", len(toLove),
		"to_remove", len(toUnlove),
		"source", *source,
		"source_count", len(sourceTracks),
	)

	if *dryRun {
		for _, track := range toLove {
			slog.Info("Would love", "artist", track.Artist, "title", track.Track, "mbid", track.TrackMBID, "destination", name)
		}
		if *removeOther {
			for _, track := range toUnlove {
				slog.Info("Would unlove", "artist", track.Artist, "title", track.Track, "mbid", track.TrackMBID, "destination", name)
			}
		}
		return nil
	}

	if err := dest.Love(toLove); err != nil {
		return fmt.Errorf("failed to love tracks: %w", err)
	}

	if *removeOther {
		if err := dest.Unlove(toUnlove); err != nil {
			return fmt.Errorf("failed to unlove tracks: %w", err)
		}
	}

	return nil
}
