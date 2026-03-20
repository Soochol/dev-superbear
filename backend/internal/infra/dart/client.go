package dart

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const dartBaseURL = "https://opendart.fss.or.kr/api"

type Client struct {
	httpClient *http.Client
	apiKey     string
}

func NewClient(apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		apiKey:     apiKey,
	}
}

func toEok(val string) *float64 {
	if val == "" {
		return nil
	}
	num, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return nil
	}
	result := math.Round(num / 100_000_000)
	return &result
}

func NormalizeFinancialStatements(raw RawFinancials) NormalizedFinancials {
	revenue := toEok(raw.Revenue)
	operatingProfit := toEok(raw.OperatingProfit)
	netIncome := toEok(raw.NetIncome)

	var netMargin *float64
	if revenue != nil && netIncome != nil && *revenue != 0 {
		m := (*netIncome / *revenue) * 100
		netMargin = &m
	}

	return NormalizedFinancials{
		Revenue:         revenue,
		OperatingProfit: operatingProfit,
		NetMargin:       netMargin,
		PER:             nil,
		PBR:             nil,
		ROE:             nil,
	}
}

func (c *Client) FetchFinancialStatements(ctx context.Context, corpCode, year, reportCode string) (NormalizedFinancials, error) {
	if reportCode == "" {
		reportCode = "11011"
	}

	params := url.Values{}
	params.Set("crtfc_key", c.apiKey)
	params.Set("corp_code", corpCode)
	params.Set("bsns_year", year)
	params.Set("reprt_code", reportCode)

	reqURL := fmt.Sprintf("%s/fnlttSinglAcnt.json?%s", dartBaseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return NormalizedFinancials{}, fmt.Errorf("create DART request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return NormalizedFinancials{}, fmt.Errorf("DART request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NormalizedFinancials{}, fmt.Errorf("DART request returned status %d", resp.StatusCode)
	}

	var dartResp DARTFinancialResponse
	if err := json.NewDecoder(resp.Body).Decode(&dartResp); err != nil {
		return NormalizedFinancials{}, fmt.Errorf("decode DART response: %w", err)
	}

	if dartResp.Status != "000" {
		return NormalizedFinancials{}, fmt.Errorf("DART API error: status=%s message=%s", dartResp.Status, dartResp.Message)
	}

	raw := RawFinancials{}
	for _, item := range dartResp.List {
		switch item.AccountNm {
		case "매출액", "수익(매출액)":
			raw.Revenue = item.ThstrmAmt
		case "영업이익", "영업이익(손실)":
			raw.OperatingProfit = item.ThstrmAmt
		case "당기순이익", "당기순이익(손실)":
			raw.NetIncome = item.ThstrmAmt
		}
	}

	slog.Info("DART: fetched financial statements",
		"corpCode", corpCode,
		"year", year,
	)

	return NormalizeFinancialStatements(raw), nil
}
