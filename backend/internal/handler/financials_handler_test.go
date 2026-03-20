package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dev-superbear/nexus-backend/internal/infra/dart"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFinancialsFetcher implements FinancialsFetcher for testing.
type mockFinancialsFetcher struct {
	data dart.NormalizedFinancials
	err  error
}

func (m *mockFinancialsFetcher) GetFinancials(_ context.Context, _ string) (dart.NormalizedFinancials, error) {
	return m.data, m.err
}

func TestFinancialsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	revenue := 100000.0
	roe := 15.5

	tests := []struct {
		name           string
		symbol         string
		mock           *mockFinancialsFetcher
		handlerFunc    func(h *FinancialsHandler) gin.HandlerFunc
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:   "GET returns 200 with financial data",
			symbol: "005930",
			mock: &mockFinancialsFetcher{
				data: dart.NormalizedFinancials{
					Revenue: &revenue,
					ROE:     &roe,
				},
			},
			handlerFunc: func(h *FinancialsHandler) gin.HandlerFunc {
				return h.GetFinancials
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].(map[string]interface{})
				assert.Equal(t, revenue, data["revenue"])
				assert.Equal(t, roe, data["roe"])
			},
		},
		{
			name:   "service error returns 502",
			symbol: "005930",
			mock: &mockFinancialsFetcher{
				err: errors.New("DART API down"),
			},
			handlerFunc: func(h *FinancialsHandler) gin.HandlerFunc {
				return h.GetFinancials
			},
			expectedStatus: http.StatusBadGateway,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "failed to fetch data", body["error"])
			},
		},
		{
			name:   "GetSectorCompare returns 200 with empty array",
			symbol: "",
			mock:   &mockFinancialsFetcher{},
			handlerFunc: func(h *FinancialsHandler) gin.HandlerFunc {
				return h.GetSectorCompare
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].([]interface{})
				assert.Empty(t, data)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewFinancialsHandler(tt.mock)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			if tt.symbol != "" {
				c.Params = gin.Params{{Key: "symbol", Value: tt.symbol}}
			}
			c.Request = httptest.NewRequest(http.MethodGet, "/financials/"+tt.symbol, nil)

			handler := tt.handlerFunc(h)
			handler(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var body map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &body)
			require.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, body)
			}
		})
	}
}
