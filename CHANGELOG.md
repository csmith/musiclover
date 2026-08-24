# Changelog

## 1.1.0 - 2026-08-24

- Handle ListenBrainz rate limiting more gracefully: requests are paced based
  on the API's rate limit headers, and rate limited requests are retried with
  increasing backoff instead of giving up after three attempts.
- When running periodically, a failed sync no longer exits the process: it is
  logged and retried at the next update, rather than being restarted straight
  back into the same rate limit. A failure syncing to one destination no
  longer stops the others being attempted.
- Send a `User-Agent` identifying musiclover to ListenBrainz, reuse HTTP
  connections, and log the running version at startup. Docker images now
  report the git commit they were built from.

## 1.0.0 - 2025-10-04

_Initial release._