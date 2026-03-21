import { test as base } from "@playwright/test";
import { SignJWT } from "jose";
import { ChartPage } from "../pages/chart.page";

const JWT_SECRET = "test-secret";
const TEST_USER_ID = "00000000-0000-0000-0000-000000000001";
const TEST_USER_EMAIL = "e2e@test.com";

async function generateTestToken(): Promise<string> {
  const secret = new TextEncoder().encode(JWT_SECRET);
  return new SignJWT({ userId: TEST_USER_ID, email: TEST_USER_EMAIL })
    .setProtectedHeader({ alg: "HS256" })
    .setIssuer("nexus")
    .setSubject(TEST_USER_ID)
    .setIssuedAt()
    .setExpirationTime("1h")
    .sign(secret);
}

interface ChartFixtures {
  chartPage: ChartPage;
}

export const test = base.extend<ChartFixtures>({
  chartPage: async ({ page }, use) => {
    const token = await generateTestToken();
    await page.route("**/api/**", async (route) => {
      const headers = {
        ...route.request().headers(),
        authorization: `Bearer ${token}`,
      };
      await route.continue({ headers });
    });
    await use(new ChartPage(page));
  },
});

export { expect } from "@playwright/test";
