package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dev-superbear/nexus-backend/internal/infra/kis"
)

func TestBuildDSLContext_ValidSnapshot(t *testing.T) {
	snapshot := []byte(`{
		"high": 55000, "low": 50000, "close": 53000, "volume": 1000000,
		"preClose": 52000,
		"preMa": {"5": 52500, "20": 51000, "60": 50000, "120": 49000, "200": 48000}
	}`)
	price := &kis.PriceSnapshot{Close: 54000, High: 55500, Low: 53500, Volume: 500000}
	ctx, err := BuildDSLContext(snapshot, price)
	require.NoError(t, err)
	assert.Equal(t, 54000.0, ctx.Close)
	assert.Equal(t, 55500.0, ctx.High)
	assert.Equal(t, 55000.0, ctx.EventHigh)
	assert.Equal(t, 53000.0, ctx.EventClose)
	assert.Equal(t, 52500.0, ctx.PreEventMA5)
	assert.Equal(t, 48000.0, ctx.PreEventMA200)
	assert.Equal(t, 52000.0, ctx.PreEventClose)
}

func TestBuildDSLContext_InvalidJSON(t *testing.T) {
	price := &kis.PriceSnapshot{Close: 54000}
	_, err := BuildDSLContext([]byte(`{invalid`), price)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal event snapshot")
}

func TestBuildDSLContext_MissingFields(t *testing.T) {
	snapshot := []byte(`{"high": 55000}`)
	price := &kis.PriceSnapshot{Close: 54000}
	ctx, err := BuildDSLContext(snapshot, price)
	require.NoError(t, err)
	assert.Equal(t, 55000.0, ctx.EventHigh)
	assert.Equal(t, 0.0, ctx.EventLow)
	assert.Equal(t, 0.0, ctx.PreEventMA5)
}
