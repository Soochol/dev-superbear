import { ApiError, apiClient, apiGet, apiPost, apiPut, apiDelete } from "../client";

// Mock fetch globally
const mockFetch = jest.fn();
global.fetch = mockFetch;

beforeEach(() => {
  mockFetch.mockClear();
});

describe("ApiError", () => {
  it("includes status and body", () => {
    const err = new ApiError(404, "Not found");
    expect(err.status).toBe(404);
    expect(err.body).toBe("Not found");
    expect(err.message).toContain("404");
    expect(err.name).toBe("ApiError");
  });
});

describe("apiClient", () => {
  it("returns parsed JSON on success", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ id: 1, name: "test" }),
    });
    const result = await apiGet("/test");
    expect(result).toEqual({ id: 1, name: "test" });
  });

  it("throws ApiError on non-ok response", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      text: () => Promise.resolve("Internal Server Error"),
    });
    await expect(apiGet("/test")).rejects.toThrow(ApiError);
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      text: () => Promise.resolve("Internal Server Error"),
    });
    try {
      await apiGet("/fail");
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      // Reset for the catch
    }
  });

  it("throws ApiError with status 0 on network error", async () => {
    mockFetch.mockRejectedValueOnce(new TypeError("Failed to fetch"));
    try {
      await apiGet("/test");
      throw new Error("should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      expect((e as ApiError).status).toBe(0);
      expect((e as ApiError).body).toContain("Network error");
    }
  });

  it("includes credentials in requests", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({}),
    });
    await apiGet("/test");
    expect(mockFetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ credentials: "include" })
    );
  });

  it("does NOT set Content-Type on GET requests", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({}),
    });
    await apiGet("/test");
    const callArgs = mockFetch.mock.calls[0][1];
    expect(callArgs.headers["Content-Type"]).toBeUndefined();
  });

  it("sets Content-Type on POST requests with body", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({}),
    });
    await apiPost("/test", { key: "value" });
    const callArgs = mockFetch.mock.calls[0][1];
    expect(callArgs.headers["Content-Type"]).toBe("application/json");
  });

  it("sends JSON stringified body on POST", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({}),
    });
    await apiPost("/test", { key: "value" });
    const callArgs = mockFetch.mock.calls[0][1];
    expect(callArgs.body).toBe('{"key":"value"}');
  });
});

describe("HTTP method helpers", () => {
  it("apiGet sends GET method", async () => {
    mockFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) });
    await apiGet("/test");
    expect(mockFetch.mock.calls[0][1].method).toBe("GET");
  });

  it("apiPost sends POST method", async () => {
    mockFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) });
    await apiPost("/test", {});
    expect(mockFetch.mock.calls[0][1].method).toBe("POST");
  });

  it("apiPut sends PUT method", async () => {
    mockFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) });
    await apiPut("/test", {});
    expect(mockFetch.mock.calls[0][1].method).toBe("PUT");
  });

  it("apiDelete sends DELETE method", async () => {
    mockFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) });
    await apiDelete("/test");
    expect(mockFetch.mock.calls[0][1].method).toBe("DELETE");
  });
});
