import { Token, TokenType } from "./tokens";

const KEYWORDS: Record<string, TokenType> = {
  scan: TokenType.SCAN,
  where: TokenType.WHERE,
  sort: TokenType.SORT,
  by: TokenType.BY,
  and: TokenType.AND,
  or: TokenType.OR,
  asc: TokenType.ASC,
  desc: TokenType.DESC,
  limit: TokenType.LIMIT,
};

export class Lexer {
  private input: string;
  private pos = 0;
  private tokens: Token[] = [];

  constructor(input: string) {
    this.input = input;
  }

  tokenize(): Token[] {
    while (this.pos < this.input.length) {
      this.skipWhitespaceAndCapture();
      if (this.pos >= this.input.length) break;

      const ch = this.input[this.pos];

      if (this.isDigit(ch) || (ch === "." && this.isDigit(this.input[this.pos + 1]))) {
        this.readNumber();
      } else if (this.isAlpha(ch) || ch === "_") {
        this.readIdentifierOrKeyword();
      } else if (ch === '"' || ch === "'") {
        this.readString(ch);
      } else {
        this.readOperator();
      }
    }

    this.tokens.push({ type: TokenType.EOF, value: "", position: this.pos });
    return this.tokens;
  }

  private skipWhitespaceAndCapture() {
    const start = this.pos;
    while (this.pos < this.input.length && /\s/.test(this.input[this.pos])) {
      this.pos++;
    }
    if (this.pos > start) {
      this.tokens.push({
        type: TokenType.WHITESPACE,
        value: this.input.slice(start, this.pos),
        position: start,
      });
    }
  }

  private readNumber() {
    const start = this.pos;
    while (this.pos < this.input.length && (this.isDigit(this.input[this.pos]) || this.input[this.pos] === ".")) {
      this.pos++;
    }
    this.tokens.push({ type: TokenType.NUMBER, value: this.input.slice(start, this.pos), position: start });
  }

  private readIdentifierOrKeyword() {
    const start = this.pos;
    while (this.pos < this.input.length && (this.isAlphaNumeric(this.input[this.pos]) || this.input[this.pos] === "_")) {
      this.pos++;
    }
    const word = this.input.slice(start, this.pos);
    const kwType = KEYWORDS[word.toLowerCase()];
    this.tokens.push({ type: kwType ?? TokenType.IDENTIFIER, value: word, position: start });
  }

  private readString(quote: string) {
    const start = this.pos;
    this.pos++; // skip opening quote
    while (this.pos < this.input.length && this.input[this.pos] !== quote) {
      if (this.input[this.pos] === "\\") this.pos++; // skip escaped char
      this.pos++;
    }
    if (this.pos < this.input.length) this.pos++; // skip closing quote
    this.tokens.push({ type: TokenType.STRING, value: this.input.slice(start, this.pos), position: start });
  }

  private readOperator() {
    const start = this.pos;
    const ch = this.input[this.pos];
    const next = this.input[this.pos + 1];

    let type: TokenType;
    let len = 1;

    switch (ch) {
      case ">":
        if (next === "=") { type = TokenType.GTE; len = 2; } else { type = TokenType.GT; }
        break;
      case "<":
        if (next === "=") { type = TokenType.LTE; len = 2; } else { type = TokenType.LT; }
        break;
      case "=":
        if (next === "=") { type = TokenType.EQ; len = 2; } else { type = TokenType.ASSIGN; }
        break;
      case "!":
        if (next === "=") { type = TokenType.NEQ; len = 2; } else { type = TokenType.IDENTIFIER; }
        break;
      case "+": type = TokenType.PLUS; break;
      case "-": type = TokenType.MINUS; break;
      case "*": type = TokenType.STAR; break;
      case "/": type = TokenType.SLASH; break;
      case "(": type = TokenType.LPAREN; break;
      case ")": type = TokenType.RPAREN; break;
      case ",": type = TokenType.COMMA; break;
      default: type = TokenType.IDENTIFIER; break;
    }

    this.pos += len;
    this.tokens.push({ type, value: this.input.slice(start, this.pos), position: start });
  }

  private isDigit(ch: string | undefined): boolean {
    return ch !== undefined && ch >= "0" && ch <= "9";
  }

  private isAlpha(ch: string): boolean {
    return (ch >= "a" && ch <= "z") || (ch >= "A" && ch <= "Z") || ch === "_";
  }

  private isAlphaNumeric(ch: string): boolean {
    return this.isAlpha(ch) || this.isDigit(ch);
  }
}
