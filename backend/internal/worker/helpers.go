package worker

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dev-superbear/nexus-backend/internal/infra/pgutil"
)

func uuidToString(u pgtype.UUID) string {
	return pgutil.UUIDToString(u)
}

func stringToUUID(s string) (pgtype.UUID, error) {
	return pgutil.ParseUUID(s)
}
