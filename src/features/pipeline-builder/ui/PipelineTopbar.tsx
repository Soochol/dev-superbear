"use client";

interface PipelineTopbarProps {
  pipelineName: string;
  onPipelineNameChange: (name: string) => void;
  selectedSymbol: string;
  onSymbolChange: (symbol: string) => void;
  onOpenAIGenerate: () => void;
  onRegisterAndRun: () => void;
  isRunning: boolean;
}

export default function PipelineTopbar({
  pipelineName,
  onPipelineNameChange,
  selectedSymbol,
  onSymbolChange,
  onOpenAIGenerate,
  onRegisterAndRun,
  isRunning,
}: PipelineTopbarProps) {
  return (
    <header className="flex items-center gap-3 border-b border-nexus-border bg-nexus-surface px-4 py-2.5 shrink-0">
      {/* Pipeline name */}
      <input
        type="text"
        value={pipelineName}
        onChange={(e) => onPipelineNameChange(e.target.value)}
        placeholder="Pipeline Name"
        className="bg-nexus-bg border border-nexus-border rounded-md px-3 py-1.5 text-sm text-nexus-text-primary placeholder:text-nexus-text-muted/50 w-56 focus:outline-none focus:border-nexus-accent"
      />

      {/* Symbol input */}
      <input
        type="text"
        value={selectedSymbol}
        onChange={(e) => onSymbolChange(e.target.value.toUpperCase())}
        placeholder="Symbol (e.g. 005930)"
        className="bg-nexus-bg border border-nexus-border rounded-md px-3 py-1.5 text-sm font-mono text-nexus-text-primary placeholder:text-nexus-text-muted/50 w-44 focus:outline-none focus:border-nexus-accent"
      />

      <div className="flex-1" />

      {/* AI Generate button */}
      <button
        type="button"
        onClick={onOpenAIGenerate}
        className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-nexus-accent border border-nexus-accent/30 rounded-md hover:bg-nexus-accent/10 transition-colors"
      >
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M12 2v4" />
          <path d="m16.2 7.8 2.9-2.9" />
          <path d="M18 12h4" />
          <path d="m16.2 16.2 2.9 2.9" />
          <path d="M12 18v4" />
          <path d="m4.9 19.1 2.9-2.9" />
          <path d="M2 12h4" />
          <path d="m4.9 4.9 2.9 2.9" />
        </svg>
        AI Generate
      </button>

      {/* Register & Run button */}
      <button
        type="button"
        onClick={onRegisterAndRun}
        disabled={isRunning}
        className="flex items-center gap-1.5 px-4 py-1.5 text-sm bg-nexus-accent hover:bg-nexus-accent/80 text-white rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {isRunning ? (
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
            Running...
          </>
        ) : (
          <>
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <polygon points="6 3 20 12 6 21 6 3" />
            </svg>
            Register &amp; Run
          </>
        )}
      </button>
    </header>
  );
}
