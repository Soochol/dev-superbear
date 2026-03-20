import { test as base } from "@playwright/test";
import { SignJWT } from "jose";
import { SearchPage } from "../pages/search.page";

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

interface SearchFixtures {
  searchPage: SearchPage;
}

export const test = base.extend<SearchFixtures>({
  searchPage: async ({ page }, use) => {
    // Inject Authorization header into all API requests.
    // The actual request still goes to the real backend — this is not mocking.
    const token = await generateTestToken();
    await page.route("**/api/**", async (route) => {
      const headers = {
        ...route.request().headers(),
        authorization: `Bearer ${token}`,
      };
      await route.continue({ headers });
    });
    await use(new SearchPage(page));
  },
});

export { expect } from "@playwright/test";
