import { ApiError, apiClient, apiGet, apiPost, apiPut, apiDelete } from "../client";

const mockFetch = jest.fn();
global.fetch = mockFetch;

function jsonResponse(data: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  };
}

beforeEach(() => {
  mockFetch.mockReset();
});

describe("API Client", () => {
  it("ApiError includes status and body", () => {
    const err = new ApiError(404, "Not found");
    expect(err.status).toBe(404);
    expect(err.body).toBe("Not found");
    expect(err.message).toContain("404");
  });

  it("apiClient calls fetch with correct URL (API_BASE prefix)", async () => {
    mockFetch.mockResolvedValue(jsonResponse({ ok: true }));
    await apiClient("/api/v1/test");
    expect(mockFetch).toHaveBeenCalledTimes(1);
    const [url] = mockFetch.mock.calls[0];
    expect(url).toBe("http://localhost:8080/api/v1/test");
  });

  it("apiClient includes credentials: 'include'", async () => {
    mockFetch.mockResolvedValue(jsonResponse({}));
    await apiClient("/path");
    const [, init] = mockFetch.mock.calls[0];
    expect(init.credentials).toBe("include");
  });

  it("apiClient includes Content-Type: application/json header", async () => {
    mockFetch.mockResolvedValue(jsonResponse({}));
    await apiClient("/path");
    const [, init] = mockFetch.mock.calls[0];
    expect(init.headers["Content-Type"]).toBe("application/json");
  });

  it("apiClient throws ApiError when res.ok is false", async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 422,
      text: () => Promise.resolve("Validation failed"),
    });
    await expect(apiClient("/fail")).rejects.toThrow(ApiError);
    try {
      await apiClient("/fail");
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      expect((e as ApiError).status).toBe(422);
      expect((e as ApiError).body).toBe("Validation failed");
    }
  });

  it("apiGet calls with GET method", async () => {
    mockFetch.mockResolvedValue(jsonResponse({ data: 1 }));
    const result = await apiGet("/items");
    expect(mockFetch).toHaveBeenCalledTimes(1);
    const [, init] = mockFetch.mock.calls[0];
    expect(init.method).toBe("GET");
    expect(result).toEqual({ data: 1 });
  });

  it("apiPost calls with POST method and stringified body", async () => {
    mockFetch.mockResolvedValue(jsonResponse({ id: 42 }));
    const body = { name: "test", value: 123 };
    const result = await apiPost("/items", body);
    const [, init] = mockFetch.mock.calls[0];
    expect(init.method).toBe("POST");
    expect(init.body).toBe(JSON.stringify(body));
    expect(result).toEqual({ id: 42 });
  });

  it("apiPut calls with PUT method and stringified body", async () => {
    mockFetch.mockResolvedValue(jsonResponse({ updated: true }));
    const body = { name: "updated" };
    const result = await apiPut("/items/1", body);
    const [, init] = mockFetch.mock.calls[0];
    expect(init.method).toBe("PUT");
    expect(init.body).toBe(JSON.stringify(body));
    expect(result).toEqual({ updated: true });
  });

  it("apiDelete calls with DELETE method", async () => {
    mockFetch.mockResolvedValue(jsonResponse({ deleted: true }));
    const result = await apiDelete("/items/1");
    const [, init] = mockFetch.mock.calls[0];
    expect(init.method).toBe("DELETE");
    expect(result).toEqual({ deleted: true });
  });
});
