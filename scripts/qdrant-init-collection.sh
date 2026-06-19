#!/bin/sh
# Bootstraps Qdrant chunk collections (hybrid dense + sparse vectors).
# Full article text is stored in MongoDB; do not use this script for articles.
#
# Local Docker: docker compose up qdrant-init
# Qdrant Cloud: set QDRANT_HOST and QDRANT_API_KEY in .env, then:
#   ./scripts/qdrant-init-collection.sh
set -eu

COLLECTION="${COLLECTION:-indonesian_articles}"

if [ -z "${CONFIG_FILE:-}" ]; then
  SCRIPT_DIR=$(CDPATH= cd "$(dirname "$0")" && pwd)
  CONFIG_FILE="${SCRIPT_DIR}/../config/qdrant/indonesian_articles.json"
fi

normalize_host() {
  host="$1"
  host="${host#https://}"
  host="${host#http://}"
  host="${host%%/*}"
  printf '%s' "$host"
}

resolve_qdrant_url() {
  if [ -n "${QDRANT_URL:-}" ]; then
    case "$QDRANT_URL" in
      http://*|https://*)
        printf '%s' "$QDRANT_URL"
        return
        ;;
    esac
    host=$(normalize_host "$QDRANT_URL")
  elif [ -n "${QDRANT_HOST:-}" ]; then
    host=$(normalize_host "$QDRANT_HOST")
  else
    printf '%s' "http://qdrant:6333"
    return
  fi

  port="${QDRANT_REST_PORT:-6333}"
  if [ -n "${QDRANT_API_KEY:-}" ]; then
    printf '%s' "https://${host}:${port}"
  else
    printf '%s' "http://${host}:${port}"
  fi
}

QDRANT_URL=$(resolve_qdrant_url)
QDRANT_URL="${QDRANT_URL%/}"

qdrant_curl() {
  if [ -n "${QDRANT_API_KEY:-}" ]; then
    curl -sf -H "api-key: ${QDRANT_API_KEY}" "$@"
  else
    curl -sf "$@"
  fi
}

echo "Waiting for Qdrant at ${QDRANT_URL}..."
until qdrant_curl "${QDRANT_URL}/healthz" >/dev/null; do
  sleep 2
done

if qdrant_curl "${QDRANT_URL}/collections/${COLLECTION}" >/dev/null; then
  echo "Collection '${COLLECTION}' already exists — skipping creation."
  exit 0
fi

echo "Creating collection '${COLLECTION}'..."
qdrant_curl -X PUT "${QDRANT_URL}/collections/${COLLECTION}" \
  -H "Content-Type: application/json" \
  -d @"${CONFIG_FILE}"
echo ""
echo "Collection '${COLLECTION}' created."
