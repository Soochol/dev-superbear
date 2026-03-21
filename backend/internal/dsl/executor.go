package dsl

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SearchResult struct {
	Symbol       string   `json:"symbol"`
	Name         string   `json:"name"`
	MatchedValue any      `json:"matchedValue"`
	Close        *float64 `json:"close,omitempty"`
	Volume       *int64   `json:"volume,omitempty"`
	TradeValue   *float64 `json:"tradeValue,omitempty"`
	Change       *float64 `json:"change,omitempty"`
	ChangePct    *float64 `json:"changePct,omitempty"`
}

type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

type SortSpec struct {
	Field     string        `json:"field"`
	Direction SortDirection `json:"direction"`
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

type Executor struct {
	pool *pgxpool.Pool
}

func NewExecutor(pool ...*pgxpool.Pool) *Executor {
	e := &Executor{}
	if len(pool) > 0 {
		e.pool = pool[0]
	}
	return e
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
	ast, err := e.parse(dslCode)
	if err != nil {
		return nil, fmt.Errorf("invalid DSL: %w", err)
	}
	if ast.Type != "ScanQuery" {
		return nil, fmt.Errorf("not a valid scan query")
	}
	if e.pool == nil {
		return []SearchResult{}, nil
	}
	return e.executeQuery(ctx, ast)
}

// FieldInfo describes a single DSL field with its SQL mapping and metadata.
type FieldInfo struct {
	Name        string `json:"name"`
	SQL         string `json:"-"`
	Description string `json:"description"`
	Unit        string `json:"unit"`
}

// fieldRegistry is the single source of truth for all DSL fields.
// Order is preserved for deterministic output.
var fieldRegistry = []FieldInfo{
	{Name: "close", SQL: "d.close", Description: "종가/현재가", Unit: "numeric, KRW"},
	{Name: "open", SQL: "d.open", Description: "시가", Unit: "numeric, KRW"},
	{Name: "high", SQL: "d.high", Description: "고가", Unit: "numeric, KRW"},
	{Name: "low", SQL: "d.low", Description: "저가", Unit: "numeric, KRW"},
	{Name: "volume", SQL: "d.volume", Description: "거래량", Unit: "integer, shares"},
	{Name: "trade_value", SQL: "(d.close * d.volume)", Description: "거래대금", Unit: "numeric, close × volume"},
	{Name: "change_pct", SQL: "CASE WHEN prev.close > 0 THEN ((d.close - prev.close)::float / prev.close * 100) ELSE 0 END", Description: "전일 대비 등락률", Unit: "numeric, %"},
}

// allowedFields is built from fieldRegistry at init time.
var allowedFields map[string]string

var allowedOps = []string{">", "<", ">=", "<=", "="}
var allowedOpsMap map[string]string

func init() {
	allowedFields = make(map[string]string, len(fieldRegistry))
	for _, f := range fieldRegistry {
		allowedFields[f.Name] = f.SQL
	}
	allowedOpsMap = make(map[string]string, len(allowedOps))
	for _, op := range allowedOps {
		allowedOpsMap[op] = op
	}
}

// AvailableFields returns the field registry (ordered).
func (e *Executor) AvailableFields() []FieldInfo {
	return fieldRegistry
}

// AllowedOps returns the list of supported comparison operators.
func (e *Executor) AllowedOps() []string {
	return allowedOps
}

// GrammarText returns the DSL grammar description, generated from the registry.
func (e *Executor) GrammarText() string {
	fieldNames := make([]string, len(fieldRegistry))
	for i, f := range fieldRegistry {
		fieldNames[i] = f.Name
	}
	return fmt.Sprintf(`DSL Grammar:
  scan where <conditions> [sort by <field> [asc|desc]] [limit N]

Conditions:
  <field> <operator> <value>
  Multiple conditions joined with AND (OR is NOT supported)

Fields: %s

Operators: %s

Defaults:
  limit: 50 (max: 500)
  sort: volume DESC

Example:
  scan where volume > 1000000 and close > 50000 sort by trade_value desc limit 20`,
		strings.Join(fieldNames, ", "),
		strings.Join(allowedOps, ", "))
}

func (e *Executor) executeQuery(ctx context.Context, ast *parsedAST) ([]SearchResult, error) {
	// Build parameterized WHERE clause
	sqlWhere, args, err := e.buildWhere(ast.Conditions)
	if err != nil {
		return nil, err
	}

	needsPrev := false
	for _, c := range ast.Conditions {
		if c.Field == "change_pct" {
			needsPrev = true
			break
		}
	}
	if ast.SortBy != nil && ast.SortBy.Field == "change_pct" {
		needsPrev = true
	}

	// Query latest date's data per stock
	prevJoin := ""
	if needsPrev {
		prevJoin = `LEFT JOIN LATERAL (
			SELECT close FROM daily_candles
			WHERE symbol = d.symbol AND date < d.date
			ORDER BY date DESC LIMIT 1
		) prev ON true`
	} else {
		prevJoin = "LEFT JOIN LATERAL (SELECT 0::bigint AS close) prev ON true"
	}

	orderClause := "d.volume DESC"
	if ast.SortBy != nil {
		sortCol, ok := allowedFields[ast.SortBy.Field]
		if ok {
			dir := "DESC"
			if ast.SortBy.Direction == SortAsc {
				dir = "ASC"
			}
			orderClause = sortCol + " " + dir
		}
	}

	limit := ast.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}

	query := fmt.Sprintf(`
		WITH latest AS (
			SELECT DISTINCT ON (symbol) symbol, date, open, high, low, close, volume
			FROM daily_candles
			WHERE date >= (SELECT MAX(date) - interval '7 days' FROM daily_candles)
			ORDER BY symbol, date DESC
		)
		SELECT s.symbol, s.name, d.close, d.volume,
			   (d.close * d.volume) AS trade_value,
			   CASE WHEN prev.close > 0 THEN ((d.close - prev.close)::float / prev.close * 100) ELSE 0 END AS change_pct
		FROM latest d
		JOIN stocks s ON s.symbol = d.symbol
		%s
		WHERE %s
		ORDER BY %s
		LIMIT $%d
	`, prevJoin, sqlWhere, orderClause, len(args)+1)

	args = append(args, limit)

	rows, err := e.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var close, vol, tv int64
		var changePct float64
		if err := rows.Scan(&r.Symbol, &r.Name, &close, &vol, &tv, &changePct); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		closeF := float64(close)
		tvF := float64(tv)
		r.Close = &closeF
		r.Volume = &vol
		r.TradeValue = &tvF
		r.ChangePct = &changePct
		r.MatchedValue = vol
		results = append(results, r)
	}
	return results, nil
}

