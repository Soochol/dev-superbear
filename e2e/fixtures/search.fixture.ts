import { test as base } from "@playwright/test";
import { SearchPage } from "../pages/search.page";
import { SearchApiMock } from "../mocks/search-api.mock";

interface SearchFixtures {
  searchPage: SearchPage;
  searchApiMock: SearchApiMock;
}

export const test = base.extend<SearchFixtures>({
  searchPage: async ({ page }, use) => {
    await use(new SearchPage(page));
  },
  searchApiMock: async ({ page }, use) => {
    await use(new SearchApiMock(page));
  },
});

export { expect } from "@playwright/test";
