package kis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Client struct {
	httpClient *http.Client
	appKey     string
	appSecret  string
	baseURL    string

	mu          sync.Mutex
	cachedToken *AuthToken
}

func NewClient(appKey, appSecret, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://openapi.koreainvestment.com:9443"
	}
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		appKey:     appKey,
		appSecret:  appSecret,
		baseURL:    baseURL,
	}
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cachedToken != nil && time.Now().Before(c.cachedToken.ExpiresAt.Add(-time.Minute)) {
		return c.cachedToken.AccessToken, nil
	}

	slog.Info("KIS: refreshing access token")

	body := fmt.Sprintf(`{"grant_type":"client_credentials","appkey":"%s","appsecret":"%s"}`,
		c.appKey, c.appSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/oauth2/tokenP", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.ExpiresIn == 0 {
		tokenResp.ExpiresIn = 86400
	}

	c.cachedToken = &AuthToken{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	return c.cachedToken.AccessToken, nil
}

func (c *Client) authHeaders(token string) http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("authorization", "Bearer "+token)
	h.Set("appkey", c.appKey)
	h.Set("appsecret", c.appSecret)
	return h
}

func (c *Client) GetCandles(ctx context.Context, symbol, startDate, endDate string) ([]NormalizedCandle, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("FID_COND_MRKT_DIV_CODE", "J")
	params.Set("FID_INPUT_ISCD", symbol)
	params.Set("FID_INPUT_DATE_1", startDate)
	params.Set("FID_INPUT_DATE_2", endDate)
	params.Set("FID_PERIOD_DIV_CODE", "D")
	params.Set("FID_ORG_ADJ_PRC", "0")

	reqURL := fmt.Sprintf("%s/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice?%s",
		c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create candle request: %w", err)
	}
	req.Header = c.authHeaders(token)
	req.Header.Set("tr_id", "FHKST03010100")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("candle request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read candle response: %w", err)
	}

	var candleResp CandleResponse
	if err := json.Unmarshal(respBody, &candleResp); err != nil {
		return nil, fmt.Errorf("decode candle response: %w", err)
	}

	return NormalizeKISCandles(candleResp.Output2), nil
}

func (c *Client) GetCurrentPrice(ctx context.Context, symbol string) (*KISPriceResponse, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("FID_COND_MRKT_DIV_CODE", "J")
	params.Set("FID_INPUT_ISCD", symbol)

	reqURL := fmt.Sprintf("%s/uapi/domestic-stock/v1/quotations/inquire-price?%s",
		c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create price request: %w", err)
	}
	req.Header = c.authHeaders(token)
	req.Header.Set("tr_id", "FHKST01010100")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("price request failed: %w", err)
	}
	defer resp.Body.Close()

	var priceResp PriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
		return nil, fmt.Errorf("decode price response: %w", err)
	}

	return priceResp.Output, nil
}
