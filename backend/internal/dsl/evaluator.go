package dsl

import "fmt"

// Evaluate recursively evaluates an AST node against an EvalContext.
// Returns float64, bool, or string depending on the expression.
func Evaluate(node Node, ctx *EvalContext) (interface{}, error) {
	switch n := node.(type) {
	case *NumberLiteral:
		return n.Value, nil

	case *BooleanLiteral:
		return n.Value, nil

	case *StringLiteral:
		return n.Value, nil

	case *Identifier:
		val, ok := ctx.Variables[n.Name]
		if !ok {
			return nil, fmt.Errorf("undefined variable '%s'", n.Name)
		}
		return val, nil

	case *UnaryExpr:
		return evalUnary(n, ctx)

	case *BinaryExpr:
		return evalBinary(n, ctx)

	case *FunctionCall:
		return evalFunctionCall(n, ctx)

	case *AssignmentExpr:
		val, err := Evaluate(n.Value, ctx)
		if err != nil {
			return nil, err
		}
		ctx.Variables[n.Name] = val
		return val, nil

	case *ScanQuery:
		// ScanQuery evaluation returns the boolean result of the where clause.
		if n.Where != nil {
			return Evaluate(n.Where, ctx)
		}
		return true, nil

	default:
		return nil, fmt.Errorf("unknown node type: %T", node)
	}
}

func evalUnary(n *UnaryExpr, ctx *EvalContext) (interface{}, error) {
	operand, err := Evaluate(n.Operand, ctx)
	if err != nil {
		return nil, err
	}

	switch n.Operator {
	case "-":
		num, err := toFloat64(operand)
		if err != nil {
			return nil, fmt.Errorf("unary '-' requires numeric operand: %w", err)
		}
		return -num, nil

	case "not":
		b, err := toBool(operand)
		if err != nil {
			return nil, fmt.Errorf("'not' requires boolean operand: %w", err)
		}
		return !b, nil

	default:
		return nil, fmt.Errorf("unknown unary operator '%s'", n.Operator)
	}
}

func evalBinary(n *BinaryExpr, ctx *EvalContext) (interface{}, error) {
	left, err := Evaluate(n.Left, ctx)
	if err != nil {
		return nil, err
	}

	// Short-circuit for logical operators
	switch n.Operator {
	case "and":
		lb, err := toBool(left)
		if err != nil {
			return nil, fmt.Errorf("'and' requires boolean operands: %w", err)
		}
		if !lb {
			return false, nil
		}
		right, err := Evaluate(n.Right, ctx)
		if err != nil {
			return nil, err
		}
		rb, err := toBool(right)
		if err != nil {
			return nil, fmt.Errorf("'and' requires boolean operands: %w", err)
		}
		return rb, nil

	case "or":
		lb, err := toBool(left)
		if err != nil {
			return nil, fmt.Errorf("'or' requires boolean operands: %w", err)
		}
		if lb {
			return true, nil
		}
		right, err := Evaluate(n.Right, ctx)
		if err != nil {
			return nil, err
		}
		rb, err := toBool(right)
		if err != nil {
			return nil, fmt.Errorf("'or' requires boolean operands: %w", err)
		}
		return rb, nil
	}

	right, err := Evaluate(n.Right, ctx)
	if err != nil {
		return nil, err
	}

	switch n.Operator {
	case "+", "-", "*", "/":
		return evalArithmetic(n.Operator, left, right)
	case "==", "!=", "<", ">", "<=", ">=":
		return evalComparison(n.Operator, left, right)
	default:
		return nil, fmt.Errorf("unknown operator '%s'", n.Operator)
	}
}

func evalArithmetic(op string, left, right interface{}) (interface{}, error) {
	l, err := toFloat64(left)
	if err != nil {
		return nil, fmt.Errorf("arithmetic '%s' requires numeric operands: %w", op, err)
	}
	r, err := toFloat64(right)
	if err != nil {
		return nil, fmt.Errorf("arithmetic '%s' requires numeric operands: %w", op, err)
	}

	switch op {
	case "+":
		return l + r, nil
	case "-":
		return l - r, nil
	case "*":
		return l * r, nil
	case "/":
		if r == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return l / r, nil
	default:
		return nil, fmt.Errorf("unknown arithmetic operator '%s'", op)
	}
}

func evalComparison(op string, left, right interface{}) (interface{}, error) {
	l, err := toFloat64(left)
	if err != nil {
		return nil, fmt.Errorf("comparison '%s' requires numeric operands: %w", op, err)
	}
	r, err := toFloat64(right)
	if err != nil {
		return nil, fmt.Errorf("comparison '%s' requires numeric operands: %w", op, err)
	}

	switch op {
	case "==":
		return l == r, nil
	case "!=":
		return l != r, nil
	case "<":
		return l < r, nil
	case ">":
		return l > r, nil
	case "<=":
		return l <= r, nil
	case ">=":
		return l >= r, nil
	default:
		return nil, fmt.Errorf("unknown comparison operator '%s'", op)
	}
}

func evalFunctionCall(n *FunctionCall, ctx *EvalContext) (interface{}, error) {
	fn, ok := ctx.Functions[n.Name]
	if !ok {
		return nil, fmt.Errorf("undefined function '%s'", n.Name)
	}

	var args []float64
	for _, argNode := range n.Args {
		val, err := Evaluate(argNode, ctx)
		if err != nil {
			return nil, err
		}
		num, err := toFloat64(val)
		if err != nil {
			return nil, fmt.Errorf("function '%s' argument must be numeric: %w", n.Name, err)
		}
		args = append(args, num)
	}

	return fn(args...)
}

func toFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}

func toBool(val interface{}) (bool, error) {
	switch v := val.(type) {
	case bool:
		return v, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", val)
	}
}
