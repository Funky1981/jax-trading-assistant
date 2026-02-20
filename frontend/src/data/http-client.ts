export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

export interface HttpClientOptions {
  baseUrl?: string;
  timeoutMs?: number;
}

export class HttpError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly body: unknown
  ) {
    super(message);
    this.name = 'HttpError';
  }
}

export function buildUrl(baseUrl: string, path: string, params?: Record<string, string>) {
  const url = new URL(path, baseUrl);
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      url.searchParams.set(key, value);
    }
  }
  return url.toString();
}

// ── Auth token injection ───────────────────────────────────────────────────────
// Reads directly from localStorage using the same key as AuthContext so that
// there is no import cycle (http-client does not import AuthContext).

const TOKEN_KEY = 'jax_token';

function getAuthToken(): string | null {
  try {
    const token = localStorage.getItem(TOKEN_KEY);
    if (!token) return null;
    const payload = JSON.parse(atob(token.split('.')[1])) as { exp?: number };
    if (payload.exp && payload.exp * 1000 < Date.now()) {
      localStorage.removeItem(TOKEN_KEY);
      return null;
    }
    return token;
  } catch {
    return null;
  }
}

async function fetchWithTimeout(url: string, init: RequestInit, timeoutMs: number) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(url, { ...init, signal: controller.signal });
    return response;
  } finally {
    clearTimeout(timeout);
  }
}

export function createHttpClient(options: HttpClientOptions = {}) {
  const baseUrl = options.baseUrl ?? 'http://localhost:8081';
  const timeoutMs = options.timeoutMs ?? 10_000;

  async function request<T>(method: HttpMethod, path: string, body?: unknown) {
    const url = buildUrl(baseUrl, path);
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };

    const token = getAuthToken();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const init: RequestInit = {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    };

    const response = await fetchWithTimeout(url, init, timeoutMs);
    const contentType = response.headers.get('content-type') ?? '';
    const payload = contentType.includes('application/json')
      ? await response.json()
      : await response.text();

    if (!response.ok) {
      throw new HttpError(`Request failed: ${response.status}`, response.status, payload);
    }

    return payload as T;
  }

  return {
    get: <T>(path: string) => request<T>('GET', path),
    post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
    buildUrl: (path: string, params?: Record<string, string>) => buildUrl(baseUrl, path, params),
  };
}

// Backend service clients
export const apiClient = createHttpClient({
  baseUrl: import.meta.env.VITE_API_URL || 'http://localhost:8081',
  timeoutMs: 30_000,
});

export const memoryClient = createHttpClient({
  baseUrl: import.meta.env.VITE_MEMORY_API_URL || 'http://localhost:8091',
  timeoutMs: 15_000,
});
