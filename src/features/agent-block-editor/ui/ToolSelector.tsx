"use client";

import {
  AGENT_TOOLS,
  getToolsByCategory,
  getCategoryLabel,
  type ToolCategory,
} from "@/entities/agent-block/lib/agent-tools";

interface ToolSelectorProps {
  selected: string[];
  onChange: (tools: string[]) => void;
}

export default function ToolSelector({ selected, onChange }: ToolSelectorProps) {
  const toolsByCategory = getToolsByCategory();

  const handleToggle = (toolName: string) => {
    if (selected.includes(toolName)) {
      onChange(selected.filter((t) => t !== toolName));
    } else {
      onChange([...selected, toolName]);
    }
  };

  const categories = Array.from(toolsByCategory.entries()) as [
    ToolCategory,
    (typeof AGENT_TOOLS)[number][],
  ][];

  return (
    <div className="space-y-3">
      <label className="block text-xs font-medium text-nexus-text-muted">
        Tools
      </label>
      {categories.map(([category, tools]) => (
        <div key={category}>
          <span className="text-[10px] font-semibold text-nexus-text-muted uppercase tracking-wider">
            {getCategoryLabel(category)}
          </span>
          <div className="flex flex-wrap gap-1.5 mt-1">
            {tools.map((tool) => {
              const isActive = selected.includes(tool.name);
              return (
                <button
                  key={tool.name}
                  type="button"
                  onClick={() => handleToggle(tool.name)}
                  className={`flex items-center gap-1 px-2 py-1 text-xs rounded-md border transition-colors ${
                    isActive
                      ? "bg-nexus-accent/20 border-nexus-accent/50 text-nexus-accent"
                      : "bg-nexus-bg border-nexus-border text-nexus-text-secondary hover:border-nexus-accent/30"
                  }`}
                >
                  <span
                    className={`w-2 h-2 rounded-sm ${
                      isActive ? "bg-nexus-accent" : "bg-nexus-border"
                    }`}
                  />
                  {tool.description}
                </button>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}
