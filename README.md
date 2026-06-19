# Islamic Article RAG

A retrieval-augmented generation (RAG) backend for answering questions about Indonesian Islamic articles. Articles are ingested from Markdown files, embedded and indexed in [Qdrant](https://qdrant.tech/), stored in MongoDB, and served through an HTTP API that combines hybrid search with an LLM. Redis enforces per-IP rate limits on the ask endpoint.

## Features

- **Ingestion pipeline** — reads `.md` files, splits them into overlapping paragraph windows, extracts Quran references (`(QS. Surah: verse)`), stores full text in MongoDB, and indexes chunk vectors in Qdrant.
- **Hybrid retrieval** — dense semantic search (Ollama `bge-m3`, 1024-dim) fused with sparse BM25 keyword search via Reciprocal Rank Fusion (RRF) in Qdrant.
- **Flexible LLM backends** — Ollama (default), Google Gemini, or Groq.
- **Structured citations** — the LLM is prompted to quote Quran and Hadith using XML-style tags (`<quran>`, `<hadith>`). (TODO: cross check to validation api)
- **Configurable context** — feed the LLM either retrieved chunks or full source articles.
- **MongoDB persistence** — full article text and QnA audit records (question, answer, sources, model metadata).
- **Per-IP rate limiting** — Redis-backed sliding window on `/api/v1/ask` to protect the API from abuse.

## Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────┐
│  .md files  │────▶│  ingest (CLI)    │────▶│   Qdrant    │
└─────────────┘     │  embed + chunk   │     │  collections│
                    └────────┬─────────┘     └──────┬──────┘
                             │                      │
                             ▼                      │
                    ┌──────────────────┐            │
                    │     MongoDB      │            │
                    │ articles + qna   │            │
                    └────────┬─────────┘            │
                             │                      │
┌─────────────┐     ┌────────┴─────────┐            │
│   Client    │────▶│  api (HTTP)      │◀───────────┘
└─────────────┘     │  hybrid search   │
                    │  + LLM answer    │
                    └────────┬─────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │      Redis       │
                    │  IP rate limits  │
                    └──────────────────┘
```


| Collection                 | Purpose                                                                          |
| -------------------------- | -------------------------------------------------------------------------------- |
| `indonesian_articles`      | Chunk vectors (dense + sparse) for hybrid search                                 |
| `indonesian_articles_full` | Legacy full-article vectors in Qdrant (optional; full text is stored in MongoDB) |



| MongoDB collection | Purpose                                                                |
| ------------------ | ---------------------------------------------------------------------- |
| `articles`         | Full article text, keyed by ID with a unique index on `url`            |
| `qna_records`      | QnA audit log (question, answer, LLM metadata, source chunks/articles) |


## Prerequisites

- [Go](https://go.dev/) 1.25+
- [Docker](https://www.docker.com/) (for Qdrant and Redis; MongoDB service is available in `docker-compose.yml` but commented out by default)
- [Ollama](https://ollama.com/) (default embedding and LLM provider)
- MongoDB instance reachable at `MONGO_URI` (required for ingest and API)

Pull the required Ollama models:

```bash
ollama pull bge-m3
ollama pull qwen2.5:7b
```

## Quick start

### 1. Clone and configure

```bash
cp .env.example .env
```

Edit `.env` as needed. The defaults assume Qdrant on `localhost:6333`, MongoDB on `localhost:27017`, Redis on `localhost:6379`, and Ollama on `localhost:11434`.

### 2. Start infrastructure

**Local (Docker)**

```bash
docker compose up -d
```

This starts:

- **Qdrant** — vector database; dashboard at [http://localhost:6333/dashboard](http://localhost:6333/dashboard)
- **qdrant-init** — creates the `indonesian_articles` collection from `config/qdrant/indonesian_articles.json` if missing
- **Redis** — used by the API for per-IP rate limiting (`6379`)

MongoDB is defined in `docker-compose.yml` but commented out. Either uncomment the `mongo` service and run `docker compose up -d mongo`, or point `MONGO_URI` at an external MongoDB instance.

**Qdrant Cloud**

Set your cluster host and API key in `.env`:

```env
QDRANT_HOST=<cluster-id>.gcp.cloud.qdrant.io
QDRANT_API_KEY=<your-api-key>
QDRANT_GRPC_PORT=6334
```

Then bootstrap the collection (requires `curl`):

```bash
set -a && source .env && set +a
./scripts/qdrant-init-collection.sh
```

On Windows Git Bash, the same `source .env` line works if your shell supports it; otherwise export `QDRANT_HOST` and `QDRANT_API_KEY` manually before running the script.

**Run the script manually (any environment)**

The script is idempotent — it skips creation if the collection already exists.


| Variable           | Default                                  | Description                                                             |
| ------------------ | ---------------------------------------- | ----------------------------------------------------------------------- |
| `QDRANT_HOST`      | —                                        | Cluster hostname (no `https://`); used when `QDRANT_URL` is unset       |
| `QDRANT_URL`       | `http://qdrant:6333`                     | Full REST base URL; overrides host-based resolution                     |
| `QDRANT_API_KEY`   | —                                        | Required for Qdrant Cloud; enables HTTPS and sends the `api-key` header |
| `QDRANT_REST_PORT` | `6333`                                   | REST port when building URL from `QDRANT_HOST`                          |
| `COLLECTION`       | `indonesian_articles`                    | Collection name to create                                               |
| `CONFIG_FILE`      | `config/qdrant/indonesian_articles.json` | Collection schema JSON                                                  |


Examples:

```bash
# Local Qdrant (no API key)
QDRANT_HOST=localhost ./scripts/qdrant-init-collection.sh

# Explicit REST URL
QDRANT_URL=https://my-cluster.gcp.cloud.qdrant.io:6333 \
QDRANT_API_KEY=your-key \
./scripts/qdrant-init-collection.sh
```

### 3. Add articles

Place Markdown files in `data/raw_articles/`:

```bash
mkdir -p data/raw_articles
# copy your .md files here
```

Articles may include a source URL in the text; otherwise the filename is used as a `file://` reference.

### 4. Ingest

Requires MongoDB and Qdrant to be running.

```bash
go run ./cmd/ingest
```

### 5. Run the API

Requires MongoDB and Redis to be running.

```bash
go run ./cmd/api
```

For live reload during development:

```bash
air
```

### 6. Ask a question

```bash
curl -s -X POST http://localhost:8080/api/v1/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "Apa hukum puasa Ramadhan?"}' | jq
```

Example response:

```json
{
  "answer": "...",
  "full_articles": [...],
  "chunks": [...]
}
```

## API

### `POST /api/v1/ask`


| Field      | Type   | Required | Description         |
| ---------- | ------ | -------- | ------------------- |
| `question` | string | yes      | The user's question |


Returns the generated answer along with the retrieved chunks and (when using `full_articles` context) the resolved source articles.

Each successful request is persisted to the `qna_records` MongoDB collection.

**Rate limiting:** requests are counted per client IP in Redis using a 1-minute window (`MAX_IP_REQUESTS_PER_MINUTE`). When exceeded, the API returns `429 Too Many Requests`. Client IP is resolved from `X-Forwarded-For`, `X-Real-IP`, then `RemoteAddr`.


| Status | Meaning                              |
| ------ | ------------------------------------ |
| `400`  | Invalid or missing `question`        |
| `429`  | Per-IP rate limit exceeded           |
| `500`  | Retrieval, LLM, or persistence error |


## Configuration

Environment variables are loaded from `.env` (walked up from the working directory), then OS environment, then defaults. Set `ENV_FILE` to point at a specific file.


| Variable                     | Default                                 | Description                                                   |
| ---------------------------- | --------------------------------------- | ------------------------------------------------------------- |
| `HTTP_PORT`                  | `8080`                                  | API listen port                                               |
| `QDRANT_HOST`                | `localhost`                             | Qdrant gRPC host (hostname only, no `https://`)               |
| `QDRANT_API_KEY`             |                                         | Qdrant Cloud API key; enables TLS for the Go gRPC client      |
| `QDRANT_GRPC_PORT`           | `6334`                                  | Qdrant gRPC port                                              |
| `QDRANT_URL`                 | `http://localhost:6333`                 | Qdrant REST URL (used by `scripts/qdrant-init-collection.sh`) |
| `QDRANT_COLLECTION`          | `indonesian_articles`                   | Chunk collection name                                         |
| `QDRANT_ARTICLE_COLLECTION`  | `indonesian_articles_full`              | Full-article collection name                                  |
| `MONGO_URI`                  | `mongodb://localhost:27017`             | MongoDB connection string                                     |
| `MONGO_DATABASE`             | `islamic_article_rag`                   | MongoDB database name                                         |
| `MONGO_ARTICLES_COLLECTION`  | `articles`                              | Collection for full article documents                         |
| `MONGO_QNA_COLLECTION`       | `qna_records`                           | Collection for QnA audit records                              |
| `REDIS_URL`                  | `redis://localhost:6379`                | Redis connection URL for rate limiting                        |
| `MAX_IP_REQUESTS_PER_MINUTE` | `5`                                     | Max `/ask` requests per IP per minute (enforced)              |
| `MAX_REQUESTS_PER_MINUTE`    | `30`                                    | Global per-minute limit (configured, not yet enforced)        |
| `MAX_REQUESTS_PER_DAY`       | `1000`                                  | Global daily limit (configured, not yet enforced)             |
| `MAX_QUESTION_CHARS`         | `200`                                   | Max question length (configured, not yet enforced)            |
| `LLM_PROVIDER`               | `ollama`                                | `ollama`, `google`, or `groq`                                 |
| `LLM_API_KEY`                |                                         | Required for `google` and `groq`                              |
| `LLM_API_URL`                | Ollama generate URL                     | Provider-specific endpoint                                    |
| `LLM_MODEL`                  | `qwen2.5:7b`                            | Model name                                                    |
| `OLLAMA_EMBEDDING_URL`       | `http://localhost:11434/api/embeddings` | Embedding endpoint                                            |
| `OLLAMA_EMBEDDING_MODEL`     | `bge-m3`                                | Embedding model (1024 dimensions)                             |
| `RAW_ARTICLES_DIR`           | `data/raw_articles`                     | Directory of `.md` files to ingest                            |
| `CHUNK_WINDOW_SIZE`          | `3`                                     | Paragraphs per chunk window                                   |
| `CHUNK_STEP_SIZE`            | `2`                                     | Paragraph step between windows                                |
| `MAX_CHUNK_CHARS`            | `6000`                                  | Max characters per embedded sub-chunk                         |
| `MIN_SIMILARITY_SCORE`       | `0.40`                                  | Minimum dense similarity threshold                            |
| `QNA_RETRIEVAL_LIMIT`        | `5`                                     | Number of chunks to retrieve                                  |
| `QNA_CONTEXT_SOURCE`         | `chunks`                                | `chunks` or `full_articles` — what the LLM sees               |


### LLM provider examples

**Ollama** (default — no API key):

```env
LLM_PROVIDER=ollama
LLM_API_URL=http://localhost:11434/api/generate
LLM_MODEL=qwen2.5:7b
```

**Google Gemini**:

```env
LLM_PROVIDER=google
LLM_API_KEY=your-key
LLM_API_URL=https://generativelanguage.googleapis.com/v1beta
LLM_MODEL=gemini-2.0-flash
```

**Groq**:

```env
LLM_PROVIDER=groq
LLM_API_KEY=your-key
LLM_API_URL=https://api.groq.com/openai/v1/chat/completions
LLM_MODEL=openai/gpt-oss-120b
```

## Project layout

```
cmd/
  api/          # HTTP server (POST /api/v1/ask)
  ingest/       # CLI ingestion tool
config/qdrant/  # Qdrant collection schemas
data/
  raw_articles/ # Source Markdown files (gitignored)
internal/
  config/       # Environment configuration
  handler/      # HTTP handlers
  model/        # Article, Chunk, QnARecord structs
  repository/
    mongo/      # Article and QnA record storage
    qdrant/     # Vector search
    redis/      # IP rate limiting
  service/      # Ingestion, embedding, LLM, QnA orchestration
pkg/regexutil/  # Quran reference extraction
scripts/        # Qdrant collection bootstrap
```

## How ingestion works

1. Each `.md` file is upserted as a full article in MongoDB (`articles` collection).
2. The file is split into overlapping paragraph windows (`CHUNK_WINDOW_SIZE` / `CHUNK_STEP_SIZE`).
3. Arabic script is stripped from chunks before embedding.
4. Quran references matching `(QS. Surah: verse)` are extracted and stored in metadata.
5. Long chunks are split at paragraph or word boundaries (`MAX_CHUNK_CHARS`).
6. Each sub-chunk is embedded via Ollama and upserted into `indonesian_articles` with dense and sparse (BM25) vectors.

## How QnA works

1. The API checks the client IP against Redis (`MAX_IP_REQUESTS_PER_MINUTE`, 1-minute TTL window).
2. The question is embedded with the same model used at ingest time.
3. Qdrant runs hybrid search: dense cosine similarity + sparse BM25, fused with RRF.
4. Depending on `QNA_CONTEXT_SOURCE`, the orchestrator builds context from retrieved chunks or fetches full articles from MongoDB by ID/URL.
5. The LLM generates an answer using a system prompt that enforces Indonesian/English output and structured Quran/Hadith citation tags.
6. The question, answer, model metadata, and source references are saved to MongoDB (`qna_records`).

## Development

```bash
# Run tests
go test ./...

# Build binaries
go build -o bin/api ./cmd/api
go build -o bin/ingest ./cmd/ingest
```

## License

[MIT](LICENSE)