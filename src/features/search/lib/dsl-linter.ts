import { Lexer } from "@/shared/lib/dsl/lexer";
import { TokenType } from "@/shared/lib/dsl/tokens";

export interface DSLDiagnostic {
  from: number;
  to: number;
  severity: "error" | "warning";
  message: string;
}

const ALLOWED_FIELDS = new Set([
  "close", "open", "high", "low", "volume", "trade_value", "change_pct",
]);

export function lintDSL(input: string): DSLDiagnostic[] {
  if (!input.trim()) return [];

  const lexer = new Lexer(input);
  const tokens = lexer.tokenize();
  const meaningful = tokens.filter(
    (t) => t.type !== TokenType.WHITESPACE && t.type !== TokenType.EOF,
  );

  const diagnostics: DSLDiagnostic[] = [];

  if (meaningful.length === 0) return [];

  // Check: first token must be SCAN
  if (meaningful[0].type !== TokenType.SCAN) {
    diagnostics.push({
      from: meaningful[0].position,
      to: meaningful[0].position + meaningful[0].value.length,
      severity: "error",
      message: "쿼리는 'scan'으로 시작해야 합니다",
    });
    return diagnostics;
  }

  // Check: second meaningful token must be WHERE
  if (meaningful.length > 1 && meaningful[1].type !== TokenType.WHERE) {
    diagnostics.push({
      from: meaningful[1].position,
      to: meaningful[1].position + meaningful[1].value.length,
      severity: "error",
      message: "'scan' 다음에 'where'가 필요합니다",
    });
    return diagnostics;
  }

  // Check for OR usage
  for (const token of meaningful) {
    if (token.type === TokenType.OR) {
      diagnostics.push({
        from: token.position,
        to: token.position + token.value.length,
        severity: "error",
        message: "OR은 지원되지 않습니다. AND를 사용하세요",
      });
    }
  }

  // Check field names after WHERE and AND
  for (let i = 0; i < meaningful.length; i++) {
    const token = meaningful[i];
    if (
      (token.type === TokenType.WHERE || token.type === TokenType.AND) &&
      i + 1 < meaningful.length
    ) {
      const next = meaningful[i + 1];
      if (
        next.type === TokenType.IDENTIFIER &&
        !ALLOWED_FIELDS.has(next.value.toLowerCase())
      ) {
        diagnostics.push({
          from: next.position,
          to: next.position + next.value.length,
          severity: "error",
          message: `알 수 없는 필드: ${next.value}. 사용 가능: ${[...ALLOWED_FIELDS].join(", ")}`,
        });
      }
    }
  }

  return diagnostics;
}
