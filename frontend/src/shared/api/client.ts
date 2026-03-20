const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export async function apiClient<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    credentials: "include",
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  if (!res.ok) {
    let errorMessage = `API error: ${res.status}`;
    try {
      const body = await res.json();
      if (body.error) errorMessage = body.error;
    } catch {
      // Response body was not valid JSON
    }
    throw new Error(errorMessage);
  }
  try {
    return (await res.json()) as T;
  } catch {
    throw new Error(`Invalid response from server for ${path}`);
  }
}
