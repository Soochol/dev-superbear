export const AGENT_TOOLS = [
  { name: "get_candles", category: "price", description: "캔들 데이터 조회" },
  { name: "get_price", category: "price", description: "현재가 조회" },
  { name: "scan_stocks", category: "price", description: "조건 기반 종목 스캐닝" },
  { name: "get_financials", category: "fundamental", description: "재무제표 조회" },
  { name: "get_disclosures", category: "fundamental", description: "공시 목록 조회" },
  { name: "get_valuation", category: "fundamental", description: "밸류에이션 지표" },
  { name: "search_news", category: "news", description: "뉴스 검색 및 분석" },
  { name: "get_sector_stocks", category: "sector", description: "동일 섹터 종목 목록" },
  { name: "compare_sector", category: "sector", description: "섹터 내 상대 비교" },
  { name: "get_fund_flow", category: "sector", description: "외국인/기관 매매 동향" },
  { name: "dsl_evaluate", category: "dsl", description: "DSL 표현식 평가" },
] as const;

export type AgentToolName = (typeof AGENT_TOOLS)[number]["name"];
export type ToolCategory = (typeof AGENT_TOOLS)[number]["category"];

const CATEGORY_LABELS: Record<ToolCategory, string> = {
  price: "가격/차트",
  fundamental: "펀더멘털",
  news: "뉴스",
  sector: "섹터",
  dsl: "DSL",
};

export function getCategoryLabel(category: ToolCategory): string {
  return CATEGORY_LABELS[category];
}

export function getToolsByCategory(): Map<ToolCategory, typeof AGENT_TOOLS[number][]> {
  const map = new Map<ToolCategory, typeof AGENT_TOOLS[number][]>();
  for (const tool of AGENT_TOOLS) {
    const existing = map.get(tool.category) ?? [];
    existing.push(tool);
    map.set(tool.category, existing);
  }
  return map;
}
