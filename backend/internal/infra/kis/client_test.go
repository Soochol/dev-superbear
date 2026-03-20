package kis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFakeKISServer creates an httptest server that handles token and API requests.
// tokenCount tracks how many token requests have been made.
func newFakeKISServer(t *testing.T, tokenCount *atomic.Int32, candleHandler, priceHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/oauth2/tokenP", func(w http.ResponseWriter, r *http.Request) {
		if tokenCount != nil {
			tokenCount.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "fake-token-abc123",
			"token_type":   "Bearer",
			"expires_in":   86400,
		})
	})

	if candleHandler != nil {
		mux.HandleFunc("/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice", candleHandler)
	}
	if priceHandler != nil {
		mux.HandleFunc("/uapi/domestic-stock/v1/quotations/inquire-price", priceHandler)
	}

	return httptest.NewServer(mux)
}

func TestGetCandles_CorrectURLAndQueryParams(t *testing.T) {
	var capturedReq *http.Request

	srv := newFakeKISServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CandleResponse{Output2: []KISCandle{}})
	}, nil)
	defer srv.Close()

	client := NewClient("mykey", "mysecret", srv.URL)
	_, err := client.GetCandles(context.Background(), "005930", "20260101", "20260318", "D")
	require.NoError(t, err)
	require.NotNil(t, capturedReq)

	assert.Equal(t, "/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice", capturedReq.URL.Path)
	q := capturedReq.URL.Query()
	assert.Equal(t, "J", q.Get("FID_COND_MRKT_DIV_CODE"))
	assert.Equal(t, "005930", q.Get("FID_INPUT_ISCD"))
	assert.Equal(t, "20260101", q.Get("FID_INPUT_DATE_1"))
	assert.Equal(t, "20260318", q.Get("FID_INPUT_DATE_2"))
	assert.Equal(t, "D", q.Get("FID_PERIOD_DIV_CODE"))
	assert.Equal(t, "0", q.Get("FID_ORG_ADJ_PRC"))
}

func TestGetCandles_CorrectAuthHeaders(t *testing.T) {
	var capturedReq *http.Request

	srv := newFakeKISServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CandleResponse{Output2: []KISCandle{}})
	}, nil)
	defer srv.Close()

	client := NewClient("mykey", "mysecret", srv.URL)
	_, err := client.GetCandles(context.Background(), "005930", "20260101", "20260318", "D")
	require.NoError(t, err)
	require.NotNil(t, capturedReq)

	assert.Equal(t, "Bearer fake-token-abc123", capturedReq.Header.Get("authorization"))
	assert.Equal(t, "mykey", capturedReq.Header.Get("appkey"))
	assert.Equal(t, "mysecret", capturedReq.Header.Get("appsecret"))
	assert.Equal(t, trIDDailyChart, capturedReq.Header.Get("tr_id"))
}

