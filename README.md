<p align="center">
  <img src="web/public/og-image.png" alt="Koskidex" width="100%" />
</p>

<h1 align="center">Koskidex</h1>

<p align="center">
  A lightning-fast, self-hosted full-text search engine written in Go.<br/>
  Sub-15MB binary. Sub-5ms queries. Zero dependencies. Drop it into any stack.
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> •
  <a href="#features">Features</a> •
  <a href="#api-reference">API</a> •
  <a href="#integrations">Integrations</a> •
  <a href="#architecture">Architecture</a>
</p>

---

## Quick Start

```bash
git clone https://github.com/GeneralKoski/Koskidex.git
cd Koskidex
docker compose up -d
```

That's it. Two services are now running:

| Service | URL | Description |
|---------|-----|-------------|
| **API** | `http://localhost:7700` | Search engine REST API |
| **Web UI** | `http://localhost:8080` | Interactive playground |

```bash
# Create an index
curl -X POST http://localhost:7700/indexes \
  -H 'Content-Type: application/json' \
  -d '{"name": "movies"}'

# Add documents
curl -X POST http://localhost:7700/indexes/movies/documents \
  -H 'Content-Type: application/json' \
  -d '[
  {"id": "1", "title": "The Matrix", "genre": "Sci-Fi", "year": 1999},
  {"id": "2", "title": "Inception", "genre": "Action", "year": 2010},
  {"id": "3", "title": "Interstellar", "genre": "Sci-Fi", "year": 2014}
]'

# Search — try a typo!
curl "http://localhost:7700/indexes/movies/search?q=matrx"
```

---

## Features

| | Feature | Details |
|---|---------|---------|
| **Performance** | Sub-5ms response times on reasonable datasets |
| **Lightweight** | Binary <15MB, RAM <20MB idle, zero runtime dependencies |
| **Typo tolerance** | Damerau-Levenshtein fuzzy matching with dynamic fuzziness (`0`, `1`, `2`, `AUTO`) |
| **Smart ranking** | Multi-factor pipeline: exactness, typo count, field weight, term frequency |
| **Field weighting** | Boost specific fields via `field_weights` settings (e.g. `name: 5`, `description: 1`) |
| **Explicit sorting** | Sort results by any field: `sort=price:asc,rating:desc` |
| **Faceted search** | Aggregate counts per field: `facets=genre,category` |
| **Geospatial** | Haversine distance filtering: `filter=distance(_geo,45.46,9.19)<50000` |
| **Vector search** | Cosine similarity on `_vector` fields, hybrid mode with full-text scoring |
| **WAL** | Write-Ahead Log for crash recovery — zero data loss on unexpected shutdowns |
| **Search operators** | `AND` (default), `OR`, `NOT` (prefix `-`) |
| **Field filters** | `filter=genre=Sci-Fi,year>2000` with `=`, `!=`, `>`, `<`, `>=`, `<=` |
| **Pagination** | `limit` and `offset` params, `total_hits` in response |
| **Multi-index** | Create and manage independent indexes with their own settings |
| **Schemaless** | Index any JSON object — only requires a unique `id` field |
| **Synonyms** | Configure per-index synonym mappings |
| **Auth** | Optional API key via `--api-key` flag, Bearer token auth |
| **Rate limiting** | Per-IP token bucket via `--rate-limit` flag |
| **TLS** | Native HTTPS via `--tls-cert` and `--tls-key` flags |
| **LRU cache** | Query results cached, auto-invalidated on writes |
| **Docker ready** | Multi-stage Alpine build, healthcheck included |
| **Client libraries** | PHP/Laravel, Node.js, Python — ready to copy |

---

