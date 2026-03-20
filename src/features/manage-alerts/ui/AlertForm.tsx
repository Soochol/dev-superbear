'use client';

import { useState } from 'react';
import { useAlertStore } from '../model/alert.store';

interface AlertFormProps {
  caseId: string;
  onDone: () => void;
}

export function AlertForm({ caseId, onDone }: AlertFormProps) {
  const { addAlert } = useAlertStore();
  const [condition, setCondition] = useState('');
  const [label, setLabel] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await addAlert(caseId, condition, label);
    onDone();
  };

  return (
    <form onSubmit={handleSubmit} className="mt-2 space-y-2 p-3 rounded border border-nexus-border bg-nexus-surface">
      <input type="text" placeholder="Condition (e.g. close >= 200000)" value={condition} onChange={(e) => setCondition(e.target.value)}
        className="w-full bg-nexus-bg border border-nexus-border rounded px-2 py-1 text-xs text-nexus-text-primary" required />
      <input type="text" placeholder="Label (e.g. 목표가 도달)" value={label} onChange={(e) => setLabel(e.target.value)}
        className="w-full bg-nexus-bg border border-nexus-border rounded px-2 py-1 text-xs text-nexus-text-primary" required />
      <div className="flex gap-2">
        <button type="submit" className="px-3 py-1 text-xs rounded bg-nexus-accent text-white hover:bg-nexus-accent/80">Add</button>
        <button type="button" onClick={onDone} className="px-3 py-1 text-xs text-nexus-text-muted hover:text-nexus-text-secondary">Cancel</button>
      </div>
    </form>
  );
}
