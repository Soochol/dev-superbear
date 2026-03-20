package pgutil

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUUID_RoundTrip(t *testing.T) {
	input := "550e8400-e29b-41d4-a716-446655440000"
	u, err := ParseUUID(input)
	require.NoError(t, err)
	assert.True(t, u.Valid)
	assert.Equal(t, input, UUIDToString(u))
}

func TestParseUUID_Invalid(t *testing.T) {
	_, err := ParseUUID("not-a-uuid")
	assert.Error(t, err)
}

func TestUUIDToString_Invalid(t *testing.T) {
	var u pgtype.UUID
	assert.Equal(t, "", UUIDToString(u))
}