## API Reference

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/indexes` | Create an index |
| `GET` | `/indexes` | List all indexes |
| `GET` | `/indexes/{name}` | Get index info |
| `DELETE` | `/indexes/{name}` | Delete index |
| `POST` | `/indexes/{name}/documents` | Add documents (single, array, or file upload) |
| `GET` | `/indexes/{name}/documents/{id}` | Get document by ID |
| `DELETE` | `/indexes/{name}/documents/{id}` | Delete document |
| `GET` | `/indexes/{name}/search?q=` | Full-text search |
| `POST` | `/indexes/{name}/search` | Full-text search (POST body) |
| `GET` | `/indexes/{name}/settings` | Get index settings |
| `PUT` | `/indexes/{name}/settings` | Update settings (synonyms, stop words, etc.) |
| `GET` | `/health` | Health check (no auth required) |

### Search

```bash
# Basic search
curl "http://localhost:7700/indexes/movies/search?q=matrx"

# Pagination
curl "http://localhost:7700/indexes/movies/search?q=matrix&limit=10&offset=0"

# Field filters
curl "http://localhost:7700/indexes/movies/search?q=matrix&filter=genre=Sci-Fi"
curl "http://localhost:7700/indexes/movies/search?q=movie&filter=year>2000,genre=Action"

# Dynamic fuzziness (0 = exact, 1 = one typo, 2 = two typos, AUTO = adaptive)
curl "http://localhost:7700/indexes/movies/search?q=matrx&fuzziness=AUTO"
curl "http://localhost:7700/indexes/movies/search?q=matrx&fuzziness=0"  # no results (exact only)

# Explicit sorting
curl "http://localhost:7700/indexes/movies/search?q=nolan&sort=year:desc"
curl "http://localhost:7700/indexes/products/search?q=apple&sort=price:asc,rating:desc"

# Faceted search (returns aggregated counts per field)
curl "http://localhost:7700/indexes/movies/search?q=the&facets=genre,director"

# Geospatial filtering (distance in meters, Haversine)
curl "http://localhost:7700/indexes/places/search?q=pizza&filter=distance(_geo,45.4642,9.1900)<5000"

# OR operator
curl "http://localhost:7700/indexes/movies/search?q=matrix OR inception"

# NOT operator
curl "http://localhost:7700/indexes/movies/search?q=movie -horror"

# POST search (for complex queries or vector search)
curl -X POST http://localhost:7700/indexes/movies/search \
  -H 'Content-Type: application/json' \
  -d '{
  "q": "matrix",
  "fuzziness": "AUTO",
  "sort": "year:desc",
  "facets": "genre",
  "filter": "year>2000",
  "limit": 10,
  "offset": 0,
  "vector": [0.1, 0.2, 0.3]
}'
```

### Documents

```bash
# Add array of documents
curl -X POST http://localhost:7700/indexes/movies/documents \
  -H 'Content-Type: application/json' \
  -d '[{"id": "1", "title": "Inception"}, {"id": "2", "title": "Tenet"}]'

# Add single document
curl -X POST http://localhost:7700/indexes/movies/documents \
  -H 'Content-Type: application/json' \
  -d '{"id": "3", "title": "The Matrix"}'

# Upload JSON file
curl -X POST http://localhost:7700/indexes/movies/documents \
  -F "file=@movies.json"
```

### Settings

```bash
curl -X PUT http://localhost:7700/indexes/movies/settings \
  -H 'Content-Type: application/json' \
  -d '{
  "synonyms": {"iphone": ["apple phone", "smartphone"]},
  "searchable_fields": ["title", "description"],
  "displayed_fields": ["title", "price"],
  "stop_words": ["the", "a", "is"],
  "field_weights": {"title": 5, "description": 1},
  "typo_tolerance": {
    "enabled": true,
    "min_word_length_one_typo": 4,
    "min_word_length_two_typos": 8
  }
}'
```

### Authentication

```bash
# Start with API key
./koskidex --api-key your-secret-key

# All requests require the Bearer token
curl -H "Authorization: Bearer your-secret-key" \
  http://localhost:7700/indexes

# /health is always public
curl http://localhost:7700/health
```

---

## Integrations

Client libraries are in the `examples/` directory. Clone the repo inside your project and copy what you need.

### Docker Compose (any stack)

```yaml
services:
  koskidex:
    build: ./koskidex
    ports:
      - "${KOSKIDEX_PORT:-7700}:7700"
    volumes:
      - koskidex_data:/data
    command: ["/app/koskidex", "--port", "7700", "--data-dir", "/data", "--api-key", "${KOSKIDEX_API_KEY}"]

