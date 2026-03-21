import { lintDSL, type DSLDiagnostic } from "../dsl-linter";

describe("lintDSL", () => {
  it("returns no diagnostics for valid DSL", () => {
    expect(lintDSL("scan where volume > 1000000")).toEqual([]);
  });

  it("returns no diagnostics for valid DSL with sort and limit", () => {
    expect(lintDSL("scan where volume > 1000000 sort by trade_value desc limit 50")).toEqual([]);
  });

  it("reports missing 'scan' keyword", () => {
    const diags = lintDSL("where volume > 100");
    expect(diags).toHaveLength(1);
    expect(diags[0].message).toContain("scan");
  });

  it("reports missing 'where' keyword", () => {
    const diags = lintDSL("scan volume > 100");
    expect(diags).toHaveLength(1);
    expect(diags[0].message).toContain("where");
  });

  it("reports unknown field", () => {
    const diags = lintDSL("scan where market_cap > 100");
    expect(diags).toHaveLength(1);
    expect(diags[0].message).toContain("market_cap");
  });

  it("reports OR not supported", () => {
    const diags = lintDSL("scan where volume > 100 or close > 50000");
    expect(diags).toHaveLength(1);
    expect(diags[0].message).toContain("OR");
  });

  it("returns empty for empty input", () => {
    expect(lintDSL("")).toEqual([]);
  });

  it("reports error position", () => {
    const diags = lintDSL("scan where unknown_field > 100");
    expect(diags[0].from).toBeGreaterThan(0);
    expect(diags[0].to).toBeGreaterThan(diags[0].from);
  });
});
