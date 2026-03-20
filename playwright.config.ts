import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30000,
  retries: 0,
  use: {
    headless: true,
    screenshot: "only-on-failure",
  },
  webServer: [
    {
      command: "npm run dev -- --port 3000",
      port: 3000,
      timeout: 30000,
      reuseExistingServer: true,
    },
    {
      command:
        "WATCHPACK_POLLING=true npx next dev --webpack --port 3001",
      port: 3001,
      timeout: 60000,
      reuseExistingServer: true,
      cwd: "./frontend",
    },
  ],
  projects: [
    {
      name: "root-app",
      use: {
        browserName: "chromium",
        baseURL: "http://localhost:3000",
      },
      testMatch: /landing\.spec\.ts/,
    },
    {
      name: "frontend-app",
      use: {
        browserName: "chromium",
        baseURL: "http://localhost:3001",
      },
      testMatch: /search.*\.spec\.ts/,
    },
    {
      name: "chart-app",
      use: {
        browserName: "chromium",
        baseURL: "http://localhost:3000",
      },
      testMatch: /chart.*\.spec\.ts/,
    },
    {
      name: "monitoring-api",
      use: {
        baseURL: "http://localhost:8080",
      },
      testMatch: /monitoring-api\.spec\.ts/,
    },
  ],
});
