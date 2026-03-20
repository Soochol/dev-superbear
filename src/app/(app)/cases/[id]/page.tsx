import { MonitorPanel } from "@/features/case/ui/MonitorPanel";

export default async function CaseDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return (
    <div style={{ padding: "2rem", background: "#111827", minHeight: "100vh", color: "#e5e7eb" }}>
      <h1 style={{ fontSize: "1.5rem", marginBottom: "1rem" }}>케이스 상세</h1>
      <MonitorPanel caseId={id} />
    </div>
  );
}
