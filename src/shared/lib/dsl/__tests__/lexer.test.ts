import { Lexer } from "../lexer";
import { TokenType } from "../tokens";

function tokenize(input: string) {
  return new Lexer(input).tokenize().filter((t) => t.type !== TokenType.WHITESPACE && t.type !== TokenType.EOF);
}

function tokenTypes(input: string) {
  return tokenize(input).map((t) => t.type);
}

describe("Lexer", () => {
  it("tokenizes a full scan query", () => {
    const tokens = tokenize("scan where volume > 1000000 sort by trade_value desc limit 50");
    expect(tokens.map((t) => t.value)).toEqual([
      "scan", "where", "volume", ">", "1000000", "sort", "by", "trade_value", "desc", "limit", "50",
    ]);
  });

  it("recognizes all keywords case-insensitively", () => {
    expect(tokenTypes("scan")).toEqual([TokenType.SCAN]);
    expect(tokenTypes("SCAN")).toEqual([TokenType.SCAN]);
    expect(tokenTypes("Scan")).toEqual([TokenType.SCAN]);
    expect(tokenTypes("where")).toEqual([TokenType.WHERE]);
    expect(tokenTypes("sort")).toEqual([TokenType.SORT]);
    expect(tokenTypes("by")).toEqual([TokenType.BY]);
    expect(tokenTypes("and")).toEqual([TokenType.AND]);
    expect(tokenTypes("or")).toEqual([TokenType.OR]);
    expect(tokenTypes("asc")).toEqual([TokenType.ASC]);
    expect(tokenTypes("desc")).toEqual([TokenType.DESC]);
    expect(tokenTypes("limit")).toEqual([TokenType.LIMIT]);
  });

  it("tokenizes comparison operators", () => {
    expect(tokenTypes(">")).toEqual([TokenType.GT]);
    expect(tokenTypes(">=")).toEqual([TokenType.GTE]);
    expect(tokenTypes("<")).toEqual([TokenType.LT]);
    expect(tokenTypes("<=")).toEqual([TokenType.LTE]);
    expect(tokenTypes("==")).toEqual([TokenType.EQ]);
    expect(tokenTypes("!=")).toEqual([TokenType.NEQ]);
    expect(tokenTypes("=")).toEqual([TokenType.ASSIGN]);
  });

  it("tokenizes arithmetic operators", () => {
    expect(tokenTypes("+")).toEqual([TokenType.PLUS]);
    expect(tokenTypes("-")).toEqual([TokenType.MINUS]);
    expect(tokenTypes("*")).toEqual([TokenType.STAR]);
    expect(tokenTypes("/")).toEqual([TokenType.SLASH]);
  });

  it("tokenizes numbers", () => {
    const intToken = tokenize("42");
    expect(intToken[0].type).toBe(TokenType.NUMBER);
    expect(intToken[0].value).toBe("42");

    const decToken = tokenize("3.14");
    expect(decToken[0].type).toBe(TokenType.NUMBER);
    expect(decToken[0].value).toBe("3.14");

    const dotToken = tokenize(".5");
    expect(dotToken[0].type).toBe(TokenType.NUMBER);
    expect(dotToken[0].value).toBe(".5");
  });

  it("tokenizes identifiers", () => {
    const tokens = tokenize("volume trade_value close");
    expect(tokens.map((t) => t.type)).toEqual([
      TokenType.IDENTIFIER,
      TokenType.IDENTIFIER,
      TokenType.IDENTIFIER,
    ]);
    expect(tokens.map((t) => t.value)).toEqual(["volume", "trade_value", "close"]);
  });

  it("tokenizes string literals", () => {
    const tokens = tokenize('"hello"');
    expect(tokens[0].type).toBe(TokenType.STRING);
    expect(tokens[0].value).toBe('"hello"');
  });

  it("handles escaped quotes in strings", () => {
    const tokens = tokenize('"he said \\"hi\\""');
    expect(tokens[0].type).toBe(TokenType.STRING);
    expect(tokens[0].value).toBe('"he said \\"hi\\""');
  });

  it("handles unterminated strings without crashing", () => {
    const tokens = tokenize('"unterminated');
    expect(tokens[0].type).toBe(TokenType.STRING);
    expect(tokens[0].value).toBe('"unterminated');
  });

  it("tokenizes delimiters", () => {
    expect(tokenTypes("(")).toEqual([TokenType.LPAREN]);
    expect(tokenTypes(")")).toEqual([TokenType.RPAREN]);
    expect(tokenTypes(",")).toEqual([TokenType.COMMA]);
  });

  it("tokenizes function calls", () => {
    const tokens = tokenize("ma(20)");
    expect(tokens.map((t) => t.type)).toEqual([
      TokenType.IDENTIFIER,
      TokenType.LPAREN,
      TokenType.NUMBER,
      TokenType.RPAREN,
    ]);
  });

  it("produces EOF for empty input", () => {
    const tokens = new Lexer("").tokenize();
    expect(tokens).toEqual([{ type: TokenType.EOF, value: "", position: 0 }]);
  });

  it("preserves whitespace tokens", () => {
    const allTokens = new Lexer("a b").tokenize();
    const wsTokens = allTokens.filter((t) => t.type === TokenType.WHITESPACE);
    expect(wsTokens.length).toBeGreaterThan(0);
  });

  it("tokenizes a complex DSL expression", () => {
    const tokens = tokenize("success = close >= event_high * 2.0");
    expect(tokens.map((t) => t.value)).toEqual([
      "success", "=", "close", ">=", "event_high", "*", "2.0",
    ]);
  });
});
