package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/dev-superbear/nexus-backend/internal/dsl"
)

type Server struct {
	executor *dsl.Executor
}

func NewServer(executor *dsl.Executor) *Server {
	return &Server{executor: executor}
}

func (s *Server) HandleGetDSLGrammar() string {
	return `DSL Grammar:
  scan where <conditions> [sort by <field> [asc|desc]] [limit N]

Conditions:
  <field> <operator> <value>
  Multiple conditions joined with AND (OR is NOT supported)

Operators: >, <, >=, <=, =

Defaults:
  limit: 50 (max: 500)
  sort: volume DESC

Example:
  scan where volume > 1000000 and close > 50000 sort by trade_value desc limit 20`
}

func (s *Server) HandleListAvailableFields() string {
	return `Available fields:
  close       — 종가/현재가 (numeric, KRW)
  open        — 시가 (numeric, KRW)
  high        — 고가 (numeric, KRW)
  low         — 저가 (numeric, KRW)
  volume      — 거래량 (integer, shares)
  trade_value — 거래대금 (numeric, close × volume)
  change_pct  — 전일 대비 등락률 (numeric, %)

All fields support operators: >, <, >=, <=, =
All fields can be used in sort by clause.`
}

func (s *Server) HandleValidateDSL(dslCode string) (string, error) {
	result := s.executor.Validate(dslCode)
	if !result.Valid {
		return "", fmt.Errorf("%s", result.Error)
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}
