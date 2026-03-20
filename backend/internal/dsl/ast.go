package dsl

// Node is the interface implemented by all AST nodes.
type Node interface {
	nodeType() string
}

// ScanQuery represents a scan/where/sort/limit query.
type ScanQuery struct {
	Where  Node
	SortBy *SortClause
	Limit  *int
}

func (s *ScanQuery) nodeType() string { return "ScanQuery" }

// SortClause represents a sort directive with field and direction.
type SortClause struct {
	Field     string
	Direction string
}

// AssignmentExpr represents a variable assignment (e.g., success = expr).
type AssignmentExpr struct {
	Name  string
	Value Node
}

func (a *AssignmentExpr) nodeType() string { return "AssignmentExpr" }

// BinaryExpr represents a binary operation (e.g., a + b, x >= y).
type BinaryExpr struct {
	Operator string
	Left     Node
	Right    Node
}

func (b *BinaryExpr) nodeType() string { return "BinaryExpr" }

// UnaryExpr represents a unary operation (e.g., -x, not x).
type UnaryExpr struct {
	Operator string
	Operand  Node
}

func (u *UnaryExpr) nodeType() string { return "UnaryExpr" }

// FunctionCall represents a function invocation (e.g., pre_event_ma(120)).
type FunctionCall struct {
	Name string
	Args []Node
}

func (f *FunctionCall) nodeType() string { return "FunctionCall" }

// Identifier represents a variable or field reference.
type Identifier struct {
	Name string
}

func (i *Identifier) nodeType() string { return "Identifier" }

// NumberLiteral represents a numeric constant.
type NumberLiteral struct {
	Value float64
}

func (n *NumberLiteral) nodeType() string { return "NumberLiteral" }

// BooleanLiteral represents a boolean constant (true/false).
type BooleanLiteral struct {
	Value bool
}

func (b *BooleanLiteral) nodeType() string { return "BooleanLiteral" }

// StringLiteral represents a string constant.
type StringLiteral struct {
	Value string
}

func (s *StringLiteral) nodeType() string { return "StringLiteral" }
