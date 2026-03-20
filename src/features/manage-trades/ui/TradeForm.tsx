'use client';

import { useState } from 'react';
import { useTradeStore } from '../model/trade.store';

interface TradeFormProps {
  caseId: string;
  onDone: () => void;
}

export function TradeForm({ caseId, onDone }: TradeFormProps) {
  const { addTrade } = useTradeStore();
  const [type, setType] = useState<'BUY' | 'SELL'>('BUY');
  const [price, setPrice] = useState('');
  const [quantity, setQuantity] = useState('');
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await addTrade(caseId, {
      type,
      price: Number(price),
      quantity: Number(quantity),
      date,
    });
    onDone();
  };

  return (
    <form onSubmit={handleSubmit} className="mt-2 space-y-2 p-3 rounded border border-nexus-border bg-nexus-surface">
      <div className="flex gap-1">
        {(['BUY', 'SELL'] as const).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setType(t)}
            className={`px-3 py-1 text-xs rounded ${
              type === t
                ? t === 'BUY' ? 'bg-nexus-success/20 text-nexus-success' : 'bg-nexus-failure/20 text-nexus-failure'
                : 'text-nexus-text-muted hover:bg-nexus-border/50'
            }`}
          >
            {t}
          </button>
        ))}
      </div>
      <div className="grid grid-cols-3 gap-2">
        <input type="number" placeholder="Price" value={price} onChange={(e) => setPrice(e.target.value)}
          className="bg-nexus-bg border border-nexus-border rounded px-2 py-1 text-xs text-nexus-text-primary" required />
        <input type="number" placeholder="Qty" value={quantity} onChange={(e) => setQuantity(e.target.value)}
          className="bg-nexus-bg border border-nexus-border rounded px-2 py-1 text-xs text-nexus-text-primary" required />
        <input type="date" value={date} onChange={(e) => setDate(e.target.value)}
          className="bg-nexus-bg border border-nexus-border rounded px-2 py-1 text-xs text-nexus-text-primary" required />
      </div>
      <div className="flex gap-2">
        <button type="submit" className="px-3 py-1 text-xs rounded bg-nexus-accent text-white hover:bg-nexus-accent/80">Save</button>
        <button type="button" onClick={onDone} className="px-3 py-1 text-xs text-nexus-text-muted hover:text-nexus-text-secondary">Cancel</button>
      </div>
    </form>
  );
}
