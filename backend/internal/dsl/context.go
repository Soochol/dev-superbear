package dsl

import (
	"fmt"
	"math"
)

// EventContext holds the market data fields available for DSL evaluation.
type EventContext struct {
	Close, High, Low, Volume, TradeValue                       float64
	EventHigh, EventLow, EventClose, EventVolume               float64
	PreEventClose, PostHigh, PostLow, DaysSinceEvent           float64
}

// FunctionRegistry maps function names to their Go implementations.
type FunctionRegistry map[string]func(args ...float64) (float64, error)

// EvalContext holds variables and functions available during evaluation.
type EvalContext struct {
	Variables map[string]interface{}
	Functions FunctionRegistry
}

// NewEventEvalContext creates an EvalContext populated with event-related
// variables and built-in functions (pre_event_ma, max, min, abs).
func NewEventEvalContext(event EventContext, preEventMA map[int]float64) *EvalContext {
	vars := map[string]interface{}{
		"close":            event.Close,
		"high":             event.High,
		"low":              event.Low,
		"volume":           event.Volume,
		"trade_value":      event.TradeValue,
		"event_high":       event.EventHigh,
		"event_low":        event.EventLow,
		"event_close":      event.EventClose,
		"event_volume":     event.EventVolume,
		"pre_event_close":  event.PreEventClose,
		"post_high":        event.PostHigh,
		"post_low":         event.PostLow,
		"days_since_event": event.DaysSinceEvent,
	}

	funcs := FunctionRegistry{
		"pre_event_ma": func(args ...float64) (float64, error) {
			if len(args) != 1 {
				return 0, fmt.Errorf("pre_event_ma expects 1 argument, got %d", len(args))
			}
			val, ok := preEventMA[int(args[0])]
			if !ok {
				return 0, fmt.Errorf("pre_event_ma(%d) not available", int(args[0]))
			}
			return val, nil
		},
		"max": func(args ...float64) (float64, error) {
			if len(args) != 2 {
				return 0, fmt.Errorf("max expects 2 arguments")
			}
			return math.Max(args[0], args[1]), nil
		},
		"min": func(args ...float64) (float64, error) {
			if len(args) != 2 {
				return 0, fmt.Errorf("min expects 2 arguments")
			}
			return math.Min(args[0], args[1]), nil
		},
		"abs": func(args ...float64) (float64, error) {
			if len(args) != 1 {
				return 0, fmt.Errorf("abs expects 1 argument")
			}
			return math.Abs(args[0]), nil
		},
	}

	return &EvalContext{Variables: vars, Functions: funcs}
}
