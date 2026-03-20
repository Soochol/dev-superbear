import { Lexer } from "./lexer";
import { TokenType } from "./tokens";

const TOKEN_COLORS: Partial<Record<TokenType, string>> = {
  [TokenType.SCAN]: "text-purple-400",
  [TokenType.WHERE]: "text-purple-400",
  [TokenType.SORT]: "text-purple-400",
  [TokenType.BY]: "text-purple-400",
  [TokenType.AND]: "text-purple-400",
  [TokenType.OR]: "text-purple-400",
  [TokenType.ASC]: "text-purple-400",
  [TokenType.DESC]: "text-purple-400",
  [TokenType.LIMIT]: "text-purple-400",
  [TokenType.NUMBER]: "text-green-400",
  [TokenType.GTE]: "text-yellow-300",
  [TokenType.LTE]: "text-yellow-300",
  [TokenType.GT]: "text-yellow-300",
  [TokenType.LT]: "text-yellow-300",
  [TokenType.EQ]: "text-yellow-300",
  [TokenType.NEQ]: "text-yellow-300",
  [TokenType.ASSIGN]: "text-yellow-300",
  [TokenType.STAR]: "text-yellow-300",
  [TokenType.SLASH]: "text-yellow-300",
  [TokenType.PLUS]: "text-yellow-300",
  [TokenType.MINUS]: "text-yellow-300",
};

export interface HighlightedToken {
  text: string;
  className: string;
}

export function highlightDSL(code: string): HighlightedToken[] {
  try {
    const tokens = new Lexer(code).tokenize();
    return tokens
      .filter((t) => t.type !== TokenType.EOF)
      .map((t) => ({
        text: t.value,
        className: TOKEN_COLORS[t.type] ?? "text-nexus-text-primary",
      }));
  } catch (err) {
    console.error("[highlightDSL] Lexer failed for input:", code, err);
    return [{ text: code, className: "text-nexus-text-primary" }];
  }
}
