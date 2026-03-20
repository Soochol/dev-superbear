package dsl

// TokenType represents the type of a lexer token.
type TokenType int

const (
	TOKEN_NUMBER TokenType = iota
	TOKEN_STRING
	TOKEN_IDENTIFIER
	TOKEN_SCAN
	TOKEN_WHERE
	TOKEN_SORT
	TOKEN_BY
	TOKEN_ASC
	TOKEN_DESC
	TOKEN_AND
	TOKEN_OR
	TOKEN_NOT
	TOKEN_TRUE
	TOKEN_FALSE
	TOKEN_LIMIT
	TOKEN_PLUS    // +
	TOKEN_MINUS   // -
	TOKEN_STAR    // *
	TOKEN_SLASH   // /
	TOKEN_EQ      // ==
	TOKEN_NEQ     // !=
	TOKEN_LT      // <
	TOKEN_GT      // >
	TOKEN_LTE     // <=
	TOKEN_GTE     // >=
	TOKEN_ASSIGN  // =
	TOKEN_LPAREN  // (
	TOKEN_RPAREN  // )
	TOKEN_COMMA   // ,
	TOKEN_DOT     // .
	TOKEN_EOF
	TOKEN_NEWLINE
)

// Token represents a single lexer token with position information.
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

// keywords maps keyword strings to their token types.
var keywords = map[string]TokenType{
	"scan":  TOKEN_SCAN,
	"where": TOKEN_WHERE,
	"sort":  TOKEN_SORT,
	"by":    TOKEN_BY,
	"asc":   TOKEN_ASC,
	"desc":  TOKEN_DESC,
	"and":   TOKEN_AND,
	"or":    TOKEN_OR,
	"not":   TOKEN_NOT,
	"true":  TOKEN_TRUE,
	"false": TOKEN_FALSE,
	"limit": TOKEN_LIMIT,
}
