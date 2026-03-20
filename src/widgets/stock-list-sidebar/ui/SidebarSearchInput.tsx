"use client";

interface Props {
  value: string;
  onChange: (value: string) => void;
}

export function SidebarSearchInput({ value, onChange }: Props) {
  return (
    <div className="p-2 border-b border-nexus-border">
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="종목 검색..."
        className="w-full px-2 py-1 text-sm bg-nexus-bg border border-nexus-border rounded text-nexus-text-primary placeholder-nexus-text-muted focus:outline-none focus:border-nexus-accent"
      />
    </div>
  );
}