func (e *Executor) buildWhere(conditions []condition) (string, []any, error) {
	if len(conditions) == 0 {
		return "1=1", nil, nil
	}
	var parts []string
	var args []any
	for _, c := range conditions {
		col, ok := allowedFields[c.Field]
		if !ok {
			return "", nil, fmt.Errorf("unknown field: %s", c.Field)
		}
		op, ok := allowedOpsMap[c.Op]
		if !ok {
			return "", nil, fmt.Errorf("unknown operator: %s", c.Op)
		}
		args = append(args, c.Value)
		parts = append(parts, fmt.Sprintf("%s %s $%d", col, op, len(args)))
	}
	return strings.Join(parts, " AND "), args, nil
}

// --- Parser ---

type condition struct {
	Field string
	Op    string
	Value float64
}

type parsedAST struct {
	Type       string
	Conditions []condition
	SortBy     *SortSpec
	Limit      int
	// kept for backward compat
	WhereClause string
}

var scanRe = regexp.MustCompile(`(?i)^scan\s+where\s+(.+)$`)
var condRe = regexp.MustCompile(`(?i)^(\w+)\s*(>=|<=|>|<|=)\s*([\d.]+)$`)
var sortRe = regexp.MustCompile(`(?i)\s+sort\s+by\s+(\w+)(?:\s+(asc|desc))?\s*$`)
var limitRe = regexp.MustCompile(`(?i)\s+limit\s+(\d+)\s*$`)

func (e *Executor) parse(input string) (*parsedAST, error) {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	m := scanRe.FindStringSubmatch(input)
	if m == nil {
		return nil, fmt.Errorf("expected format: scan where <conditions>")
	}

	rest := m[1]
	ast := &parsedAST{Type: "ScanQuery", Limit: 50}

	// Extract limit
	if lm := limitRe.FindStringSubmatch(rest); lm != nil {
		n, _ := strconv.Atoi(lm[1])
		ast.Limit = n
		rest = rest[:len(rest)-len(lm[0])]
	}

	// Extract sort
	if sm := sortRe.FindStringSubmatch(rest); sm != nil {
		dir := SortDesc
		if strings.EqualFold(sm[2], "asc") {
			dir = SortAsc
		}
		ast.SortBy = &SortSpec{Field: strings.ToLower(sm[1]), Direction: dir}
		rest = rest[:len(rest)-len(sm[0])]
	}

	// Parse conditions (AND-separated)
	ast.WhereClause = strings.TrimSpace(rest)
	parts := regexp.MustCompile(`(?i)\s+and\s+`).Split(rest, -1)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		cm := condRe.FindStringSubmatch(p)
		if cm == nil {
			return nil, fmt.Errorf("invalid condition: %s", p)
		}
		val, err := strconv.ParseFloat(cm[3], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number: %s", cm[3])
		}
		ast.Conditions = append(ast.Conditions, condition{
			Field: strings.ToLower(cm[1]),
			Op:    cm[2],
			Value: val,
		})
	}

	return ast, nil
}
