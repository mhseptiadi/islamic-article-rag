#!/bin/sh
# Bootstraps Qdrant chunk collections (hybrid dense + sparse vectors).
# Full article text is stored in MongoDB; do not use this script for articles.
set -eu

QDRANT_URL="${QDRANT_URL:-http://qdrant:6333}"
COLLECTION="${COLLECTION:-indonesian_articles}"
CONFIG_FILE="${CONFIG_FILE:-/init/collection.json}"

echo "Waiting for Qdrant at ${QDRANT_URL}..."
until curl -sf "${QDRANT_URL}/healthz" >/dev/null; do
  sleep 2
done

if curl -sf "${QDRANT_URL}/collections/${COLLECTION}" >/dev/null; then
  echo "Collection '${COLLECTION}' already exists — skipping creation."
  exit 0
fi

echo "Creating collection '${COLLECTION}'..."
curl -sf -X PUT "${QDRANT_URL}/collections/${COLLECTION}" \
  -H "Content-Type: application/json" \
  -d @"${CONFIG_FILE}"
echo ""
echo "Collection '${COLLECTION}' created."
