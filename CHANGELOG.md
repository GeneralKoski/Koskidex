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
- Every search hit carries its `score`. It was computed and then thrown away.
- `POST /indexes/{name}/documents/delete-batch` — deletes the documents whose
  ids are in the body (a JSON array of strings or integers) with one WAL record
  and one sync for the whole list, and reports how many were actually there.
  Unknown ids are not an error; an unusable id rejects the whole request with
  `400` and deletes nothing.
- `substring_match` index setting (off by default): a search also finds the
  documents with an indexed term that contains the whole query, and adds to
  their score, as Elasticsearch's `wildcard` `*query*` next to a `match` in a
  `bool` `should`. It is what Documentale's folder search does. A query with a
  space or punctuation finds nothing this way, as in Elasticsearch, where the
  wildcard compares the query with one term at a time. It scans the whole
  vocabulary rather than keeping an n-gram index: a folder index has a few
  thousand paths. `InvertedIndex.VocabularySize` reports the vocabulary size,
  and `scripts/compare -sottostringa` measures the cost on a corpus.
- `ids_only` on search (query parameter or POST body field): hits carry only
  `id` and `score`, without building documents and highlights, as
  Elasticsearch's `_source: false`.

### Changed
- `POST /indexes/{name}/documents` rejects the whole request with `400` when any
  document has a missing or unusable id, and says at which positions; nothing is
  added. `skipped` stays in the success response for compatibility and is
  always `0`.
- Search `limit` goes up to 10,000 instead of 1,000: a caller that filters its
  own database with the whole id set lost every result past the thousandth.
- `POST /indexes/{name}/documents` now reports `{added, skipped}` instead of a raw count.
- Document re-indexing on settings update happens in place (`Reindex`) instead of
  swapping the engine pointer.
- Backend Docker image runs as a non-root user.
- Optimized social share image (`og-image`) from ~5 MB to ~107 KB.

### Fixed
- List fields (`["a", "b"]`) were stored but never indexed, silently: a word
  found only in a list returned no hits. Lists are now indexed element by
  element, strings and numbers, with a gap of 100 positions between elements so
  that two elements never read as one phrase (as Elasticsearch's
  `position_increment_gap`). Without declared searchable fields, list fields are
  picked up automatically like string fields. A number on its own is indexed
  only in a declared searchable field.
- Integer ids were dropped with a `202` and `skipped: 1`. They are now accepted
  and stored as their canonical string (`7`, and `1000000` rather than `1e+06`).
- Documents without a valid `id` were silently dropped while the API reported success.
- Data race / lost documents when updating settings concurrently with document writes.
- Sitemap URLs are now XML-escaped (malformed XML / injection on `&`, `<`, `>`).
- robots.txt now advertises per-index `Sitemap:` directives.
- Internal errors no longer leak raw Go error strings to API clients.
- Index name is validated (length, path-traversal characters).
