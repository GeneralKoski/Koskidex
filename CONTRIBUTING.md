# Contributing to Koskidex

Thanks for your interest in improving Koskidex! This guide covers how to set up
your environment and submit changes.

## Development Setup

### Backend (Go)

Requires Go 1.23+.

```bash
make build      # build the binary
make test       # run the test suite
make lint       # run go vet + gofmt check
make bench      # run benchmarks
```

### Frontend (React + Vite)

Requires Node.js 20+.

```bash
cd web
npm install
npm run dev     # start dev server
npm run lint    # run eslint
npm run build   # production build
```

### Full stack

```bash
docker compose up --build
```

## Guidelines

- Keep the backend dependency-free where possible (the engine intentionally
  relies only on the standard library and `golang.org/x/text`).
- Format Go code with `gofmt` and pass `go vet` before committing.
- Add or update tests for any behavioral change.
- Keep changes focused — one logical change per pull request.

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/) style:

```
feat(engine): add geospatial radius filter
fix(server): sanitize internal error responses
docs(readme): document error response format
```

Write commit messages in English.

## Pull Requests

1. Fork and create a feature branch off `main`.
2. Make your changes with tests.
3. Ensure CI passes (`make test`, `make lint`, frontend build).
4. Open a PR describing the change and its motivation.
