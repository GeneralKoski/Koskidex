# Changelog

All notable changes to this project are documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- `GET /indexes/{name}/documents` — paginated document listing (`limit`, `offset`).
- Configurable CORS origin via `--cors-origin` / `KOSKIDEX_CORS_ORIGIN`.
- Environment-variable configuration for all flags (flags take precedence).
- Graceful shutdown on `SIGINT`/`SIGTERM` with connection draining.
- HTTP server read/write/idle timeouts (Slowloris protection).
- Request body size limits on all write endpoints.
- Per-route SEO meta tags (title/description/OG/canonical) in the web app.
- `manifest.json`, `theme-color`, `apple-touch-icon`; `/docs` added to the sitemap.
- nginx: gzip, security headers, and cache-control tuning.
- Backend tests for cache, filters, sitemap escaping, robots, and document listing.
- Frontend tests (Vitest + Testing Library).
- `LICENSE`, `SECURITY.md`, `CONTRIBUTING.md`.

### Changed
- `POST /indexes/{name}/documents` now reports `{added, skipped}` instead of a raw count.
- Document re-indexing on settings update happens in place (`Reindex`) instead of
  swapping the engine pointer.
- Backend Docker image runs as a non-root user.
- Optimized social share image (`og-image`) from ~5 MB to ~107 KB.

### Fixed
- Documents without a valid `id` were silently dropped while the API reported success.
- Data race / lost documents when updating settings concurrently with document writes.
- Sitemap URLs are now XML-escaped (malformed XML / injection on `&`, `<`, `>`).
- robots.txt now advertises per-index `Sitemap:` directives.
- Internal errors no longer leak raw Go error strings to API clients.
- Index name is validated (length, path-traversal characters).
