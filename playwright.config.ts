import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30000,
  expect: {
    timeout: 5000,
  },
  retries: 0,
  use: {
    headless: true,
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "root-app",
      use: {
        browserName: "chromium",
        baseURL: `http://localhost:${process.env.E2E_PORT_ROOT ?? 3100}`,
      },
      testMatch: /landing\.spec\.ts/,
    },
    {
      name: "frontend-app",
      use: {
        browserName: "chromium",
        baseURL: `http://localhost:${process.env.E2E_PORT_FRONT ?? 3200}`,
      },
      testMatch: /search.*\.spec\.ts/,
    },
    {
      name: "chart-app",
      use: {
        browserName: "chromium",
        baseURL: `http://localhost:${process.env.E2E_PORT_ROOT ?? 3100}`,
      },
      testMatch: /chart.*\.spec\.ts/,
    },
    {
      name: "monitoring-api",
      use: {
        baseURL: `http://localhost:${process.env.E2E_PORT_API ?? 3300}`,
      },
      testMatch: /monitoring-api\.spec\.ts/,
    },
    {
      name: "case-app",
      use: {
        browserName: "chromium",
        baseURL: `http://localhost:${process.env.E2E_PORT_ROOT ?? 3100}`,
      },
      testMatch: /monitoring-visual\.spec\.ts/,
    },
  ],
});
