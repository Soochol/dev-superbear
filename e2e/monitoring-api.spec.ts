import { test, expect } from "@playwright/test";

test.describe("Monitoring API E2E", () => {
  test("GET /health returns ok", async ({ request }) => {
    const res = await request.get("/api/v1/health");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.status).toBe("ok");
  });

  test("GET /cases/:id/monitors without auth returns 401", async ({ request }) => {
    const res = await request.get(
      "/api/v1/cases/00000000-0000-0000-0000-000000000100/monitors"
    );
    expect(res.status()).toBe(401);
  });

  test("PATCH /cases/:id/monitoring-status without auth returns 401", async ({ request }) => {
    const res = await request.patch(
      "/api/v1/cases/00000000-0000-0000-0000-000000000100/monitoring-status",
      { data: { enabled: false } }
    );
    expect(res.status()).toBe(401);
  });

  test("Worker health endpoint returns metrics", async ({ request }) => {
    const res = await request.get(`http://localhost:${process.env.E2E_PORT_WORKER ?? 3400}/api/health/workers`);
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.status).toBe("ok");
    expect(body.metrics).toBeDefined();
  });
});
