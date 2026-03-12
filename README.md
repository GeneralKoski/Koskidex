# Koskidex ⚡️

A lightning-fast, ultra-lightweight (<15MB), self-hosted full-text search engine written in Go. Purely RESTful, zero runtime dependencies, and built for simplicity.

![Koskidex OG](web/public/og-image.png)

## 🚀 Key Features

- **🚀 Performance**: Sub-5ms search response times on reasonable datasets.
- **📦 Lightweight**: Binary size under 15MB. RAM usage under 20MB idle.
- **🧠 Intelligent**: Handles typos via fuzzy matching (Damerau-Levenshtein) and smart ranking.
- **🔌 Integration**: Purely RESTful JSON API. Speaks anything.
- **Multilingual**: Built-in support for multiple languages (In-app translations for EN/IT).
- **🛡️ Multi-tenancy**: Create and manage multiple indexes independently.
- **🐳 Docker Ready**: Deploy anywhere with one command.

## 🛠️ Quick Start

### Run with Docker (Recommended)

```bash
docker-compose up --build
```

The stack includes:
- **Search API**: `http://localhost:7700` (Backend engine)
- **Web UI**: `http://localhost:8080` (A modern React playground)
> [!TIP]
> Use the "Load Movies" or "Load Products" buttons in the web UI to instantly populate the engine and start testing queries!

### 🔑 API Key & Security
By default, the engine starts without an API key for ease of use. If you want to secure your instance:
1. Start with `--api-key your-secret-token`.
2. All requests must then include headers: `Authorization: Bearer your-secret-token` or `X-API-Key: your-secret-token`.

### ⚓️ Port Configuration
The internal port is `7700`. You can map it to any host port in `docker-compose.yml`:
```yaml
services:
  backend:
    ports:
      - "8000:7700" # Maps host 8000 to internal 7700
```

## 📖 API Documentation

### Create an Index
```bash
curl -X POST http://localhost:7700/indexes -d '{"name": "my-index"}'
```

### Add Documents
```bash
curl -X POST http://localhost:7700/indexes/my-index/documents \
  -H 'Content-Type: application/json' -d '[
  {"id": "doc1", "title": "Inception", "genre": "Sci-Fi"},
  {"id": "doc2", "title": "The Matrix", "genre": "Sci-Fi"}
]'
```

### Search
```bash
curl "http://localhost:7700/indexes/my-index/search?q=incption"
```

### Delete a Document
```bash
curl -X DELETE http://localhost:7700/indexes/my-index/documents/doc1
```

### Manage Settings
```bash
# Get current settings
curl http://localhost:7700/indexes/my-index/settings

# Update settings (Synonyms, Ranking, etc.)
curl -X PUT http://localhost:7700/indexes/my-index/settings -d '{
  "synonyms": {"iphone": ["apple telefon", "smartphone"]},
  "searchable_fields": ["title", "description"],
  "displayed_fields": ["title", "price"]
}'
```

## 🏗️ Technical Architecture

- **Go**: High-performance backend logic.
- **Inverted Index**: Core data structure for efficient lookups.
- **Custom Tokenizer**: Normalization, stop-words, and splitting.
- **Ranker Pipeline**: Scoring matching documents based on relevance, typos, and frequency.
- **Vite/React/TS/Tailwind**: Modern frontend stack for the presentation layer.

## 🤝 Integrations

Check out the `examples/` directory for ready-to-use clients in:
- **PHP / Laravel** (with automatic model syncing)
- **Node.js**
- **Python**

---
Made with ❤️ by the Koskidex Team.
