import type { Page } from "@playwright/test";

interface NlSearchResponse {
  dsl: string;
  explanation: string;
  results: Array<{
    symbol: string;
    name: string;
    matchedValue: number;
    close: number;
    changePct: number;
  }>;
}

interface DslExecuteResponse {
  results: Array<{
    symbol: string;
    name: string;
    matchedValue: number;
    close: number;
    changePct: number;
  }>;
}

interface ValidateResponse {
  valid: boolean;
  error: string | null;
}

export class SearchApiMock {
  constructor(readonly page: Page) {}

  async mockNlSearch(response: NlSearchResponse) {
    await this.page.route("**/api/v1/search/nl-to-dsl", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(response),
      });
    });
  }

  async mockDslExecute(response: DslExecuteResponse) {
    await this.page.route("**/api/v1/search/execute", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(response),
      });
    });
  }

  async mockValidate(response: ValidateResponse) {
    await this.page.route("**/api/v1/search/validate", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(response),
      });
    });
  }

  async mockPresetSave() {
    await this.page.route("**/api/v1/search/presets", async (route) => {
      if (route.request().method() === "POST") {
        const body = route.request().postDataJSON();
        await route.fulfill({
          status: 201,
          contentType: "application/json",
          body: JSON.stringify({
            data: {
              id: "new-preset-1",
              userId: "u1",
              name: body.name,
              dsl: body.dsl,
              nlQuery: body.nlQuery ?? null,
              isPublic: false,
              createdAt: new Date().toISOString(),
              updatedAt: new Date().toISOString(),
            },
          }),
        });
      } else {
        await route.continue();
      }
    });
  }
}
