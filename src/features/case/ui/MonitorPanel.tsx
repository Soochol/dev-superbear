"use client";
import { useEffect, useState } from "react";
import styles from "./MonitorPanel.module.css";

interface MonitorBlock {
  id: string;
  enabled: boolean;
  cron: string;
  last_executed_at: string | null;
  instruction: string;
}

export function MonitorPanel({ caseId }: { caseId: string }) {
  const [blocks, setBlocks] = useState<MonitorBlock[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch(`/api/v1/cases/${caseId}/monitors`, { credentials: "include" })
      .then((r) => (r.ok ? r.json() : { data: [] }))
      .then((res) => setBlocks(res.data ?? []))
      .finally(() => setLoading(false));
  }, [caseId]);

  const toggle = async (blockId: string, enabled: boolean) => {
    await fetch(`/api/v1/cases/${caseId}/monitors/${blockId}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ enabled }),
    });
    setBlocks((prev) =>
      prev.map((b) => (b.id === blockId ? { ...b, enabled } : b))
    );
  };

  if (loading) return <div className={styles.loading}>로딩 중...</div>;
  if (blocks.length === 0) return <div className={styles.empty}>모니터 블록이 없습니다</div>;

  return (
    <div className={styles.panel} data-testid="monitor-panel">
      <h3 className={styles.title}>모니터링 블록</h3>
      <ul className={styles.list}>
        {blocks.map((b) => (
          <li key={b.id} className={styles.item}>
            <div className={styles.info}>
              <span className={styles.instruction}>{b.instruction}</span>
              <span className={styles.cron}>{b.cron}</span>
              {b.last_executed_at && (
                <span className={styles.lastRun}>
                  마지막 실행: {new Date(b.last_executed_at).toLocaleString("ko-KR")}
                </span>
              )}
            </div>
            <button
              className={`${styles.toggle} ${b.enabled ? styles.on : styles.off}`}
              onClick={() => toggle(b.id, !b.enabled)}
              data-testid={`toggle-${b.id}`}
            >
              {b.enabled ? "ON" : "OFF"}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
