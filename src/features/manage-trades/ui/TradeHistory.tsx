'use client';

import { useEffect, useState } from 'react';
import { useTradeStore } from '../model/trade.store';
import { TradeForm } from './TradeForm';

interface TradeHistoryProps {
  caseId: string;
  isClosed: boolean;
}

export function TradeHistory({ caseId, isClosed }: TradeHistoryProps) {
  const { trades, summary, fetchTrades } = useTradeStore();
  const [showForm, setShowForm] = useState(false);

  useEffect(() => {
    fetchTrades(caseId);
  }, [caseId, fetchTrades]);

  return (
    <div>
      {trades.length > 0 && (
        <table className="w-full text-xs mb-3">
          <thead>
            <tr className="text-nexus-text-muted border-b border-nexus-border">
              <th className="text-left py-1.5 font-medium">Date</th>
              <th className="text-left py-1.5 font-medium">Type</th>
              <th className="text-right py-1.5 font-medium">Price</th>
              <th className="text-right py-1.5 font-medium">Qty</th>
            </tr>
          </thead>
          <tbody>
            {trades.map((t) => (
              <tr key={t.id} className="border-b border-nexus-border/50">
                <td className="py-1.5 text-nexus-text-secondary">{t.date}</td>
                <td className={`py-1.5 ${t.type === 'BUY' ? 'text-nexus-success' : 'text-nexus-failure'}`}>{t.type}</td>
                <td className="py-1.5 text-right font-mono text-nexus-text-primary">{t.price.toLocaleString()}</td>
                <td className="py-1.5 text-right font-mono text-nexus-text-secondary">{t.quantity}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {summary && (
        <div className="space-y-1 text-xs border-t border-nexus-border pt-2 mb-3">
          <div className="flex justify-between">
            <span className="text-nexus-text-muted">Avg Buy</span>
            <span className="font-mono text-nexus-text-primary">{summary.average_buy_price.toLocaleString()}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-nexus-text-muted">Remaining</span>
            <span className="font-mono text-nexus-text-secondary">{summary.remaining_quantity}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-nexus-text-muted">Realized P&L</span>
            <span className={`font-mono ${summary.realized_pnl >= 0 ? 'text-nexus-success' : 'text-nexus-failure'}`}>
              {summary.realized_pnl >= 0 ? '+' : ''}{summary.realized_pnl.toLocaleString()} ({summary.realized_return.toFixed(1)}%)
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-nexus-text-muted">Unrealized P&L</span>
            <span className={`font-mono ${summary.unrealized_pnl >= 0 ? 'text-nexus-success' : 'text-nexus-failure'}`}>
              {summary.unrealized_pnl >= 0 ? '+' : ''}{summary.unrealized_pnl.toLocaleString()} ({summary.unrealized_return.toFixed(1)}%)
            </span>
          </div>
        </div>
      )}

      {!isClosed && (
        <button
          onClick={() => setShowForm(!showForm)}
          className="text-xs text-nexus-accent hover:text-nexus-accent/80"
        >
          + Add Trade
        </button>
      )}

      {showForm && <TradeForm caseId={caseId} onDone={() => { setShowForm(false); fetchTrades(caseId); }} />}
    </div>
  );
}
