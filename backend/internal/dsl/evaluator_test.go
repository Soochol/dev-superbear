package dsl

import (
	"testing"
)

func newTestCtx() *EvalContext {
	event := EventContext{
		Close:          78000,
		High:           80000,
		Low:            75000,
		Volume:         5000000,
		TradeValue:     390000000000,
		EventHigh:      40000,
		EventLow:       38000,
		EventClose:     39000,
		EventVolume:    3000000,
		PreEventClose:  37000,
		PostHigh:       82000,
		PostLow:        36000,
		DaysSinceEvent: 180,
	}
	preEventMA := map[int]float64{
		5:   38000,
		20:  36000,
		60:  35000,
		120: 34000,
	}
	return NewEventEvalContext(event, preEventMA)
}

func TestEvaluator_SuccessCondition(t *testing.T) {
	// success = close >= event_high * 2.0
	// 78000 >= 40000 * 2.0 → 78000 >= 80000 → false
	ctx := newTestCtx()
	result, err := EvaluateDSL("success = close >= event_high * 2.0", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, ok := result.(bool)
	if !ok {
		t.Fatalf("expected bool, got %T (%v)", result, result)
	}
	if b != false {
		t.Errorf("expected false (78000 < 80000), got true")
	}
}

func TestEvaluator_FailureCondition(t *testing.T) {
	// failure = close < pre_event_ma(120)
	// 78000 < 34000 → false
	ctx := newTestCtx()
	result, err := EvaluateDSL("failure = close < pre_event_ma(120)", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, ok := result.(bool)
	if !ok {
		t.Fatalf("expected bool, got %T (%v)", result, result)
	}
	if b != false {
		t.Errorf("expected false (78000 > 34000), got true")
	}
}

func TestEvaluator_Arithmetic(t *testing.T) {
	// result = event_high * 2.0 + 1000
	// 40000 * 2.0 + 1000 = 81000
	ctx := newTestCtx()
	result, err := EvaluateDSL("result = event_high * 2.0 + 1000", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	num, ok := result.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T (%v)", result, result)
	}
	if num != 81000 {
		t.Errorf("expected 81000, got %f", num)
	}
}

func TestEvaluator_BooleanStandalone(t *testing.T) {
	// close > 70000 → true (78000 > 70000)
	ctx := newTestCtx()
	result, err := EvaluateDSL("close > 70000", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, ok := result.(bool)
	if !ok {
		t.Fatalf("expected bool, got %T (%v)", result, result)
	}
	if b != true {
		t.Errorf("expected true (78000 > 70000), got false")
	}
}

func TestEvaluator_AndOrLogic(t *testing.T) {
	// close > 70000 and days_since_event > 100 → true and true → true
	ctx := newTestCtx()
	result, err := EvaluateDSL("close > 70000 and days_since_event > 100", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, ok := result.(bool)
	if !ok {
		t.Fatalf("expected bool, got %T (%v)", result, result)
	}
	if b != true {
		t.Errorf("expected true, got false")
	}
}

func TestEvaluator_UndefinedVariable(t *testing.T) {
	ctx := newTestCtx()
	_, err := EvaluateDSL("undefined_var > 100", ctx)
	if err == nil {
		t.Fatal("expected error for undefined variable, got nil")
	}
}
