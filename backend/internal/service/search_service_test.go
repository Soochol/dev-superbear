package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dev-superbear/nexus-backend/internal/service"
)

func TestSearchService_Validate(t *testing.T) {
	svc := service.NewSearchService(nil)

	t.Run("accepts valid scan query", func(t *testing.T) {
		result := svc.Validate(context.Background(), "scan where volume > 1000000")
		assert.True(t, result.Valid)
		assert.Empty(t, result.Error)
	})

	t.Run("accepts scan with sort and limit", func(t *testing.T) {
		result := svc.Validate(context.Background(), "scan where volume > 1000000 sort by trade_value desc limit 50")
		assert.True(t, result.Valid)
	})

	t.Run("rejects empty input", func(t *testing.T) {
		result := svc.Validate(context.Background(), "")
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Error)
	})

	t.Run("accepts condition assignment", func(t *testing.T) {
		result := svc.Validate(context.Background(), "success = close >= event_high * 2.0")
		assert.True(t, result.Valid)
	})
}
