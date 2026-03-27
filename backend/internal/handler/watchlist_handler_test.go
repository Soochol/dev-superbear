package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dev-superbear/nexus-backend/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockWatchlistRepo implements WatchlistRepository for testing.
type mockWatchlistRepo struct {
	items    []repository.WatchlistItem
	item     *repository.WatchlistItem
	getErr   error
	addErr   error
	removeErr error
}

func (m *mockWatchlistRepo) GetByUser(_ context.Context, _ string) ([]repository.WatchlistItem, error) {
	return m.items, m.getErr
}

func (m *mockWatchlistRepo) Add(_ context.Context, _ string, _, _ string) (*repository.WatchlistItem, error) {
	return m.item, m.addErr
}

func (m *mockWatchlistRepo) Remove(_ context.Context, _ string, _ string) error {
	return m.removeErr
}

func TestWatchlistHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sampleItems := []repository.WatchlistItem{
		{ID: 1, UserID: "00000000-0000-0000-0000-000000000001", Symbol: "005930", Name: "Samsung", CreatedAt: time.Now()},
		{ID: 2, UserID: "00000000-0000-0000-0000-000000000001", Symbol: "035720", Name: "Kakao", CreatedAt: time.Now()},
	}

	sampleItem := &repository.WatchlistItem{
		ID: 3, UserID: "00000000-0000-0000-0000-000000000001", Symbol: "000660", Name: "SK Hynix", CreatedAt: time.Now(),
	}

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		params         gin.Params
		mock           *mockWatchlistRepo
		handlerFunc    func(h *WatchlistHandler) gin.HandlerFunc
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:   "GET returns 200 with items list",
			method: http.MethodGet,
			path:   "/watchlist",
			mock:   &mockWatchlistRepo{items: sampleItems},
			handlerFunc: func(h *WatchlistHandler) gin.HandlerFunc {
				return h.GetWatchlist
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var body map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &body)
				require.NoError(t, err)
				data := body["data"].([]interface{})
				assert.Len(t, data, 2)
			},
		},
		{
			name:   "POST with valid body returns 201",
			method: http.MethodPost,
			path:   "/watchlist",
			body:   `{"symbol":"000660","name":"SK Hynix"}`,
			mock:   &mockWatchlistRepo{item: sampleItem},
			handlerFunc: func(h *WatchlistHandler) gin.HandlerFunc {
				return h.AddToWatchlist
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var body map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &body)
				require.NoError(t, err)
				data := body["data"].(map[string]interface{})
				assert.Equal(t, "000660", data["symbol"])
				assert.Equal(t, "SK Hynix", data["name"])
			},
		},
		{
			name:   "POST with missing symbol returns 400",
			method: http.MethodPost,
			path:   "/watchlist",
			body:   `{"name":"SK Hynix"}`,
			mock:   &mockWatchlistRepo{},
			handlerFunc: func(h *WatchlistHandler) gin.HandlerFunc {
				return h.AddToWatchlist
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var body map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &body)
				require.NoError(t, err)
				assert.Contains(t, body, "error")
			},
		},
		{
			name:   "POST with missing name returns 400",
			method: http.MethodPost,
			path:   "/watchlist",
			body:   `{"symbol":"000660"}`,
			mock:   &mockWatchlistRepo{},
			handlerFunc: func(h *WatchlistHandler) gin.HandlerFunc {
				return h.AddToWatchlist
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var body map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &body)
				require.NoError(t, err)
				assert.Contains(t, body, "error")
			},
		},
		{
			name:   "DELETE returns 204 no content",
			method: http.MethodDelete,
			path:   "/watchlist/005930",
			params: gin.Params{{Key: "symbol", Value: "005930"}},
			mock:   &mockWatchlistRepo{},
			handlerFunc: func(h *WatchlistHandler) gin.HandlerFunc {
				return h.RemoveFromWatchlist
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Empty(t, w.Body.Bytes())
			},
		},
		{
			name:   "GET repo error returns 500",
			method: http.MethodGet,
			path:   "/watchlist",
			mock:   &mockWatchlistRepo{getErr: errors.New("db down")},
			handlerFunc: func(h *WatchlistHandler) gin.HandlerFunc {
				return h.GetWatchlist
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var body map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &body)
				require.NoError(t, err)
				assert.Equal(t, "failed to fetch watchlist", body["error"])
			},
		},
		{
			name:   "POST repo error returns 500",
			method: http.MethodPost,
			path:   "/watchlist",
			body:   `{"symbol":"000660","name":"SK Hynix"}`,
			mock:   &mockWatchlistRepo{addErr: errors.New("db down")},
			handlerFunc: func(h *WatchlistHandler) gin.HandlerFunc {
				return h.AddToWatchlist
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var body map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &body)
				require.NoError(t, err)
				assert.Equal(t, "failed to add to watchlist", body["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewWatchlistHandler(tt.mock)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			if tt.params != nil {
				c.Params = tt.params
			}

			var bodyReader *bytes.Buffer
			if tt.body != "" {
				bodyReader = bytes.NewBufferString(tt.body)
			} else {
				bodyReader = &bytes.Buffer{}
			}
			c.Request = httptest.NewRequest(tt.method, tt.path, bodyReader)
			if tt.body != "" {
				c.Request.Header.Set("Content-Type", "application/json")
			}

			handlerFn := tt.handlerFunc(h)
			handlerFn(c)

			// Use gin's writer status because c.Status() doesn't flush
			// to httptest.ResponseRecorder without a body write.
			assert.Equal(t, tt.expectedStatus, c.Writer.Status())

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
