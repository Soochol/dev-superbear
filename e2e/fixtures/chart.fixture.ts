import { test as base } from "@playwright/test";
import { ChartPage } from "../pages/chart.page";

interface ChartFixtures {
  chartPage: ChartPage;
}

export const test = base.extend<ChartFixtures>({
  chartPage: async ({ page }, use) => {
    await use(new ChartPage(page));
  },
});

export { expect } from "@playwright/test";