volumes:
  koskidex_data:
```

```env
KOSKIDEX_HOST=http://koskidex:7700
KOSKIDEX_API_KEY=your-secret-key
```

### PHP / Laravel

Copy the integration files and add the `Searchable` trait to your models — they auto-sync on every `save()`, `update()`, and `delete()`.

```bash
cp koskidex/examples/laravel/app/Services/KoskidexClient.php app/Services/
cp koskidex/examples/laravel/app/Traits/Searchable.php app/Traits/
cp koskidex/examples/laravel/config/koskidex.php config/
```

```php
use App\Traits\Searchable;

class Movie extends Model
{
    use Searchable;
}

// Auto-synced on create/update/delete
Movie::create(['title' => 'The Matrix', 'genre' => 'Sci-Fi']);

// Search
$results = Movie::koskidexSearch('matrx');
```

### Node.js

```bash
npm install ./koskidex/examples/nodejs
```

```js
const KoskidexClient = require('koskidex-node');
const client = new KoskidexClient('http://localhost:7700', 'your-api-key');

await client.createIndex('movies');
await client.addDocuments('movies', [{ id: '1', title: 'The Matrix' }]);
const results = await client.search('movies', 'matrx');
```

### Python

```bash
pip install ./koskidex/examples/python
```

```python
from koskidex_client import KoskidexClient

client = KoskidexClient('http://localhost:7700', api_key='your-api-key')

client.create_index('movies')
client.add_documents('movies', [{'id': '1', 'title': 'The Matrix'}])
results = client.search('movies', 'matrx')
```

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│                   HTTP Server                    │
│  CORS → Rate Limiter → Auth → Router            │
├─────────────────────────────────────────────────┤
│                 Index Manager                    │
│  Create / Delete / List indexes                  │
│  Cache invalidation on writes                    │
├──────────────┬──────────────┬───────────────────┤
│   Tokenizer  │    Engine    │    Persistence    │
│  Normalize   │  Inverted    │  GOB snapshots    │
│  Stop words  │  Index       │  Write-Ahead Log  │
│  Split       │  Bigram      │  Crash recovery   │
│              │  Prefix      │                    │
├──────────────┼──────────────┤                    │
│    Ranker    │   Filters    │                    │
│  Fuzzy match │  Field ops   │                    │
│  Field boost │  Geo/Haver.  │                    │
│  Vector/Cos. │  Facets      │                    │
│  Sorting     │  OR / NOT    │                    │
├──────────────┴──────────────┤                    │
│          LRU Cache          │                    │
│  1024 entries, prefix       │                    │
│  invalidation               │                    │
└─────────────────────────────┴───────────────────┘
```

### Tech Stack

| Layer | Technology |
|-------|------------|
| **Engine** | Go 1.23, zero external dependencies |
| **Search** | Inverted index, Damerau-Levenshtein, bigram prefix, cosine similarity |
| **Ranking** | Field weighting, dynamic fuzziness, hybrid text+vector scoring |
| **Storage** | GOB snapshots + Write-Ahead Log (WAL), debounced persistence |
| **Cache** | LRU (doubly-linked list + map), auto-invalidation |
| **API** | net/http, token bucket rate limiter |
| **Frontend** | React 19, TypeScript, Vite 6, Tailwind CSS 4 |
| **i18n** | English, Italian |
| **Deploy** | Docker multi-stage Alpine, docker-compose |

---

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `7700` | HTTP port |
| `--data-dir` | `./data` | Data directory for persistence |
| `--api-key` | _(empty)_ | API key for authentication |
| `--rate-limit` | `0` | Max requests/sec per IP (0 = disabled) |
| `--tls-cert` | _(empty)_ | Path to TLS certificate |
| `--tls-key` | _(empty)_ | Path to TLS key |
| `--log-level` | `info` | Log level: debug, info, warn, error |
| `--version` | | Print version and exit |

---

## License

MIT
