package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dev-superbear/nexus-backend/internal/infra/kis"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCandleFetcher implements CandleFetcher for testing.
type mockCandleFetcher struct {
	candles []kis.NormalizedCandle
	err     error
	// captured args
	calledSymbol    string
	calledStartDate string
	calledEndDate   string
	calledPeriod    string
}

func (m *mockCandleFetcher) GetCandles(_ context.Context, symbol, startDate, endDate, period string) ([]kis.NormalizedCandle, error) {
	m.calledSymbol = symbol
	m.calledStartDate = startDate
	m.calledEndDate = endDate
	m.calledPeriod = period
	return m.candles, m.err
}

func TestGetCandles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sampleCandles := []kis.NormalizedCandle{
		{Time: "2024-01-01", Open: 100, High: 110, Low: 90, Close: 105, Volume: 1000},
		{Time: "2024-01-02", Open: 105, High: 115, Low: 95, Close: 110, Volume: 2000},
	}

	tests := []struct {
		name           string
		symbol         string
		queryString    string
		mockCandles    []kis.NormalizedCandle
		mockErr        error
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{}, mock *mockCandleFetcher)
	}{
		{
			name:           "valid request returns 200 with candles and symbol",
			symbol:         "005930",
			queryString:    "?period=W&startDate=2024-01-01&endDate=2024-01-31",
			mockCandles:    sampleCandles,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}, _ *mockCandleFetcher) {
				data := body["data"].(map[string]interface{})
				assert.Equal(t, "005930", data["symbol"])
				candles := data["candles"].([]interface{})
				assert.Len(t, candles, 2)
			},
		},
		{
			name:           "service error returns 502",
			symbol:         "005930",
			queryString:    "",
			mockErr:        errors.New("upstream failure"),
			expectedStatus: http.StatusBadGateway,
			checkResponse: func(t *testing.T, body map[string]interface{}, _ *mockCandleFetcher) {
				assert.Equal(t, "failed to fetch data", body["error"])
			},
		},
		{
			name:           "default period is D when not specified",
			symbol:         "005930",
			queryString:    "",
			mockCandles:    sampleCandles,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, _ map[string]interface{}, mock *mockCandleFetcher) {
				assert.Equal(t, "D", mock.calledPeriod)
			},
		},
		{
			name:           "custom period is passed through",
			symbol:         "005930",
			queryString:    "?period=W",
			mockCandles:    sampleCandles,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, _ map[string]interface{}, mock *mockCandleFetcher) {
				assert.Equal(t, "W", mock.calledPeriod)
			},
		},
		{
			name:           "startDate and endDate are passed to service",
			symbol:         "005930",
			queryString:    "?startDate=2024-01-01&endDate=2024-06-30",
			mockCandles:    sampleCandles,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, _ map[string]interface{}, mock *mockCandleFetcher) {
				assert.Equal(t, "2024-01-01", mock.calledStartDate)
				assert.Equal(t, "2024-06-30", mock.calledEndDate)
			},
		},
		{
			name:           "empty candles list returns 200 with empty array",
			symbol:         "999999",
			queryString:    "",
			mockCandles:    []kis.NormalizedCandle{},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}, _ *mockCandleFetcher) {
				data := body["data"].(map[string]interface{})
				candles := data["candles"].([]interface{})
				assert.Empty(t, candles)
			},
		},
		{
			name:           "symbol is correctly extracted from URL param",
			symbol:         "035720",
			queryString:    "",
			mockCandles:    sampleCandles,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, _ map[string]interface{}, mock *mockCandleFetcher) {
				assert.Equal(t, "035720", mock.calledSymbol)
			},
		},
		{
			name:           "correct JSON structure with data.candles and data.symbol",
			symbol:         "005930",
			queryString:    "",
			mockCandles:    sampleCandles,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}, _ *mockCandleFetcher) {
				// Top-level must have "data" key
				data, ok := body["data"].(map[string]interface{})
				require.True(t, ok, "response must have 'data' key")

				// data must have "candles" key
				_, ok = data["candles"]
				require.True(t, ok, "data must have 'candles' key")

				// data must have "symbol" key
				_, ok = data["symbol"]
				require.True(t, ok, "data must have 'symbol' key")

				// no other top-level keys
				assert.Len(t, body, 1, "response should only have 'data' at top level")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCandleFetcher{
				candles: tt.mockCandles,
				err:     tt.mockErr,
			}
			h := NewCandleHandler(mock)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "symbol", Value: tt.symbol}}
			c.Request = httptest.NewRequest(http.MethodGet, "/candles/"+tt.symbol+tt.queryString, nil)

			h.GetCandles(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var body map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &body)
			require.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, body, mock)
			}
		})
	}
}
