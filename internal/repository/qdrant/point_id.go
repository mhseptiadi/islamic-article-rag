package qdrant

import (
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

func pointIDFromString(id string) *qdrant.PointId {
	if parsed, err := uuid.Parse(id); err == nil {
		return qdrant.NewIDUUID(parsed.String())
	}

	derived := uuid.NewSHA1(uuid.NameSpaceOID, []byte(id))
	return qdrant.NewIDUUID(derived.String())
}
