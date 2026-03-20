package dsl

import (
	"context"
	"fmt"
)

type SearchResult struct {
	Symbol       string      `json:"symbol"`
	Name         string      `json:"name"`
	MatchedValue any `json:"matchedValue"`
	Close        *float64    `json:"close,omitempty"`
	Volume       *int64      `json:"volume,omitempty"`
	TradeValue   *float64    `json:"tradeValue,omitempty"`
	Change       *float64    `json:"change,omitempty"`
	ChangePct    *float64    `json:"changePct,omitempty"`
}

type SortSpec struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type ParsedScanQuery struct {
	WhereClause string    `json:"whereClause"`
	SortBy      *SortSpec `json:"sortBy,omitempty"`
	Limit       int       `json:"limit"`
}

type ValidationResult struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Validate(input string) ValidationResult {
	if len(input) == 0 {
		return ValidationResult{Valid: false, Error: "empty DSL input"}
	}
	_, err := e.parse(input)
	if err != nil {
		return ValidationResult{Valid: false, Error: err.Error()}
	}
	return ValidationResult{Valid: true}
}

func (e *Executor) ParseScan(input string) (*ParsedScanQuery, error) {
	ast, err := e.parse(input)
	if err != nil {
		return nil, err
	}
	if ast.Type != "ScanQuery" {
		return nil, nil
	}
	return &ParsedScanQuery{
		WhereClause: ast.WhereClause,
		SortBy:      ast.SortBy,
		Limit:       ast.Limit,
	}, nil
}

func (e *Executor) Execute(ctx context.Context, dslCode string) ([]SearchResult, error) {
	parsed, err := e.ParseScan(dslCode)
	if err != nil {
		return nil, fmt.Errorf("invalid DSL: %w", err)
	}
	if parsed == nil {
		return nil, fmt.Errorf("not a valid scan query")
	}
	// TODO: Fetch stock data and filter by DSL conditions
	return []SearchResult{}, nil
}

type parsedAST struct {
	Type        string
	WhereClause string
	SortBy      *SortSpec
	Limit       int
}

func (e *Executor) parse(input string) (*parsedAST, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("empty input")
	}
	// Placeholder - will be replaced with actual DSL parser
	return &parsedAST{Type: "ScanQuery", Limit: 100}, nil
}