func TestGetCandles_ReturnsNormalizedCandles(t *testing.T) {
	srv := newFakeKISServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		resp := CandleResponse{
			Output2: []KISCandle{
				{
					StckBsopDate: "20260318",
					StckOprc:     "77000",
					StckHgpr:     "79000",
					StckLwpr:     "76500",
					StckClpr:     "78400",
					AcmlVol:      "15234000",
				},
				{
					StckBsopDate: "20260317",
					StckOprc:     "76000",
					StckHgpr:     "77500",
					StckLwpr:     "75800",
					StckClpr:     "77000",
					AcmlVol:      "12100000",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}, nil)
	defer srv.Close()

	client := NewClient("mykey", "mysecret", srv.URL)
	candles, err := client.GetCandles(context.Background(), "005930", "20260101", "20260318", "D")
	require.NoError(t, err)
	require.Len(t, candles, 2)

	// Sorted ascending by date
	assert.Equal(t, "2026-03-17", candles[0].Time)
	assert.Equal(t, float64(76000), candles[0].Open)
	assert.Equal(t, float64(77500), candles[0].High)
	assert.Equal(t, float64(75800), candles[0].Low)
	assert.Equal(t, float64(77000), candles[0].Close)
	assert.Equal(t, int64(12100000), candles[0].Volume)

	assert.Equal(t, "2026-03-18", candles[1].Time)
	assert.Equal(t, float64(77000), candles[1].Open)
}

func TestGetCandles_ErrorOnNon200(t *testing.T) {
	srv := newFakeKISServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}, nil)
	defer srv.Close()

	client := NewClient("mykey", "mysecret", srv.URL)
	_, err := client.GetCandles(context.Background(), "005930", "20260101", "20260318", "D")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestTokenCaching_ReusesToken(t *testing.T) {
	var tokenCount atomic.Int32

	srv := newFakeKISServer(t, &tokenCount, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CandleResponse{Output2: []KISCandle{}})
	}, nil)
	defer srv.Close()

	client := NewClient("mykey", "mysecret", srv.URL)
	ctx := context.Background()

	_, err := client.GetCandles(ctx, "005930", "20260101", "20260318", "D")
	require.NoError(t, err)

	_, err = client.GetCandles(ctx, "005930", "20260101", "20260318", "D")
	require.NoError(t, err)

	assert.Equal(t, int32(1), tokenCount.Load(), "should only request token once; second call should use cache")
}

func TestTokenRefresh_ExpiredTokenTriggersNewRequest(t *testing.T) {
	var tokenCount atomic.Int32

	srv := newFakeKISServer(t, &tokenCount, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CandleResponse{Output2: []KISCandle{}})
	}, nil)
	defer srv.Close()

	client := NewClient("mykey", "mysecret", srv.URL)
	ctx := context.Background()

	// First call: fetches a token
	_, err := client.GetCandles(ctx, "005930", "20260101", "20260318", "D")
	require.NoError(t, err)
	assert.Equal(t, int32(1), tokenCount.Load())

	// Simulate expired token by setting ExpiresAt to the past
	client.mu.Lock()
	client.cachedToken.ExpiresAt = time.Now().Add(-time.Hour)
	client.mu.Unlock()

	// Second call: token is expired, should fetch a new one
	_, err = client.GetCandles(ctx, "005930", "20260101", "20260318", "D")
	require.NoError(t, err)
	assert.Equal(t, int32(2), tokenCount.Load(), "expired token should trigger a new token request")
}

func TestGetCurrentPrice_CorrectURLAndHeaders(t *testing.T) {
	var capturedReq *http.Request

	srv := newFakeKISServer(t, nil, nil, func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		resp := PriceResponse{
			Output: &KISPriceResponse{
				StckPrpr:   "78400",
				PrdyVrss:   "1400",
				PrdyCtrt:   "1.82",
				AcmlVol:    "15234000",
				Per:        "12.5",
				Eps:        "6272",
				HtsKorIsnm: "삼성전자",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	client := NewClient("mykey", "mysecret", srv.URL)
	price, err := client.GetCurrentPrice(context.Background(), "005930")
	require.NoError(t, err)
	require.NotNil(t, capturedReq)

	// Check URL path and query params
	assert.Equal(t, "/uapi/domestic-stock/v1/quotations/inquire-price", capturedReq.URL.Path)
	q := capturedReq.URL.Query()
	assert.Equal(t, "J", q.Get("FID_COND_MRKT_DIV_CODE"))
	assert.Equal(t, "005930", q.Get("FID_INPUT_ISCD"))

	// Check headers
	assert.Equal(t, "Bearer fake-token-abc123", capturedReq.Header.Get("authorization"))
	assert.Equal(t, "mykey", capturedReq.Header.Get("appkey"))
	assert.Equal(t, "mysecret", capturedReq.Header.Get("appsecret"))
	assert.Equal(t, trIDCurrentPrice, capturedReq.Header.Get("tr_id"))

	// Check response parsing
	require.NotNil(t, price)
	assert.Equal(t, "78400", price.StckPrpr)
	assert.Equal(t, "삼성전자", price.HtsKorIsnm)
}
