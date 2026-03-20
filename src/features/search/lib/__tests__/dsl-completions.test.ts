import { getContextualCompletions } from "../dsl-completions";

describe("getContextualCompletions", () => {
  it("suggests 'scan' at empty input", () => {
    const items = getContextualCompletions("");
    expect(items.map((c) => c.label)).toEqual(["scan"]);
  });

  it("suggests 'where' after scan", () => {
    const items = getContextualCompletions("scan ");
    expect(items.map((c) => c.label)).toEqual(["where"]);
  });

  it("suggests fields after 'where'", () => {
    const items = getContextualCompletions("scan where ");
    const labels = items.map((c) => c.label);
    expect(labels).toContain("volume");
    expect(labels).toContain("close");
    expect(labels).toContain("change_pct");
    expect(labels).not.toContain("market_cap");
    expect(labels).not.toContain("ma");
  });

  it("suggests operators after field", () => {
    const items = getContextualCompletions("scan where volume ");
    const labels = items.map((c) => c.label);
    expect(labels).toContain(">");
    expect(labels).toContain(">=");
  });

  it("suggests 'and', 'sort', 'limit' after value", () => {
    const items = getContextualCompletions("scan where volume > 1000000 ");
    const labels = items.map((c) => c.label);
    expect(labels).toContain("and");
    expect(labels).toContain("sort");
    expect(labels).toContain("limit");
  });

  it("suggests fields after 'and'", () => {
    const items = getContextualCompletions("scan where volume > 1000000 and ");
    const labels = items.map((c) => c.label);
    expect(labels).toContain("close");
  });

  it("suggests 'by' after 'sort'", () => {
    const items = getContextualCompletions("scan where volume > 1000000 sort ");
    expect(items.map((c) => c.label)).toEqual(["by"]);
  });

  it("suggests fields after 'sort by'", () => {
    const items = getContextualCompletions("scan where volume > 1000000 sort by ");
    const labels = items.map((c) => c.label);
    expect(labels).toContain("volume");
    expect(labels).toContain("trade_value");
  });

  it("suggests 'asc'/'desc' after sort field", () => {
    const items = getContextualCompletions("scan where volume > 1000000 sort by volume ");
    const labels = items.map((c) => c.label);
    expect(labels).toContain("asc");
    expect(labels).toContain("desc");
  });
});
