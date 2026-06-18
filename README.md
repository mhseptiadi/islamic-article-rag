# Islamic Article RAG

A retrieval-augmented generation (RAG) backend for answering questions about Indonesian Islamic articles. Articles are ingested from Markdown files, embedded and indexed in [Qdrant](https://qdrant.tech/), and served through a simple HTTP API that combines hybrid search with an LLM.

## Features

- **Ingestion pipeline** — reads `.md` files, splits them into overlapping paragraph windows, extracts Quran references (`(QS. Surah: verse)`), and stores both chunk vectors and full article text.
- **Hybrid retrieval** — dense semantic search (Ollama `bge-m3`, 1024-dim) fused with sparse BM25 keyword search via Reciprocal Rank Fusion (RRF) in Qdrant.
- **Flexible LLM backends** — Ollama (default), Google Gemini, or Groq.
- **Structured citations** — the LLM is prompted to quote Quran and Hadith using XML-style tags (`<quran>`, `<hadith>`).
- **Configurable context** — feed the LLM either retrieved chunks or full source articles.

## Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────┐
│  .md files  │────▶│  ingest (CLI)    │────▶│   Qdrant    │
└─────────────┘     │  embed + chunk   │     │  collections│
                    └──────────────────┘     └──────┬──────┘
                                                    │
┌─────────────┐     ┌──────────────────┐            │
│   Client    │────▶│  api (HTTP)      │◀───────────┘
└─────────────┘     │  hybrid search   │
                    │  + LLM answer    │
                    └──────────────────┘
```

| Collection | Purpose |
|---|---|
| `indonesian_articles` | Chunk vectors (dense + sparse) for hybrid search |
| `indonesian_articles_full` | Full article text, keyed by article ID / source URL |

## Prerequisites

- [Go](https://go.dev/) 1.25+
- [Docker](https://www.docker.com/) (for Qdrant)
- [Ollama](https://ollama.com/) (default embedding and LLM provider)

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

Edit `.env` as needed. The defaults assume Qdrant on `localhost:6333` and Ollama on `localhost:11434`.

### 2. Start Qdrant

```bash
docker compose up -d
```

This starts Qdrant and bootstraps both collections from `config/qdrant/`. The dashboard is available at [http://localhost:6333/dashboard](http://localhost:6333/dashboard).

### 3. Add articles

Place Markdown files in `data/raw_articles/`:

```bash
mkdir -p data/raw_articles
# copy your .md files here
```

Articles may include a source URL in the text; otherwise the filename is used as a `file://` reference.

### 4. Ingest

```bash
go run ./cmd/ingest
```

### 5. Run the API

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

| Field | Type | Required | Description |
|---|---|---|---|
| `question` | string | yes | The user's question |

Returns the generated answer along with the retrieved chunks and (when using `full_articles` context) the resolved source articles.

## Configuration

Environment variables are loaded from `.env` (walked up from the working directory), then OS environment, then defaults. Set `ENV_FILE` to point at a specific file.

| Variable | Default | Description |
|---|---|---|
| `HTTP_PORT` | `8080` | API listen port |
| `QDRANT_HOST` | `localhost` | Qdrant gRPC host |
| `QDRANT_GRPC_PORT` | `6334` | Qdrant gRPC port |
| `QDRANT_COLLECTION` | `indonesian_articles` | Chunk collection name |
| `QDRANT_ARTICLE_COLLECTION` | `indonesian_articles_full` | Full-article collection name |
| `LLM_PROVIDER` | `ollama` | `ollama`, `google`, or `groq` |
| `LLM_API_KEY` | | Required for `google` and `groq` |
| `LLM_API_URL` | Ollama generate URL | Provider-specific endpoint |
| `LLM_MODEL` | `qwen2.5:7b` | Model name |
| `OLLAMA_EMBEDDING_URL` | `http://localhost:11434/api/embeddings` | Embedding endpoint |
| `OLLAMA_EMBEDDING_MODEL` | `bge-m3` | Embedding model (1024 dimensions) |
| `RAW_ARTICLES_DIR` | `data/raw_articles` | Directory of `.md` files to ingest |
| `CHUNK_WINDOW_SIZE` | `3` | Paragraphs per chunk window |
| `CHUNK_STEP_SIZE` | `2` | Paragraph step between windows |
| `MAX_CHUNK_CHARS` | `6000` | Max characters per embedded sub-chunk |
| `MIN_SIMILARITY_SCORE` | `0.40` | Minimum dense similarity threshold |
| `QNA_RETRIEVAL_LIMIT` | `5` | Number of chunks to retrieve |
| `QNA_CONTEXT_SOURCE` | `chunks` | `chunks` or `full_articles` — what the LLM sees |

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
  model/        # Article, Chunk, Metadata structs
  repository/   # Qdrant data access
  service/      # Ingestion, embedding, LLM, QnA orchestration
pkg/regexutil/  # Quran reference extraction
scripts/        # Qdrant collection bootstrap
```

## How ingestion works

1. Each `.md` file is stored as a full article in `indonesian_articles_full`.
2. The file is split into overlapping paragraph windows (`CHUNK_WINDOW_SIZE` / `CHUNK_STEP_SIZE`).
3. Arabic script is stripped from chunks before embedding.
4. Quran references matching `(QS. Surah: verse)` are extracted and stored in metadata.
5. Long chunks are split at paragraph or word boundaries (`MAX_CHUNK_CHARS`).
6. Each sub-chunk is embedded via Ollama and upserted into `indonesian_articles` with dense and sparse (BM25) vectors.

## How QnA works

1. The question is embedded with the same model used at ingest time.
2. Qdrant runs hybrid search: dense cosine similarity + sparse BM25, fused with RRF.
3. Depending on `QNA_CONTEXT_SOURCE`, the orchestrator builds context from retrieved chunks or fetches full articles by ID/URL.
4. The LLM generates an answer using a system prompt that enforces Indonesian/English output and structured Quran/Hadith citation tags.

## Development

```bash
# Run tests
go test ./...

# Build binaries
go build -o bin/api ./cmd/api
go build -o bin/ingest ./cmd/ingest
```

## License

Private project.
