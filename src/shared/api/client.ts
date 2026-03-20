const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export class ApiError extends Error {
  constructor(public status: number, public body: string) {
    super(`API Error ${status}: ${body}`);
    this.name = 'ApiError';
  }
}

export async function apiClient<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      ...init,
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...init?.headers,
      },
    });
  } catch {
    throw new ApiError(0, 'Network error: unable to reach server');
  }
  if (!res.ok) {
    const body = await res.text().catch(() => 'unknown error');
    throw new ApiError(res.status, body);
  }
  return res.json().catch(() => {
    throw new ApiError(res.status, 'Invalid JSON response');
  });
}

export function apiGet<T>(path: string): Promise<T> {
  return apiClient<T>(path, { method: 'GET' });
}

export function apiPost<T>(path: string, body: unknown): Promise<T> {
  return apiClient<T>(path, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function apiPut<T>(path: string, body: unknown): Promise<T> {
  return apiClient<T>(path, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
}

export function apiDelete<T>(path: string): Promise<T> {
  return apiClient<T>(path, { method: 'DELETE' });
}
