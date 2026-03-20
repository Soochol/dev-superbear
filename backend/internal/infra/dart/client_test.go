package dart

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeFinancialStatements(t *testing.T) {
	t.Run("normalizes financial statement response", func(t *testing.T) {
		raw := RawFinancials{
			Revenue:         "67890000000000",
			OperatingProfit: "12345000000000",
			NetIncome:       "9876000000000",
		}

		result := NormalizeFinancialStatements(raw)
		assert.NotNil(t, result.Revenue)
		assert.InDelta(t, 678900.0, *result.Revenue, 1.0)
		assert.NotNil(t, result.OperatingProfit)
		assert.InDelta(t, 123450.0, *result.OperatingProfit, 1.0)
	})

	t.Run("handles missing values gracefully", func(t *testing.T) {
		result := NormalizeFinancialStatements(RawFinancials{})
		assert.Nil(t, result.Revenue)
		assert.Nil(t, result.OperatingProfit)
	})

	t.Run("zero revenue results in nil net margin (division by zero protection)", func(t *testing.T) {
		raw := RawFinancials{
			Revenue:   "0",
			NetIncome: "5000000000000",
		}
		result := NormalizeFinancialStatements(raw)
		// Revenue of "0" parses to 0.0, toEok rounds 0/1e8 = 0, so *revenue == 0 => netMargin is nil
		assert.Nil(t, result.NetMargin, "net margin should be nil when revenue is zero")
	})

	t.Run("negative net income results in negative net margin", func(t *testing.T) {
		raw := RawFinancials{
			Revenue:   "10000000000000",  // 100000 eok
			NetIncome: "-2000000000000",  // -20000 eok
		}
		result := NormalizeFinancialStatements(raw)
		require.NotNil(t, result.NetMargin)
		assert.Less(t, *result.NetMargin, 0.0, "net margin should be negative when net income is negative")
	})

	t.Run("calculates correct net margin percentage", func(t *testing.T) {
		raw := RawFinancials{
			Revenue:   "10000000000000",  // 100000 eok
			NetIncome: "1000000000000",   // 10000 eok
		}
		result := NormalizeFinancialStatements(raw)
		require.NotNil(t, result.NetMargin)
		// (10000 / 100000) * 100 = 10.0%
		assert.InDelta(t, 10.0, *result.NetMargin, 0.01, "net margin should be 10%")
	})

	t.Run("PER PBR ROE are always nil (not computed yet)", func(t *testing.T) {
		raw := RawFinancials{
			Revenue:         "10000000000000",
			OperatingProfit: "5000000000000",
			NetIncome:       "3000000000000",
		}
		result := NormalizeFinancialStatements(raw)
		assert.Nil(t, result.PER, "PER should be nil")
		assert.Nil(t, result.PBR, "PBR should be nil")
		assert.Nil(t, result.ROE, "ROE should be nil")
	})
}

// newFakeDARTServer creates an httptest server that serves DART API responses.
func newFakeDARTServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/fnlttSinglAcnt.json", handler)
	return httptest.NewServer(mux)
}

func TestFetchFinancialStatements_ValidData(t *testing.T) {
	srv := newFakeDARTServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := DARTFinancialResponse{
			Status:  "000",
			Message: "정상",
			List: []DARTFinancialItem{
				{AccountNm: "매출액", ThstrmAmt: "10000000000000"},
				{AccountNm: "영업이익", ThstrmAmt: "3000000000000"},
				{AccountNm: "당기순이익", ThstrmAmt: "2000000000000"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	client := NewClient("test-api-key", srv.URL)
	result, err := client.FetchFinancialStatements(context.Background(), "00126380", "2025", "11011")
	require.NoError(t, err)

	require.NotNil(t, result.Revenue)
	assert.InDelta(t, 100000.0, *result.Revenue, 1.0)
	require.NotNil(t, result.OperatingProfit)
	assert.InDelta(t, 30000.0, *result.OperatingProfit, 1.0)
	require.NotNil(t, result.NetMargin)
	assert.InDelta(t, 20.0, *result.NetMargin, 0.01)
}

func TestFetchFinancialStatements_DARTStatusError(t *testing.T) {
	srv := newFakeDARTServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := DARTFinancialResponse{
			Status:  "013",
			Message: "조회된 데이터가 없습니다.",
			List:    nil,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	client := NewClient("test-api-key", srv.URL)
	_, err := client.FetchFinancialStatements(context.Background(), "00126380", "2025", "11011")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "013")
	assert.Contains(t, err.Error(), "DART API error")
}

func TestFetchFinancialStatements_HTTPError(t *testing.T) {
	srv := newFakeDARTServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	client := NewClient("test-api-key", srv.URL)
	_, err := client.FetchFinancialStatements(context.Background(), "00126380", "2025", "11011")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestFetchFinancialStatements_CorrectQueryParams(t *testing.T) {
	var capturedReq *http.Request

	srv := newFakeDARTServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		resp := DARTFinancialResponse{
			Status:  "000",
			Message: "정상",
			List:    []DARTFinancialItem{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	client := NewClient("my-dart-key", srv.URL)
	_, err := client.FetchFinancialStatements(context.Background(), "00126380", "2025", "11013")
	require.NoError(t, err)
	require.NotNil(t, capturedReq)

	q := capturedReq.URL.Query()
	assert.Equal(t, "my-dart-key", q.Get("crtfc_key"))
	assert.Equal(t, "00126380", q.Get("corp_code"))
	assert.Equal(t, "2025", q.Get("bsns_year"))
	assert.Equal(t, "11013", q.Get("reprt_code"))
}
