"use client";

export function AIFusionPanel({ symbol }: { symbol: string }) {
  return (
    <div className="p-4">
      <h3 className="text-xs font-semibold text-nexus-text-secondary uppercase mb-3">
        AI Fusion
      </h3>
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <span className="text-sm text-nexus-text-secondary">Signal</span>
          <span className="px-2 py-0.5 text-xs rounded-full bg-nexus-text-muted/20 text-nexus-text-muted">
            No analysis yet
          </span>
        </div>
        <div>
          <span className="text-sm text-nexus-text-secondary">Tags</span>
          <div className="flex flex-wrap gap-1 mt-1">
            <span className="text-xs text-nexus-text-muted">
              Run a pipeline to generate AI analysis
            </span>
          </div>
        </div>
        <div>
          <span className="text-sm text-nexus-text-secondary">Summary</span>
          <p className="text-xs text-nexus-text-muted mt-1">
            Execute a pipeline on this stock to see AI-generated cross-analysis of fundamental and technical factors.
          </p>
        </div>
      </div>
    </div>
  );
}
