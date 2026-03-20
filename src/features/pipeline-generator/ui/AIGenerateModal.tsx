"use client";

import { useState } from "react";
import type { AgentBlock } from "@/entities/agent-block";
import type { StageState, MonitorBlockState, PriceAlertState } from "@/features/pipeline-builder";
import {
  generatePipeline,
  type GenerateResponse,
} from "../lib/pipeline-generator-api";

interface AIGenerateModalProps {
  open: boolean;
  onClose: () => void;
  onApply: (data: {
    name: string;
    description: string;
    stages: StageState[];
    monitors: MonitorBlockState[];
    successScript: string;
    failureScript: string;
    priceAlerts: PriceAlertState[];
  }) => void;
}

function makeBlock(partial: {
  name: string;
  objective: string;
  inputDesc?: string;
  tools?: string[];
  outputFormat?: string;
}): AgentBlock {
  return {
    id: crypto.randomUUID(),
    userId: "",
    name: partial.name,
    objective: partial.objective,
    inputDesc: partial.inputDesc ?? "",
    tools: partial.tools ?? [],
    outputFormat: partial.outputFormat ?? "",
    constraints: null,
    examples: null,
    instruction: "",
    systemPrompt: null,
    allowedTools: partial.tools ?? [],
    isPublic: false,
    isTemplate: false,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };
}

function transformResponse(res: GenerateResponse["data"]) {
  const stages: StageState[] = res.stages
    .filter((s) => s.section === "analysis")
    .sort((a, b) => a.order - b.order)
    .map((s) => ({
      order: s.order,
      blocks: s.blocks.map((b) => makeBlock(b)),
    }));

  if (stages.length === 0) {
    stages.push({ order: 0, blocks: [] });
  }

  const monitors: MonitorBlockState[] = res.monitors.map((m) => ({
    id: crypto.randomUUID(),
    block: makeBlock(m.block),
    cron: m.cron,
    enabled: m.enabled,
  }));

  const priceAlerts: PriceAlertState[] = res.priceAlerts.map((a) => ({
    id: crypto.randomUUID(),
    condition: a.condition,
    label: a.label,
  }));

  return {
    name: res.name,
    description: res.description,
    stages,
    monitors,
    successScript: res.successScript ?? "",
    failureScript: res.failureScript ?? "",
    priceAlerts,
  };
}

export default function AIGenerateModal({
  open,
  onClose,
  onApply,
}: AIGenerateModalProps) {
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [preview, setPreview] = useState<ReturnType<
    typeof transformResponse
  > | null>(null);

  if (!open) return null;

  const handleGenerate = async () => {
    if (!description.trim()) return;
    setLoading(true);
    setError(null);
    setPreview(null);
    try {
      const res = await generatePipeline(description.trim());
      setPreview(transformResponse(res.data));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Generation failed");
    } finally {
      setLoading(false);
    }
  };

  const handleApply = () => {
    if (!preview) return;
    onApply(preview);
    setDescription("");
    setPreview(null);
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60"
        onClick={onClose}
        onKeyDown={(e) => e.key === "Escape" && onClose()}
        role="button"
        tabIndex={-1}
        aria-label="Close modal"
      />

      {/* Modal */}
      <div className="relative bg-nexus-surface border border-nexus-border rounded-lg w-full max-w-xl max-h-[80vh] overflow-y-auto shadow-2xl">
        <div className="sticky top-0 bg-nexus-surface border-b border-nexus-border px-6 py-4 flex items-center justify-between">
          <h2 className="text-base font-semibold text-nexus-text-primary">
            AI Pipeline Generator
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="text-nexus-text-muted hover:text-nexus-text-primary transition-colors"
            aria-label="Close"
          >
            <svg
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M18 6 6 18" />
              <path d="m6 6 12 12" />
            </svg>
          </button>
        </div>

        <div className="px-6 py-4 space-y-4">
          {/* Description input */}
          <div>
            <label className="block text-xs font-medium text-nexus-text-muted mb-1.5">
              Describe the pipeline you want to create
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="e.g. 삼성전자 뉴스 분석 후 재무 분석을 수행하고, 섹터 비교를 통해 투자 케이스를 생성하는 파이프라인"
              rows={4}
              className="w-full bg-nexus-bg border border-nexus-border rounded-md px-3 py-2 text-sm text-nexus-text-primary placeholder:text-nexus-text-muted/50 resize-none focus:outline-none focus:border-nexus-accent"
            />
          </div>

          <button
            type="button"
            onClick={handleGenerate}
            disabled={loading || !description.trim()}
            className="w-full flex items-center justify-center gap-2 px-4 py-2 text-sm bg-nexus-accent hover:bg-nexus-accent/80 text-white rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? (
              <>
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="none"
                  className="animate-spin"
                >
                  <circle
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeDasharray="60"
                    strokeDashoffset="20"
                    strokeLinecap="round"
                  />
                </svg>
                Generating...
              </>
            ) : (
              "Generate Pipeline"
            )}
          </button>

          {error && (
            <p className="text-xs text-nexus-failure bg-nexus-failure/10 rounded-md px-3 py-2">
              {error}
            </p>
          )}

          {/* Preview */}
          {preview && (
            <div className="space-y-3 border-t border-nexus-border pt-4">
              <h3 className="text-sm font-medium text-nexus-text-primary">
                Preview
              </h3>

              <div className="text-xs text-nexus-text-secondary space-y-2">
                <div>
                  <span className="text-nexus-text-muted">Name: </span>
                  {preview.name}
                </div>
                <div>
                  <span className="text-nexus-text-muted">Description: </span>
                  {preview.description}
                </div>

                <div>
                  <span className="text-nexus-text-muted">
                    Analysis Stages: {preview.stages.length}
                  </span>
                  {preview.stages.map((stage) => (
                    <div
                      key={stage.order}
                      className="ml-3 mt-1 text-nexus-text-muted"
                    >
                      Stage {stage.order}:{" "}
                      {stage.blocks.map((b) => b.name).join(", ")}
                    </div>
                  ))}
                </div>

                {preview.monitors.length > 0 && (
                  <div>
                    <span className="text-nexus-text-muted">
                      Monitors: {preview.monitors.length}
                    </span>
                  </div>
                )}

                {preview.successScript && (
                  <div>
                    <span className="text-nexus-success">Success: </span>
                    <code className="font-mono text-[10px]">
                      {preview.successScript}
                    </code>
                  </div>
                )}
                {preview.failureScript && (
                  <div>
                    <span className="text-nexus-failure">Failure: </span>
                    <code className="font-mono text-[10px]">
                      {preview.failureScript}
                    </code>
                  </div>
                )}

                {preview.priceAlerts.length > 0 && (
                  <div>
                    <span className="text-nexus-text-muted">
                      Price Alerts: {preview.priceAlerts.length}
                    </span>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        {preview && (
          <div className="sticky bottom-0 bg-nexus-surface border-t border-nexus-border px-6 py-3 flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setPreview(null)}
              className="px-4 py-2 text-sm text-nexus-text-secondary hover:text-nexus-text-primary hover:bg-nexus-bg rounded-md transition-colors"
            >
              Discard
            </button>
            <button
              type="button"
              onClick={handleApply}
              className="px-4 py-2 text-sm bg-nexus-accent hover:bg-nexus-accent/80 text-white rounded-md transition-colors"
            >
              Apply to Canvas
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
