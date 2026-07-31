import { RESEARCH_DIAGNOSTICS_BASE_PATH } from '@/config/api';

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
  const fallbackOrigin = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8081';
  const url = new URL(baseUrl || fallbackOrigin, fallbackOrigin);
  const endpoint = new URL(path, fallbackOrigin);
  const basePath = url.pathname === '/' ? '' : url.pathname.replace(/\/$/, '');
  url.pathname = `${basePath}${endpoint.pathname}`;
  url.search = endpoint.search;
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      url.searchParams.set(key, value);
    }
  }
  return url.toString();
}

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
    return await fetch(url, { ...init, signal: controller.signal });
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
      headers.Authorization = `Bearer ${token}`;
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
    put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
    patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, body),
    delete: <T>(path: string, body?: unknown) => request<T>('DELETE', path, body),
    buildUrl: (path: string, params?: Record<string, string>) => buildUrl(baseUrl, path, params),
  };
}

function devProxyBaseUrl(envUrl: string | undefined, productionFallback: string) {
  return import.meta.env.DEV ? '' : envUrl || productionFallback;
}

function sameOriginBaseUrl(path: string) {
  const origin = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8081';
  return new URL(path, origin).toString();
}

export const apiClient = createHttpClient({
  baseUrl: devProxyBaseUrl(import.meta.env.VITE_JAX_API_URL, 'http://localhost:8081'),
  timeoutMs: 30_000,
});

export const memoryClient = createHttpClient({
  baseUrl: sameOriginBaseUrl(RESEARCH_DIAGNOSTICS_BASE_PATH),
  timeoutMs: 15_000,
});
