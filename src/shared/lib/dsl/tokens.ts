export enum TokenType {
  // Keywords
  SCAN = "SCAN",
  WHERE = "WHERE",
  SORT = "SORT",
  BY = "BY",
  AND = "AND",
  OR = "OR",
  ASC = "ASC",
  DESC = "DESC",
  LIMIT = "LIMIT",

  // Literals
  NUMBER = "NUMBER",
  STRING = "STRING",
  IDENTIFIER = "IDENTIFIER",

  // Operators
  GTE = "GTE",
  LTE = "LTE",
  GT = "GT",
  LT = "LT",
  EQ = "EQ",
  NEQ = "NEQ",
  ASSIGN = "ASSIGN",

  // Arithmetic
  PLUS = "PLUS",
  MINUS = "MINUS",
  STAR = "STAR",
  SLASH = "SLASH",

  // Delimiters
  LPAREN = "LPAREN",
  RPAREN = "RPAREN",
  COMMA = "COMMA",

  // Special
  EOF = "EOF",
  WHITESPACE = "WHITESPACE",
}

export interface Token {
  type: TokenType;
  value: string;
  position: number;
}
