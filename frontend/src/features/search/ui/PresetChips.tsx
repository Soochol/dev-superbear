"use client";

import { useSearchStore } from "../model/search.store";

const PRESETS = [
  { label: "2yr Max Volume", nlQuery: "최근 5년 안에 2년 최대거래량이 발생한 종목" },
  { label: "Golden Cross", nlQuery: "20일 이평선이 60일 이평선을 상향 돌파한 종목" },
  { label: "RSI Oversold", nlQuery: "RSI(14)가 30 이하로 과매도 구간인 종목" },
  { label: "High Trade Value", nlQuery: "거래대금 3000억 이상인 종목" },
  { label: "PER < 10", nlQuery: "PER이 10배 미만이고 영업이익이 흑자인 종목" },
  { label: "52w High", nlQuery: "52주 신고가를 달성한 종목" },
];

export function PresetChips() {
  const setNlQuery = useSearchStore((s) => s.setNlQuery);

  return (
    <div className="flex flex-wrap gap-2">
      {PRESETS.map((preset) => (
        <button
          key={preset.label}
          onClick={() => setNlQuery(preset.nlQuery)}
          className="px-3 py-1.5 text-xs font-medium rounded-full
                     bg-nexus-border text-nexus-text-secondary
                     hover:bg-nexus-accent/20 hover:text-nexus-accent
                     transition-colors"
        >
          {preset.label}
        </button>
      ))}
    </div>
  );
}
