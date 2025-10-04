# musiclover

`musiclover` is a tool to synchronise your "loved" music data between different
services. Take your starred tracks from subsonic, and love them on Last.fm, or
rate them up ListenBrainz. Or both! Or vice-versa!

## Running

musiclover is configured via command-line flags or environment vars.

| Flag                    | Env var                 | Details                                                                                 |
|-------------------------|-------------------------|-----------------------------------------------------------------------------------------|
| `subsonic-server`       | `SUBSONIC_SERVER`       | Base address of the subsonic server to connect to                                       |
| `subsonicusername`      | `SUBSONIC_USERNAME`     | Username for the subsonic server (can be blank if no auth is needed)                    |
| `subsonic-password`     | `SUBSONIC_PASSWORD`     | Password for the subsonic server (can be blank if no auth is needed)                    |
| `lastfm-key`            | `LASTFM_KEY`            | API key for Last.fm                                                                     |
| `lastfm-secret`         | `LASTFM_SECRET`         | API secret for Last.fm                                                                  |
| `lastfm-username`       | `LASTFM_USERNAME`       | Username for Last.fm                                                                    |
| `lastfm-password`       | `LASTFM_PASSWORD`       | Password for Last.fm                                                                    |
| `listenbrainz-token`    | `LISTENBRAINZ_TOKEN`    | User token for ListenBrainz                                                             |
| `listenbrainz-username` | `LISTENBRAINZ_USERNAME` | Username for ListenBrainz                                                               |
| `source`                | `SOURCE`                | Where to get the canonical list of lived tracks (subsonic, lastfm, or listenbrainz)     |
| `destinations`          | `DESTINATIONS`          | Where to update loved tracks (comma-separated, same options as `source`)                |
| `dry-run`               | `DRY_RUN`               | If true, changes to loved tracks will be printed and not actually performed             |
| `remove-other`          | `REMOVE_OTHER`          | If true, any loved tracks in the destination that are not in the source will be removed |
| `period`                | `PERIOD`                | If set, musiclover will run indefinitely, and perform updates once per this period      |

`source`, `destinations`, and the configuration for any of your sources and
destinations are mandatory options.

For Last.fm, you can get API credentials at https://www.last.fm/api/account/create.

For ListenBrainz, your user token from https://listenbrainz.org/settings/

## Caveats

Trying to match music between sources is a mess. ListenBrainz only supports
MusicBrainz IDs. Last.fm has very patchy support for MusicBrainz IDs.
musiclover will do its best to match tracks accurately, but it's not a sure
thing. In particular variations of tracks might get conflated (like "(Acoustic)"
versions). It should be close enough for recommendations and stats, but if
you want a more careful curation you probably want to do it by hand.

If matches aren't perfect, some tracks may end up being loved every time
musiclover runs (or added and removed if `remove-other` is enabled). Again,
this shouldn't cause much trouble from a stats/recommendations point of view,
but it may be annoying.

ListenBrainz is fairly heavily rate limited. musiclover will sleep for a second
after each request, and may sleep for longer if it still nears the rate limit.
Each love/unlove has to be done in a separate request, so syncing a large amount
(e.g. the first time you use the tool) will take many minutes.

## Example docker-compose file

To sync repeatedly, the recommended way to run is using Docker.

```yaml
services:
  musiclover:
    image: ghcr.io/csmith/musiclover
    environment:
      # Read loves from subsonic
      SOURCE: 'subsonic'
      # Write them to both last.fm and listenbrainz
      DESTINATIONS: 'lastfm,listenbrainz'
      # Get rid of any other likes on those platforms
      REMOVE_OTHER: 'true'
      # Synchronise every 12 hours
      PERIOD: '12h'
      
      # Configure the subsonic server. Username and password are optional if
      # you don't need to authenticate (e.g. if using proxy auth).
      # Only needed if you use subsonic as a source or destination
      SUBSONIC_SERVER: 'https://mymusic.example.com'
      SUBSONIC_USERNAME: 'acidburn'
      SUBSONIC_PASSWORD: 'H4ckTh3Pl@n3t'
      
      # Configure Last.fm. Only needed if you use lastfm as a source or
      # destination.
      LASTFM_KEY: 'abc123........'
      LASTFM_SECRET: 'def456........'
      LASTFM_USERNAME: 'acidburn'
      LASTFM_PASSWORD: 'H4ckTh3Pl@n3t'
      
      # Configure ListenBrainz. Only needed if you use listenbrainz as a source
      # or destination
      LISTENBRAINZ_TOKEN: 'abc123-def456-.......'
      LISTENBRAINZ_USERNAME: 'acidburn'
    restart: always
```

## Provenance

This project was primarily created with Claude Code, but with a strong guiding
hand. It's not "vibe coded", but an LLM was still the primary author of most
lines of code. I believe it meets the same sort of standards I'd aim for with
hand-crafted code, but some slop may slip through. I understand if you
prefer not to use LLM-created software, and welcome human-authored alternatives
(I just don't personally have the time/motivation to do so).