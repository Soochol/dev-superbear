package dsl

import "fmt"

// ParseDSL tokenizes and parses a DSL input string into an AST.
func ParseDSL(input string) (Node, error) {
	tokens, err := Tokenize(input)
	if err != nil {
		return nil, fmt.Errorf("lexer error: %w", err)
	}
	return Parse(tokens)
}

// EvaluateDSL tokenizes, parses, and evaluates a DSL input string.
func EvaluateDSL(input string, ctx *EvalContext) (interface{}, error) {
	ast, err := ParseDSL(input)
	if err != nil {
		return nil, err
	}
	return Evaluate(ast, ctx)
}

// ValidateDSL checks whether a DSL input string is syntactically valid.
func ValidateDSL(input string) error {
	_, err := ParseDSL(input)
	return err
}
