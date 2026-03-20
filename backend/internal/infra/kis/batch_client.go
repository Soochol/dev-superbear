package kis

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const (
	batchSize    = 20   // KIS API rate limit 고려
	batchDelayMs = 1000 // 배치 간 간격
)

// PriceSnapshot holds a single symbol's latest market data.
type PriceSnapshot struct {
	Symbol    string
	Close     float64
	High      float64
	Low       float64
	Volume    float64
	Timestamp time.Time
}

// FetchPricesBatch retrieves current prices for the given symbols in batches
// to respect KIS API rate limits. Returns a map keyed by symbol.
func FetchPricesBatch(symbols []string) (map[string]*PriceSnapshot, error) {
	results := make(map[string]*PriceSnapshot)
	var mu sync.Mutex

	batches := chunk(symbols, batchSize)
	for i, batch := range batches {
		var wg sync.WaitGroup
		for _, sym := range batch {
			wg.Add(1)
			go func(symbol string) {
				defer wg.Done()
				price, err := getPrice(symbol)
				if err != nil {
					slog.Error("failed to fetch price", "symbol", symbol, "error", err)
					return
				}
				mu.Lock()
				results[symbol] = price
				mu.Unlock()
			}(sym)
		}
		wg.Wait()
		if i < len(batches)-1 {
			time.Sleep(time.Duration(batchDelayMs) * time.Millisecond)
		}
	}
	return results, nil
}

// getPrice fetches the current price for a single symbol from KIS API.
func getPrice(symbol string) (*PriceSnapshot, error) {
	// TODO: KIS API 클라이언트 연동 (Plan 1 shared infra)
	return nil, fmt.Errorf("KIS API client not yet implemented for symbol %s", symbol)
}

func chunk(items []string, size int) [][]string {
	var chunks [][]string
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}
	return chunks
}
