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
  { label: "or", type: "keyword", detail: "논리 OR" },
  { label: "limit", type: "keyword", detail: "결과 제한" },

  // Variables
  { label: "close", type: "variable", detail: "종가 / 현재가" },
  { label: "open", type: "variable", detail: "시가" },
  { label: "high", type: "variable", detail: "고가" },
  { label: "low", type: "variable", detail: "저가" },
  { label: "volume", type: "variable", detail: "거래량" },
  { label: "trade_value", type: "variable", detail: "거래대금" },
  { label: "market_cap", type: "variable", detail: "시가총액" },
  { label: "per", type: "variable", detail: "PER (주가수익비율)" },
  { label: "pbr", type: "variable", detail: "PBR (주가순자산비율)" },
  { label: "roe", type: "variable", detail: "ROE (자기자본이익률)" },
  { label: "event_high", type: "variable", detail: "이벤트 발생일 고가" },
  { label: "event_low", type: "variable", detail: "이벤트 발생일 저가" },
  { label: "event_close", type: "variable", detail: "이벤트 발생일 종가" },
  { label: "event_volume", type: "variable", detail: "이벤트 발생일 거래량" },
  { label: "pre_event_close", type: "variable", detail: "이벤트 전일 종가" },
  { label: "post_high", type: "variable", detail: "이벤트 이후 최고가" },
  { label: "post_low", type: "variable", detail: "이벤트 이후 최저가" },
  { label: "days_since_event", type: "variable", detail: "이벤트 이후 경과일" },

  // Functions
  { label: "ma", type: "function", detail: "ma(N) — N일 이동평균" },
  { label: "rsi", type: "function", detail: "rsi(N) — N일 RSI" },
  { label: "macd", type: "function", detail: "macd(short, long, signal)" },
  { label: "bb", type: "function", detail: "bb(N, K) — 볼린저밴드" },
  { label: "max_volume", type: "function", detail: "max_volume(days) — N일 최대거래량" },
  {
    label: "pre_event_ma",
    type: "function",
    detail: "pre_event_ma(N) — 이벤트 전일 기준 N일 이평선",
  },
  { label: "max", type: "function", detail: "max(a, b) — 큰 값" },
  { label: "min", type: "function", detail: "min(a, b) — 작은 값" },
  { label: "abs", type: "function", detail: "abs(x) — 절대값" },
];
