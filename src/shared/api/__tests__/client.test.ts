import { ApiError } from "../client";

describe("API Client", () => {
  it("ApiError includes status and body", () => {
    const err = new ApiError(404, "Not found");
    expect(err.status).toBe(404);
    expect(err.body).toBe("Not found");
    expect(err.message).toContain("404");
  });
});
