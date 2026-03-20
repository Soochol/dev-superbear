'use client';

import { useEffect, useState } from 'react';
import { useAlertStore } from '../model/alert.store';
import { AlertForm } from './AlertForm';

interface PriceAlertsListProps {
  caseId: string;
  isClosed: boolean;
}

export function PriceAlertsList({ caseId, isClosed }: PriceAlertsListProps) {
  const { pendingAlerts, triggeredAlerts, fetchAlerts, deleteAlert } = useAlertStore();
  const [showForm, setShowForm] = useState(false);

  useEffect(() => {
    fetchAlerts(caseId);
  }, [caseId, fetchAlerts]);

  return (
    <div>
      {pendingAlerts.length > 0 && (
        <div className="mb-3">
          <p className="text-xs text-nexus-text-muted mb-1">Pending</p>
          {pendingAlerts.map((a) => (
            <div key={a.id} className="flex items-center justify-between py-1 text-xs">
              <div className="flex items-center gap-2">
                <span className="text-nexus-warning">○</span>
                <span className="text-nexus-text-secondary font-mono">{a.condition}</span>
                <span className="text-nexus-text-muted">{a.label}</span>
              </div>
              {!isClosed && (
                <button onClick={() => deleteAlert(caseId, a.id)} className="text-nexus-text-muted hover:text-nexus-failure">×</button>
              )}
            </div>
          ))}
        </div>
      )}

      {triggeredAlerts.length > 0 && (
        <div className="mb-3">
          <p className="text-xs text-nexus-text-muted mb-1">Triggered</p>
          {triggeredAlerts.map((a) => (
            <div key={a.id} className="flex items-center gap-2 py-1 text-xs">
              <span className="text-nexus-success">●</span>
              <span className="text-nexus-text-secondary font-mono">{a.condition}</span>
              <span className="text-nexus-text-muted">{a.label}</span>
              {a.triggered_at && <span className="text-nexus-text-muted">{a.triggered_at}</span>}
            </div>
          ))}
        </div>
      )}

      {pendingAlerts.length === 0 && triggeredAlerts.length === 0 && (
        <p className="text-xs text-nexus-text-muted">No alerts</p>
      )}

      {!isClosed && (
        <button onClick={() => setShowForm(!showForm)} className="text-xs text-nexus-accent hover:text-nexus-accent/80 mt-1">
          + Add Alert
        </button>
      )}

      {showForm && <AlertForm caseId={caseId} onDone={() => { setShowForm(false); fetchAlerts(caseId); }} />}
    </div>
  );
}
