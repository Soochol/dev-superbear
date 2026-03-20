package portfolio

import "time"

// ---------------------------------------------------------------------------
// Portfolio Position (DB-mapped entity)
// ---------------------------------------------------------------------------

// PortfolioPosition represents a user's holding of a single symbol.
type PortfolioPosition struct {
	ID           string
	UserID       string
	Symbol       string
	SymbolName   string
	Market       Market
	Quantity     int
	AvgCostPrice float64
	TotalCost    float64
	Sector       *string
	SectorName   *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ---------------------------------------------------------------------------
// Realized PnL record
// ---------------------------------------------------------------------------

// RealizedPnL records profit/loss from a partial or full lot liquidation.
type RealizedPnL struct {
	ID          string
	PositionID  string
	SellTradeID string
	BuyLotID    string
	Quantity    int
	BuyPrice    float64
	SellPrice   float64
	GrossPnL    float64
	Fee         float64
	Tax         float64
	NetPnL      float64
	RealizedAt  time.Time
}

// ---------------------------------------------------------------------------
// Summary / API response types
// ---------------------------------------------------------------------------

// PortfolioSummary is the top-level portfolio overview returned by the API.
type PortfolioSummary struct {
	TotalValue       float64          `json:"totalValue"`
	TotalCost        float64          `json:"totalCost"`
	UnrealizedPnL    float64          `json:"unrealizedPnl"`
	UnrealizedPnLPct float64          `json:"unrealizedPnlPct"`
	RealizedPnL      float64          `json:"realizedPnl"`
	TotalPnL         float64          `json:"totalPnl"`
	Positions        []PositionDetail `json:"positions"`
}

// PositionDetail carries enriched per-symbol data (with live price).
type PositionDetail struct {
	Symbol          string  `json:"symbol"`
	SymbolName      string  `json:"symbolName"`
	Market          Market  `json:"market"`
	Quantity        int     `json:"quantity"`
	AvgCostPrice    float64 `json:"avgCostPrice"`
	CurrentPrice    float64 `json:"currentPrice"`
	TotalCost       float64 `json:"totalCost"`
	TotalValue      float64 `json:"totalValue"`
	UnrealizedPnL   float64 `json:"unrealizedPnl"`
	UnrealizedPnLPct float64 `json:"unrealizedPnlPct"`
	Sector          *string `json:"sector,omitempty"`
	SectorName      *string `json:"sectorName,omitempty"`
	Weight          float64 `json:"weight"` // portfolio weight %
}

// ---------------------------------------------------------------------------
// Sector analysis
// ---------------------------------------------------------------------------

// SectorPosition represents one symbol within a sector.
type SectorPosition struct {
	Symbol     string  `json:"symbol"`
	SymbolName string  `json:"symbolName"`
	Value      float64 `json:"value"`
	Weight     float64 `json:"weight"`
}

// SectorWeight aggregates a sector's holdings.
type SectorWeight struct {
	Sector          string           `json:"sector"`
	SectorName      string           `json:"sectorName"`
	TotalValue      float64          `json:"totalValue"`
	Weight          float64          `json:"weight"`
	Positions       []SectorPosition `json:"positions"`
	UnrealizedPnL   float64          `json:"unrealizedPnl"`
	UnrealizedPnLPct float64         `json:"unrealizedPnlPct"`
}

// ---------------------------------------------------------------------------
// Trade sync input types
// ---------------------------------------------------------------------------

// BuyTradeInput carries parameters to record a buy into the portfolio.
type BuyTradeInput struct {
	UserID     string
	TradeID    string
	Symbol     string
	SymbolName string
	Price      float64
	Quantity   int
	Fee        float64
	Market     Market
	Sector     *string
	SectorName *string
}

// SellTradeInput carries parameters to record a sell from the portfolio.
type SellTradeInput struct {
	UserID   string
	TradeID  string
	Symbol   string
	Price    float64
	Quantity int
	Fee      float64
}

// ---------------------------------------------------------------------------
// Tax simulation
// ---------------------------------------------------------------------------

// TaxSimulation is the response for the annual tax simulation endpoint.
type TaxSimulation struct {
	KR       KRTaxSummary `json:"kr"`
	US       USTaxSummary `json:"us"`
	TotalTax float64      `json:"totalTax"`
}

// KRTaxSummary holds domestic tax totals.
type KRTaxSummary struct {
	TotalSellAmount float64 `json:"totalSellAmount"`
	TransactionTax  float64 `json:"transactionTax"`
}

// USTaxSummary holds overseas tax totals.
type USTaxSummary struct {
	TotalGain       float64 `json:"totalGain"`
	BasicDeduction  float64 `json:"basicDeduction"`
	TaxableGain     float64 `json:"taxableGain"`
	CapitalGainsTax float64 `json:"capitalGainsTax"`
}

// ---------------------------------------------------------------------------
// PnL history
// ---------------------------------------------------------------------------

// PnLHistoryEntry represents a single data point in the PnL history.
type PnLHistoryEntry struct {
	Date          string  `json:"date"`
	RealizedPnL   float64 `json:"realizedPnl"`
	CumulativePnL float64 `json:"cumulativePnl"`
}
