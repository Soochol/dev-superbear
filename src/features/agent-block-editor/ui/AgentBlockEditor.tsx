"use client";

import { useState, useEffect } from "react";
import type { AgentBlock } from "@/entities/agent-block/model/types";
import { apiPut } from "@/shared/api/client";
import ToolSelector from "./ToolSelector";

interface AgentBlockEditorProps {
  block: AgentBlock | null;
  open: boolean;
  onClose: () => void;
  onSave: (updated: AgentBlock) => void;
}

export default function AgentBlockEditor({
  block,
  open,
  onClose,
  onSave,
}: AgentBlockEditorProps) {
  const [name, setName] = useState("");
  const [objective, setObjective] = useState("");
  const [inputDesc, setInputDesc] = useState("");
  const [tools, setTools] = useState<string[]>([]);
  const [outputFormat, setOutputFormat] = useState("");
  const [constraints, setConstraints] = useState("");
  const [examples, setExamples] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (block) {
      setName(block.name);
      setObjective(block.objective);
      setInputDesc(block.inputDesc);
      setTools([...block.tools]);
      setOutputFormat(block.outputFormat);
      setConstraints(block.constraints ?? "");
      setExamples(block.examples ?? "");
    }
  }, [block]);

  if (!open || !block) return null;

  const handleSave = async () => {
    setSaving(true);
    try {
      const body = {
        name,
        objective,
        inputDesc,
        tools,
        outputFormat,
        constraints: constraints || null,
        examples: examples || null,
      };

      // If the block already has a persisted ID (not local-only), update via API
      if (block.userId) {
        await apiPut(`/agent-blocks/${block.id}`, body);
      }

      const updated: AgentBlock = {
        ...block,
        ...body,
        allowedTools: tools,
        updatedAt: new Date().toISOString(),
      };
      onSave(updated);
      onClose();
    } catch {
      // API error -- keep modal open so user can retry
    } finally {
      setSaving(false);
    }
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
      <div className="relative bg-nexus-surface border border-nexus-border rounded-lg w-full max-w-2xl max-h-[85vh] overflow-y-auto shadow-2xl">
        <div className="sticky top-0 bg-nexus-surface border-b border-nexus-border px-6 py-4 flex items-center justify-between">
          <h2 className="text-base font-semibold text-nexus-text-primary">
            Edit Agent Block
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
          {/* Name */}
          <div>
            <label className="block text-xs font-medium text-nexus-text-muted mb-1">
              Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full bg-nexus-bg border border-nexus-border rounded-md px-3 py-2 text-sm text-nexus-text-primary focus:outline-none focus:border-nexus-accent"
            />
          </div>

          {/* Objective */}
          <div>
            <label className="block text-xs font-medium text-nexus-text-muted mb-1">
              Objective
            </label>
            <textarea
              value={objective}
              onChange={(e) => setObjective(e.target.value)}
              rows={3}
              className="w-full bg-nexus-bg border border-nexus-border rounded-md px-3 py-2 text-sm text-nexus-text-primary resize-none focus:outline-none focus:border-nexus-accent"
            />
          </div>

          {/* Input Description */}
          <div>
            <label className="block text-xs font-medium text-nexus-text-muted mb-1">
              Input Description
            </label>
            <input
              type="text"
              value={inputDesc}
              onChange={(e) => setInputDesc(e.target.value)}
              className="w-full bg-nexus-bg border border-nexus-border rounded-md px-3 py-2 text-sm text-nexus-text-primary focus:outline-none focus:border-nexus-accent"
            />
          </div>

          {/* Tools */}
          <ToolSelector selected={tools} onChange={setTools} />

          {/* Output Format */}
          <div>
            <label className="block text-xs font-medium text-nexus-text-muted mb-1">
              Output Format
            </label>
            <textarea
              value={outputFormat}
              onChange={(e) => setOutputFormat(e.target.value)}
              rows={2}
              className="w-full bg-nexus-bg border border-nexus-border rounded-md px-3 py-2 text-sm font-mono text-nexus-text-primary resize-none focus:outline-none focus:border-nexus-accent"
            />
          </div>

          {/* Constraints (optional) */}
          <div>
            <label className="block text-xs font-medium text-nexus-text-muted mb-1">
              Constraints{" "}
              <span className="text-nexus-text-muted/50">(optional)</span>
            </label>
            <textarea
              value={constraints}
              onChange={(e) => setConstraints(e.target.value)}
              rows={2}
              className="w-full bg-nexus-bg border border-nexus-border rounded-md px-3 py-2 text-sm text-nexus-text-primary resize-none focus:outline-none focus:border-nexus-accent"
            />
          </div>

          {/* Examples (optional) */}
          <div>
            <label className="block text-xs font-medium text-nexus-text-muted mb-1">
              Examples{" "}
              <span className="text-nexus-text-muted/50">(optional)</span>
            </label>
            <textarea
              value={examples}
              onChange={(e) => setExamples(e.target.value)}
              rows={2}
              className="w-full bg-nexus-bg border border-nexus-border rounded-md px-3 py-2 text-sm font-mono text-nexus-text-primary resize-none focus:outline-none focus:border-nexus-accent"
            />
          </div>
        </div>

        {/* Footer */}
        <div className="sticky bottom-0 bg-nexus-surface border-t border-nexus-border px-6 py-3 flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm text-nexus-text-secondary hover:text-nexus-text-primary hover:bg-nexus-bg rounded-md transition-colors"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={saving || !name.trim()}
            className="px-4 py-2 text-sm bg-nexus-accent hover:bg-nexus-accent/80 text-white rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {saving ? "Saving..." : "Save"}
          </button>
        </div>
      </div>
    </div>
  );
}
