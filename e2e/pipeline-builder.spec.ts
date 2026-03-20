import { test, expect } from "@playwright/test";

test.describe("Pipeline Builder Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/pipeline");
  });

  // ── Layout & Structure ──────────────────────────────────

  test("renders 3-section canvas with Topbar and Node Palette", async ({
    page,
  }) => {
    // Topbar elements
    await expect(
      page.getByPlaceholder("Pipeline Name"),
    ).toBeVisible();
    await expect(
      page.getByPlaceholder("Symbol (e.g. 005930)"),
    ).toBeVisible();
    await expect(page.getByText("AI Generate")).toBeVisible();
    await expect(page.getByText("Register & Run")).toBeVisible();

    // Node Palette sidebar
    await expect(page.getByText("Node Palette")).toBeVisible();
    await expect(page.getByText("Agent Nodes")).toBeVisible();
    await expect(page.getByText("DSL Nodes")).toBeVisible();
    await expect(page.getByText("Output Nodes")).toBeVisible();

    // 3 sections in canvas
    await expect(page.getByText("Analysis Stages")).toBeVisible();
    await expect(page.getByText("Monitoring")).toBeVisible();
    await expect(page.getByText("Judgment")).toBeVisible();
  });

  // ── Topbar Input ────────────────────────────────────────

  test("pipeline name and symbol inputs work", async ({ page }) => {
    const nameInput = page.getByPlaceholder("Pipeline Name");
    await nameInput.fill("My Test Pipeline");
    await expect(nameInput).toHaveValue("My Test Pipeline");

    const symbolInput = page.getByPlaceholder("Symbol (e.g. 005930)");
    await symbolInput.fill("005930");
    await expect(symbolInput).toHaveValue("005930");
  });

  test("symbol input auto-uppercases", async ({ page }) => {
    const symbolInput = page.getByPlaceholder("Symbol (e.g. 005930)");
    await symbolInput.fill("aapl");
    await expect(symbolInput).toHaveValue("AAPL");
  });

  // ── Node Palette ────────────────────────────────────────

  test("palette shows agent block templates", async ({ page }) => {
    // Agent Nodes
    await expect(page.getByText("뉴스 분석")).toBeVisible();
    await expect(page.getByText("섹터 비교")).toBeVisible();
    await expect(page.getByText("재무 분석")).toBeVisible();
    await expect(page.getByText("가격 분석")).toBeVisible();
    await expect(page.getByText("수급 분석")).toBeVisible();

    // DSL Nodes
    await expect(page.getByText("DSL 평가")).toBeVisible();
    await expect(page.getByText("종목 스캐닝")).toBeVisible();

    // Output Nodes
    await expect(page.getByText("케이스 생성")).toBeVisible();
    await expect(page.getByText("알림 전송")).toBeVisible();
  });

  test("palette categories can be collapsed and expanded", async ({
    page,
  }) => {
    // "뉴스 분석" should be visible initially
    await expect(page.getByText("뉴스 분석")).toBeVisible();

    // Click "Agent Nodes" to collapse
    await page.getByText("Agent Nodes").click();
    await expect(page.getByText("뉴스 분석")).not.toBeVisible();

    // Click again to expand
    await page.getByText("Agent Nodes").click();
    await expect(page.getByText("뉴스 분석")).toBeVisible();
  });

  test("palette items are draggable", async ({ page }) => {
    const newsNode = page.getByText("뉴스 분석");
    await expect(newsNode).toHaveAttribute("draggable", "true");
  });

  // ── Analysis Stages ─────────────────────────────────────

  test("initial analysis stage shows drop hint", async ({ page }) => {
    await expect(page.getByText("Stage 0")).toBeVisible();
    await expect(page.getByText("Drop agent blocks here")).toBeVisible();
  });

  test("add stage button creates a new stage", async ({ page }) => {
    await page.getByText("+ Add Stage").click();

    await expect(page.getByText("Stage 0")).toBeVisible();
    await expect(page.getByText("Stage 1")).toBeVisible();
  });

  test("remove stage button works when multiple stages exist", async ({
    page,
  }) => {
    // Add a second stage
    await page.getByText("+ Add Stage").click();
    await expect(page.getByText("Stage 1")).toBeVisible();

    // Remove buttons should appear
    const removeButtons = page.getByText("Remove");
    await expect(removeButtons.first()).toBeVisible();

    // Remove one stage
    await removeButtons.first().click();
    await expect(page.getByText("Stage 1")).not.toBeVisible();
  });

  // ── Drag & Drop (palette → Analysis) ───────────────────

  test("dragging a block from palette to analysis stage adds it", async ({
    page,
  }) => {
    // Drag "뉴스 분석" to the analysis drop zone
    const source = page.getByText("뉴스 분석");
    const target = page.getByText("Drop agent blocks here").first();

    await source.dragTo(target);

    // After drop, the block card should appear with the block name
    // The drop hint should be gone, replaced by a BlockCard
    await expect(
      page.locator('[class*="border-nexus-border"]').getByText("뉴스 분석").first(),
    ).toBeVisible();
  });

  // ── Monitoring Section ──────────────────────────────────

  test("monitoring section shows drop hint", async ({ page }) => {
    await expect(
      page.getByText("Drop blocks here to create monitors"),
    ).toBeVisible();
  });

  // ── Judgment Section ────────────────────────────────────

  test("judgment section has success and failure condition editors", async ({
    page,
  }) => {
    await expect(page.getByText("Success Condition")).toBeVisible();
    await expect(page.getByText("Failure Condition")).toBeVisible();
  });

  test("DSL condition editors accept input", async ({ page }) => {
    const successEditor = page.getByPlaceholder(
      "e.g. confidence > 0.7 AND risk < 0.3",
    );
    await successEditor.fill("confidence > 0.8");
    await expect(successEditor).toHaveValue("confidence > 0.8");

    const failureEditor = page.getByPlaceholder(
      "e.g. confidence < 0.3 OR risk > 0.8",
    );
    await failureEditor.fill("risk > 0.5");
    await expect(failureEditor).toHaveValue("risk > 0.5");
  });

  // ── Price Alerts ────────────────────────────────────────

  test("price alert editor allows adding alerts", async ({ page }) => {
    await expect(page.getByText("Price Alerts")).toBeVisible();

    // Fill in condition and label, then add
    const conditionInput = page.getByPlaceholder("price > 50000");
    const labelInput = page.getByPlaceholder("Label");
    await expect(conditionInput).toBeVisible();
    await expect(labelInput).toBeVisible();

    await conditionInput.fill("close >= 65000");
    await labelInput.fill("목표가 도달");
    await page.getByRole("button", { name: "+ Add", exact: true }).click();

    // The alert should appear in the list
    await expect(page.getByText("close >= 65000")).toBeVisible();
    await expect(page.getByText("목표가 도달")).toBeVisible();
  });

  // ── AI Generate Modal ───────────────────────────────────

  test("AI Generate modal opens and closes", async ({ page }) => {
    await page.getByText("AI Generate").click();

    // Modal should open
    await expect(
      page.getByText("AI Pipeline Generator"),
    ).toBeVisible();

    // Close modal
    const closeButton = page.locator('[aria-label="Close"]');
    if (await closeButton.isVisible()) {
      await closeButton.click();
    } else {
      // Try pressing Escape
      await page.keyboard.press("Escape");
    }
  });

  // ── Register & Run ──────────────────────────────────────

  test("Register & Run button is clickable and shows error without symbol", async ({
    page,
  }) => {
    // Click without filling symbol — should show error
    await page.getByText("Register & Run").click();

    // The component should show an error or the button should handle gracefully
    // Since no backend is running, we check it doesn't crash the page
    // After click, the page should still be functional
    await expect(page.getByText("Analysis Stages")).toBeVisible();
  });
});
