import { Lexer } from "@/shared/lib/dsl/lexer";
import { TokenType } from "@/shared/lib/dsl/tokens";

export interface CompletionItem {
  label: string;
  type: "keyword" | "function" | "variable";
  detail: string;
}

export const DSL_COMPLETIONS: CompletionItem[] = [
  // Keywords
  { label: "scan", type: "keyword", detail: "종목 스캔 시작" },
  { label: "where", type: "keyword", detail: "필터 조건" },
  { label: "sort", type: "keyword", detail: "정렬" },
  { label: "by", type: "keyword", detail: "정렬 기준" },
  { label: "asc", type: "keyword", detail: "오름차순" },
  { label: "desc", type: "keyword", detail: "내림차순" },
  { label: "and", type: "keyword", detail: "논리 AND" },
  { label: "limit", type: "keyword", detail: "결과 제한" },

  // Variables (backend-supported fields only)
  { label: "close", type: "variable", detail: "종가 / 현재가" },
  { label: "open", type: "variable", detail: "시가" },
  { label: "high", type: "variable", detail: "고가" },
  { label: "low", type: "variable", detail: "저가" },
  { label: "volume", type: "variable", detail: "거래량" },
  { label: "trade_value", type: "variable", detail: "거래대금" },
  { label: "change_pct", type: "variable", detail: "등락률 (%)" },
];

const FIELD_COMPLETIONS = DSL_COMPLETIONS.filter((c) => c.type === "variable");

const OPERATOR_COMPLETIONS: CompletionItem[] = [
  { label: ">", type: "keyword", detail: "초과" },
  { label: ">=", type: "keyword", detail: "이상" },
  { label: "<", type: "keyword", detail: "미만" },
  { label: "<=", type: "keyword", detail: "이하" },
  { label: "=", type: "keyword", detail: "같음" },
];

const AFTER_VALUE_COMPLETIONS: CompletionItem[] = [
  { label: "and", type: "keyword", detail: "논리 AND" },
  { label: "sort", type: "keyword", detail: "정렬" },
  { label: "limit", type: "keyword", detail: "결과 제한" },
];

const BY_COMPLETIONS: CompletionItem[] = [
  { label: "by", type: "keyword", detail: "정렬 기준" },
];

const SORT_DIRECTION_COMPLETIONS: CompletionItem[] = [
  { label: "asc", type: "keyword", detail: "오름차순" },
  { label: "desc", type: "keyword", detail: "내림차순" },
];

const SCAN_COMPLETIONS: CompletionItem[] = [
  { label: "scan", type: "keyword", detail: "종목 스캔 시작" },
];

const WHERE_COMPLETIONS: CompletionItem[] = [
  { label: "where", type: "keyword", detail: "필터 조건" },
];

export function getContextualCompletions(input: string): CompletionItem[] {
  if (!input.trim()) {
    return SCAN_COMPLETIONS;
  }

  const lexer = new Lexer(input);
  const allTokens = lexer.tokenize();

  // Filter out WHITESPACE and EOF tokens
  const tokens = allTokens.filter(
    (t) => t.type !== TokenType.WHITESPACE && t.type !== TokenType.EOF
  );

  if (tokens.length === 0) {
    return SCAN_COMPLETIONS;
  }

  // If input ends with whitespace, we look at the last meaningful token
  // If input does NOT end with whitespace, the user is still typing the current token
  const endsWithSpace = /\s$/.test(input);

  if (!endsWithSpace) {
    // User is mid-token — no completions (or could filter, but task doesn't require it)
    return [];
  }

  const last = tokens[tokens.length - 1];
  const secondLast = tokens.length >= 2 ? tokens[tokens.length - 2] : null;

  switch (last.type) {
    case TokenType.SCAN:
      return WHERE_COMPLETIONS;

    case TokenType.WHERE:
    case TokenType.AND:
      return FIELD_COMPLETIONS;

    case TokenType.IDENTIFIER:
      // If preceded by BY token, suggest sort directions (asc/desc)
      if (secondLast && secondLast.type === TokenType.BY) {
        return SORT_DIRECTION_COMPLETIONS;
      }
      // Otherwise it's a field name — suggest operators
      return OPERATOR_COMPLETIONS;

    case TokenType.NUMBER:
      return AFTER_VALUE_COMPLETIONS;

    case TokenType.SORT:
      return BY_COMPLETIONS;

    case TokenType.BY:
      return FIELD_COMPLETIONS;

    case TokenType.ASC:
    case TokenType.DESC:
      return AFTER_VALUE_COMPLETIONS;

    case TokenType.GT:
    case TokenType.LT:
    case TokenType.GTE:
    case TokenType.LTE:
    case TokenType.EQ:
    case TokenType.ASSIGN:
      // Waiting for a number — no completions
      return [];

    default:
      return [];
  }
}

export function dslAutoComplete(input: string) {
  return getContextualCompletions(input);
}
