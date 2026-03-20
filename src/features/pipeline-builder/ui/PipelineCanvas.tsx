"use client";

import type { AgentBlock } from "@/entities/agent-block";
import AnalysisSection from "./sections/AnalysisSection";
import MonitoringSection from "./sections/MonitoringSection";
import JudgmentSection from "./sections/JudgmentSection";

interface PipelineCanvasProps {
  onEditBlock?: (block: AgentBlock) => void;
}

export default function PipelineCanvas({ onEditBlock }: PipelineCanvasProps) {
  return (
    <div className="flex-1 overflow-y-auto p-4 space-y-4">
      <AnalysisSection onEditBlock={onEditBlock} />
      <MonitoringSection onEditBlock={onEditBlock} />
      <JudgmentSection />
    </div>
  );
}
