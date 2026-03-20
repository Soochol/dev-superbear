"use client";

interface DSLConditionEditorProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  accent: "success" | "failure";
  placeholder?: string;
}

export default function DSLConditionEditor({
  label,
  value,
  onChange,
  accent,
  placeholder,
}: DSLConditionEditorProps) {
  const accentColor =
    accent === "success" ? "text-nexus-success" : "text-nexus-failure";
  const borderAccent =
    accent === "success"
      ? "focus:border-nexus-success/50"
      : "focus:border-nexus-failure/50";

  return (
    <div className="flex-1">
      <label
        className={`block text-xs font-medium mb-1.5 ${accentColor}`}
      >
        {label}
      </label>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder ?? `Enter DSL condition...`}
        rows={4}
        className={`w-full bg-nexus-bg border border-nexus-border rounded-md px-3 py-2 text-sm font-mono text-nexus-text-primary placeholder:text-nexus-text-muted/50 resize-none focus:outline-none ${borderAccent}`}
      />
    </div>
  );
}
