package dsl

import (
	"testing"
)

func TestLexer_SimpleScanQuery(t *testing.T) {
	tokens, err := Tokenize("scan where volume > 1000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []TokenType{TOKEN_SCAN, TOKEN_WHERE, TOKEN_IDENTIFIER, TOKEN_GT, TOKEN_NUMBER, TOKEN_EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, tt := range expected {
		if tokens[i].Type != tt {
			t.Errorf("token[%d]: expected type %d, got %d (value=%q)", i, tt, tokens[i].Type, tokens[i].Value)
		}
	}

	if tokens[2].Value != "volume" {
		t.Errorf("expected identifier 'volume', got %q", tokens[2].Value)
	}
	if tokens[4].Value != "1000000" {
		t.Errorf("expected number '1000000', got %q", tokens[4].Value)
	}
}

func TestLexer_ComparisonOperators(t *testing.T) {
	tokens, err := Tokenize("close >= 50000 and rsi <= 30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []TokenType{TOKEN_IDENTIFIER, TOKEN_GTE, TOKEN_NUMBER, TOKEN_AND, TOKEN_IDENTIFIER, TOKEN_LTE, TOKEN_NUMBER, TOKEN_EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, tt := range expected {
		if tokens[i].Type != tt {
			t.Errorf("token[%d]: expected type %d, got %d (value=%q)", i, tt, tokens[i].Type, tokens[i].Value)
		}
	}
}

func TestLexer_AssignVsEquality(t *testing.T) {
	tokens, err := Tokenize("success = close >= event_high * 2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// success IDENTIFIER, = ASSIGN, close IDENTIFIER, >= GTE, event_high IDENTIFIER, * STAR, 2.0 NUMBER, EOF
	expected := []TokenType{TOKEN_IDENTIFIER, TOKEN_ASSIGN, TOKEN_IDENTIFIER, TOKEN_GTE, TOKEN_IDENTIFIER, TOKEN_STAR, TOKEN_NUMBER, TOKEN_EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, tt := range expected {
		if tokens[i].Type != tt {
			t.Errorf("token[%d]: expected type %d, got %d (value=%q)", i, tt, tokens[i].Type, tokens[i].Value)
		}
	}

	// Verify ASSIGN (not EQ)
	if tokens[1].Type != TOKEN_ASSIGN {
		t.Error("expected ASSIGN token for '='")
	}
	// Verify GTE
	if tokens[3].Type != TOKEN_GTE {
		t.Error("expected GTE token for '>='")
	}
}

func TestLexer_FunctionCall(t *testing.T) {
	tokens, err := Tokenize("pre_event_ma(120)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []TokenType{TOKEN_IDENTIFIER, TOKEN_LPAREN, TOKEN_NUMBER, TOKEN_RPAREN, TOKEN_EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, tt := range expected {
		if tokens[i].Type != tt {
			t.Errorf("token[%d]: expected type %d, got %d (value=%q)", i, tt, tokens[i].Type, tokens[i].Value)
		}
	}

	if tokens[0].Value != "pre_event_ma" {
		t.Errorf("expected identifier 'pre_event_ma', got %q", tokens[0].Value)
	}
}

func TestLexer_FloatNumbers(t *testing.T) {
	tokens, err := Tokenize("event_high * 2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// event_high IDENTIFIER, * STAR, 2.0 NUMBER, EOF
	if len(tokens) != 4 {
		t.Fatalf("expected 4 tokens, got %d", len(tokens))
	}
	if tokens[2].Type != TOKEN_NUMBER {
		t.Errorf("expected NUMBER token, got type %d", tokens[2].Type)
	}
	if tokens[2].Value != "2.0" {
		t.Errorf("expected number value '2.0', got %q", tokens[2].Value)
	}
}

func TestLexer_SortByClause(t *testing.T) {
	tokens, err := Tokenize("sort by volume desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []TokenType{TOKEN_SORT, TOKEN_BY, TOKEN_IDENTIFIER, TOKEN_DESC, TOKEN_EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, tt := range expected {
		if tokens[i].Type != tt {
			t.Errorf("token[%d]: expected type %d, got %d (value=%q)", i, tt, tokens[i].Type, tokens[i].Value)
		}
	}
}

func TestLexer_LineAndColumn(t *testing.T) {
	tokens, err := Tokenize("scan\nwhere x > 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// scan (line 1), \n, where (line 2), x (line 2), > (line 2), 1 (line 2), EOF
	// After filtering, we check the raw tokens
	scanTok := tokens[0]
	if scanTok.Line != 1 || scanTok.Column != 1 {
		t.Errorf("scan: expected line 1 col 1, got line %d col %d", scanTok.Line, scanTok.Column)
	}

	// Find 'where' token (should be at index 2 after newline at index 1)
	whereTok := tokens[2]
	if whereTok.Type != TOKEN_WHERE {
		t.Fatalf("expected WHERE token at index 2, got type %d value %q", whereTok.Type, whereTok.Value)
	}
	if whereTok.Line != 2 {
		t.Errorf("where: expected line 2, got line %d", whereTok.Line)
	}
	if whereTok.Column != 1 {
		t.Errorf("where: expected column 1, got column %d", whereTok.Column)
	}

	// x should be on line 2
	xTok := tokens[3]
	if xTok.Line != 2 {
		t.Errorf("x: expected line 2, got line %d", xTok.Line)
	}
}

func TestLexer_UnexpectedCharacter(t *testing.T) {
	_, err := Tokenize("scan @ where")
	if err == nil {
		t.Fatal("expected error for unexpected character '@', got nil")
	}
}
