"use client";

import type { AgentBlock } from "@/entities/agent-block/model/types";

interface BlockCardProps {
  block: AgentBlock;
  onEdit?: (block: AgentBlock) => void;
  onDelete?: (blockId: string) => void;
}

export default function BlockCard({ block, onEdit, onDelete }: BlockCardProps) {
  return (
    <div className="group relative bg-nexus-surface border border-nexus-border rounded-lg p-3 hover:border-nexus-accent/50 transition-colors min-w-[180px]">
      <div className="flex items-start justify-between gap-2">
        <h4 className="text-sm font-medium text-nexus-text-primary truncate">
          {block.name}
        </h4>
        <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
          {onEdit && (
            <button
              type="button"
              onClick={() => onEdit(block)}
              className="p-1 text-nexus-text-muted hover:text-nexus-accent transition-colors"
              aria-label="Edit block"
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" />
                <path d="m15 5 4 4" />
              </svg>
            </button>
          )}
          {onDelete && (
            <button
              type="button"
              onClick={() => onDelete(block.id)}
              className="p-1 text-nexus-text-muted hover:text-nexus-failure transition-colors"
              aria-label="Delete block"
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M3 6h18" />
                <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" />
                <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
              </svg>
            </button>
          )}
        </div>
      </div>
      <p className="text-xs text-nexus-text-muted mt-1 line-clamp-2">
        {block.objective}
      </p>
      {block.tools.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-2">
          {block.tools.slice(0, 3).map((tool) => (
            <span
              key={tool}
              className="text-[10px] px-1.5 py-0.5 rounded bg-nexus-accent/10 text-nexus-accent"
            >
              {tool}
            </span>
          ))}
          {block.tools.length > 3 && (
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-nexus-border text-nexus-text-muted">
              +{block.tools.length - 3}
            </span>
          )}
        </div>
      )}
    </div>
  );
}
