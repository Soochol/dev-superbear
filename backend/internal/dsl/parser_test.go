package dsl

import (
	"testing"
)

func TestParser_ScanWhereQuery(t *testing.T) {
	node, err := ParseDSL("scan where volume > 1000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sq, ok := node.(*ScanQuery)
	if !ok {
		t.Fatalf("expected *ScanQuery, got %T", node)
	}

	bin, ok := sq.Where.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr for where clause, got %T", sq.Where)
	}

	if bin.Operator != ">" {
		t.Errorf("expected operator '>', got %q", bin.Operator)
	}

	left, ok := bin.Left.(*Identifier)
	if !ok {
		t.Fatalf("expected *Identifier on left, got %T", bin.Left)
	}
	if left.Name != "volume" {
		t.Errorf("expected 'volume', got %q", left.Name)
	}

	right, ok := bin.Right.(*NumberLiteral)
	if !ok {
		t.Fatalf("expected *NumberLiteral on right, got %T", bin.Right)
	}
	if right.Value != 1000000 {
		t.Errorf("expected 1000000, got %f", right.Value)
	}
}

func TestParser_ScanWhereSortLimit(t *testing.T) {
	node, err := ParseDSL("scan where volume > 1000000 sort by trade_value desc limit 50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sq, ok := node.(*ScanQuery)
	if !ok {
		t.Fatalf("expected *ScanQuery, got %T", node)
	}

	if sq.SortBy == nil {
		t.Fatal("expected SortBy clause")
	}
	if sq.SortBy.Field != "trade_value" {
		t.Errorf("expected sort field 'trade_value', got %q", sq.SortBy.Field)
	}
	if sq.SortBy.Direction != "desc" {
		t.Errorf("expected sort direction 'desc', got %q", sq.SortBy.Direction)
	}

	if sq.Limit == nil {
		t.Fatal("expected Limit clause")
	}
	if *sq.Limit != 50 {
		t.Errorf("expected limit 50, got %d", *sq.Limit)
	}
}

func TestParser_AssignmentExpr(t *testing.T) {
	node, err := ParseDSL("success = close >= event_high * 2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assign, ok := node.(*AssignmentExpr)
	if !ok {
		t.Fatalf("expected *AssignmentExpr, got %T", node)
	}

	if assign.Name != "success" {
		t.Errorf("expected name 'success', got %q", assign.Name)
	}

	// The value should be a BinaryExpr with >=
	bin, ok := assign.Value.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr for value, got %T", assign.Value)
	}
	if bin.Operator != ">=" {
		t.Errorf("expected operator '>=', got %q", bin.Operator)
	}
}

func TestParser_OperatorPrecedence(t *testing.T) {
	// success = close >= event_high * 2.0
	// Precedence: * binds tighter than >=
	// So: close >= (event_high * 2.0)
	node, err := ParseDSL("success = close >= event_high * 2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assign := node.(*AssignmentExpr)
	bin := assign.Value.(*BinaryExpr)

	// Left should be Identifier "close"
	left, ok := bin.Left.(*Identifier)
	if !ok {
		t.Fatalf("expected *Identifier on left of >=, got %T", bin.Left)
	}
	if left.Name != "close" {
		t.Errorf("expected 'close', got %q", left.Name)
	}

	// Right should be BinaryExpr with *
	right, ok := bin.Right.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr on right of >=, got %T", bin.Right)
	}
	if right.Operator != "*" {
		t.Errorf("expected operator '*' on right side, got %q", right.Operator)
	}

	// Right.Left should be Identifier "event_high"
	rLeft, ok := right.Left.(*Identifier)
	if !ok {
		t.Fatalf("expected *Identifier for event_high, got %T", right.Left)
	}
	if rLeft.Name != "event_high" {
		t.Errorf("expected 'event_high', got %q", rLeft.Name)
	}

	// Right.Right should be NumberLiteral 2.0
	rRight, ok := right.Right.(*NumberLiteral)
	if !ok {
		t.Fatalf("expected *NumberLiteral for 2.0, got %T", right.Right)
	}
	if rRight.Value != 2.0 {
		t.Errorf("expected 2.0, got %f", rRight.Value)
	}
}

func TestParser_FunctionCall(t *testing.T) {
	node, err := ParseDSL("failure = close < pre_event_ma(120)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assign, ok := node.(*AssignmentExpr)
	if !ok {
		t.Fatalf("expected *AssignmentExpr, got %T", node)
	}
	if assign.Name != "failure" {
		t.Errorf("expected name 'failure', got %q", assign.Name)
	}

	bin, ok := assign.Value.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr, got %T", assign.Value)
	}
	if bin.Operator != "<" {
		t.Errorf("expected operator '<', got %q", bin.Operator)
	}

	fc, ok := bin.Right.(*FunctionCall)
	if !ok {
		t.Fatalf("expected *FunctionCall on right, got %T", bin.Right)
	}
	if fc.Name != "pre_event_ma" {
		t.Errorf("expected function name 'pre_event_ma', got %q", fc.Name)
	}
	if len(fc.Args) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(fc.Args))
	}
	arg, ok := fc.Args[0].(*NumberLiteral)
	if !ok {
		t.Fatalf("expected *NumberLiteral arg, got %T", fc.Args[0])
	}
	if arg.Value != 120 {
		t.Errorf("expected 120, got %f", arg.Value)
	}
}

func TestParser_AndOrLogic(t *testing.T) {
	node, err := ParseDSL("scan where volume > 1000000 and close > 50000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sq, ok := node.(*ScanQuery)
	if !ok {
		t.Fatalf("expected *ScanQuery, got %T", node)
	}

	bin, ok := sq.Where.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr for where, got %T", sq.Where)
	}
	if bin.Operator != "and" {
		t.Errorf("expected operator 'and', got %q", bin.Operator)
	}

	// Left: volume > 1000000
	leftBin, ok := bin.Left.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr on left, got %T", bin.Left)
	}
	if leftBin.Operator != ">" {
		t.Errorf("expected '>' on left, got %q", leftBin.Operator)
	}

	// Right: close > 50000
	rightBin, ok := bin.Right.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr on right, got %T", bin.Right)
	}
	if rightBin.Operator != ">" {
		t.Errorf("expected '>' on right, got %q", rightBin.Operator)
	}
}

func TestParser_SyntaxError(t *testing.T) {
	_, err := ParseDSL("scan where >")
	if err == nil {
		t.Fatal("expected syntax error, got nil")
	}
}
