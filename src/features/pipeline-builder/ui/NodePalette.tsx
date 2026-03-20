"use client";

import { useState, type DragEvent } from "react";
import type { DragPayload } from "../lib/usePipelineDragDrop";

interface PaletteCategory {
  name: string;
  items: PaletteItem[];
}

interface PaletteItem {
  label: string;
  template: DragPayload["block"];
}

const PALETTE_CATEGORIES: PaletteCategory[] = [
  {
    name: "Agent Nodes",
    items: [
      {
        label: "뉴스 분석",
        template: {
          name: "뉴스 분석",
          objective: "최근 뉴스를 수집하고 주요 이벤트를 분석합니다.",
          inputDesc: "종목 코드와 기간",
          tools: ["search_news"],
          outputFormat: "뉴스 요약 및 영향 분석 결과",
          constraints: null,
          examples: null,
          instruction: "",
          systemPrompt: null,
          allowedTools: ["search_news"],
        },
      },
      {
        label: "섹터 비교",
        template: {
          name: "섹터 비교",
          objective: "동일 섹터 내 종목과 비교 분석합니다.",
          inputDesc: "종목 코드",
          tools: ["get_sector_stocks", "compare_sector"],
          outputFormat: "섹터 내 상대적 위치 및 비교 결과",
          constraints: null,
          examples: null,
          instruction: "",
          systemPrompt: null,
          allowedTools: ["get_sector_stocks", "compare_sector"],
        },
      },
      {
        label: "재무 분석",
        template: {
          name: "재무 분석",
          objective: "재무제표와 밸류에이션 지표를 분석합니다.",
          inputDesc: "종목 코드",
          tools: ["get_financials", "get_valuation"],
          outputFormat: "재무 건전성 및 밸류에이션 분석 결과",
          constraints: null,
          examples: null,
          instruction: "",
          systemPrompt: null,
          allowedTools: ["get_financials", "get_valuation"],
        },
      },
      {
        label: "가격 분석",
        template: {
          name: "가격 분석",
          objective: "캔들 데이터와 현재가를 기반으로 기술적 분석을 수행합니다.",
          inputDesc: "종목 코드와 기간",
          tools: ["get_candles", "get_price"],
          outputFormat: "기술적 분석 결과 (추세, 지지/저항, 패턴)",
          constraints: null,
          examples: null,
          instruction: "",
          systemPrompt: null,
          allowedTools: ["get_candles", "get_price"],
        },
      },
      {
        label: "수급 분석",
        template: {
          name: "수급 분석",
          objective: "외국인/기관 매매 동향을 분석합니다.",
          inputDesc: "종목 코드",
          tools: ["get_fund_flow"],
          outputFormat: "수급 분석 결과 및 투자 주체별 매매 동향",
          constraints: null,
          examples: null,
          instruction: "",
          systemPrompt: null,
          allowedTools: ["get_fund_flow"],
        },
      },
    ],
  },
  {
    name: "DSL Nodes",
    items: [
      {
        label: "DSL 평가",
        template: {
          name: "DSL 평가",
          objective: "DSL 표현식을 평가하여 조건 충족 여부를 판단합니다.",
          inputDesc: "DSL 표현식",
          tools: ["dsl_evaluate"],
          outputFormat: "조건 평가 결과 (true/false 및 세부 값)",
          constraints: null,
          examples: null,
          instruction: "",
          systemPrompt: null,
          allowedTools: ["dsl_evaluate"],
        },
      },
      {
        label: "종목 스캐닝",
        template: {
          name: "종목 스캐닝",
          objective: "조건 기반으로 종목을 스캐닝합니다.",
          inputDesc: "스캐닝 조건",
          tools: ["scan_stocks"],
          outputFormat: "조건을 충족하는 종목 목록",
          constraints: null,
          examples: null,
          instruction: "",
          systemPrompt: null,
          allowedTools: ["scan_stocks"],
        },
      },
    ],
  },
  {
    name: "Output Nodes",
    items: [
      {
        label: "케이스 생성",
        template: {
          name: "케이스 생성",
          objective: "분석 결과를 종합하여 투자 케이스를 생성합니다.",
          inputDesc: "이전 단계의 분석 결과들",
          tools: [],
          outputFormat: "투자 케이스 (요약, 근거, 리스크, 목표가)",
          constraints: null,
          examples: null,
          instruction: "",
          systemPrompt: null,
          allowedTools: [],
        },
      },
      {
        label: "알림 전송",
        template: {
          name: "알림 전송",
          objective: "조건 달성 시 알림을 전송합니다.",
          inputDesc: "알림 조건과 대상",
          tools: [],
          outputFormat: "알림 전송 결과",
          constraints: null,
          examples: null,
          instruction: "",
          systemPrompt: null,
          allowedTools: [],
        },
      },
    ],
  },
];

export default function NodePalette() {
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});

  const toggleCategory = (name: string) => {
    setCollapsed((prev) => ({ ...prev, [name]: !prev[name] }));
  };

  const handleDragStart = (e: DragEvent, template: DragPayload["block"]) => {
    const payload: DragPayload = { type: "palette-block", block: template };
    e.dataTransfer.setData("application/json", JSON.stringify(payload));
    e.dataTransfer.effectAllowed = "copy";
  };

  return (
    <aside className="w-[280px] shrink-0 border-r border-nexus-border bg-nexus-surface overflow-y-auto">
      <div className="p-3">
        <h2 className="text-xs font-semibold text-nexus-text-muted uppercase tracking-wider mb-3">
          Node Palette
        </h2>

        {PALETTE_CATEGORIES.map((category) => (
          <div key={category.name} className="mb-2">
            <button
              type="button"
              onClick={() => toggleCategory(category.name)}
              className="flex items-center gap-1.5 w-full text-left px-2 py-1.5 text-xs font-medium text-nexus-text-secondary hover:text-nexus-text-primary transition-colors rounded"
            >
              <svg
                width="12"
                height="12"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className={`transition-transform ${
                  collapsed[category.name] ? "" : "rotate-90"
                }`}
              >
                <polyline points="9 18 15 12 9 6" />
              </svg>
              {category.name}
            </button>

            {!collapsed[category.name] && (
              <div className="ml-2 space-y-0.5">
                {category.items.map((item) => (
                  <div
                    key={item.label}
                    draggable
                    onDragStart={(e) => handleDragStart(e, item.template)}
                    className="flex items-center gap-2 px-2 py-1.5 text-xs text-nexus-text-secondary hover:text-nexus-text-primary hover:bg-nexus-bg/50 rounded cursor-grab active:cursor-grabbing transition-colors"
                  >
                    <span className="w-1.5 h-1.5 rounded-full bg-nexus-accent/60 shrink-0" />
                    {item.label}
                  </div>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </aside>
  );
}
