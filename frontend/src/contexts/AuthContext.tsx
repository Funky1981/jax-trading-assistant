/**
 * AuthContext — JWT authentication with graceful fallback to anonymous mode.
 *
 * How it works:
 *  - On mount, fetches GET /auth/status to ask the backend if auth is required.
 *  - If auth is DISABLED (dev mode / JWT_SECRET not set):
 *      → User is automatically treated as authenticated (anonymous). No login page
 *        ever appears.
 *  - If auth is ENABLED (JWT_SECRET set in the environment):
 *      → Checks localStorage for an existing token. If present and not expired,
 *        user is authenticated. If missing or expired, they are shown the login
 *        page (via ProtectedRoute).
 *
 * To enable auth:  set JWT_SECRET in your environment before starting the backend.
 * To disable auth: leave JWT_SECRET unset (the docker-compose default).
 */

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';

const TOKEN_KEY = 'jax_token';

export interface AuthUser {
  username: string;
  role: string;
  anonymous: boolean; // true when backend has auth disabled
}

interface AuthContextValue {
  user: AuthUser | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  authRequired: boolean; // false → backend has auth disabled
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

// ── Token helpers ─────────────────────────────────────────────────────────────

function readToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

function storeToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token);
}

function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

function isTokenExpired(token: string): boolean {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    return payload.exp * 1000 < Date.now();
  } catch {
    return true;
  }
}

function decodeUser(token: string): AuthUser | null {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    return {
      username: payload.username ?? 'unknown',
      role: payload.role ?? 'user',
      anonymous: false,
    };
  } catch {
    return null;
  }
}

// ── Exported token accessor (used by http-client) ─────────────────────────────

export function getStoredToken(): string | null {
  const token = readToken();
  if (!token || isTokenExpired(token)) return null;
  return token;
}

// ── Provider ──────────────────────────────────────────────────────────────────

export function AuthProvider({ children }: { children: ReactNode }) {
  const [authRequired, setAuthRequired] = useState(false);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // On mount: ask backend if auth is enabled, then restore session from localStorage
  useEffect(() => {
    async function init() {
      try {
        const res = await fetch('/auth/status');
        if (res.ok) {
          const { enabled } = (await res.json()) as { enabled: boolean };
          setAuthRequired(enabled);

          if (!enabled) {
            // Auth is disabled — treat everyone as anonymous
            setUser({ username: 'anonymous', role: 'user', anonymous: true });
            setIsLoading(false);
            return;
          }

          // Auth IS required — restore session from localStorage
          const token = readToken();
          if (token && !isTokenExpired(token)) {
            setUser(decodeUser(token));
          }
          setIsLoading(false);
          return;
        }
      } catch {
        // Network error (backend unreachable)
      }

      // /auth/status returned non-ok (404, 502, etc.) or network failed
      // → treat as auth disabled (dev environment with backend not yet started)
      setUser({ username: 'anonymous', role: 'user', anonymous: true });
      setIsLoading(false);
    }
    void init();
  }, []);

  const login = useCallback(async (username: string, password: string) => {
    const res = await fetch('/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    });

    if (!res.ok) {
      const body = (await res.json().catch(() => ({}))) as { message?: string };
      throw new Error(body.message ?? 'Invalid credentials');
    }

    const { access_token } = (await res.json()) as { access_token: string };
    storeToken(access_token);
    setUser(decodeUser(access_token));
  }, []);

  const logout = useCallback(() => {
    clearToken();
    setUser(null);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isLoading,
      isAuthenticated: user !== null,
      authRequired,
      login,
      logout,
    }),
    [user, isLoading, authRequired, login, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// ── Hooks ────────────────────────────────────────────────────────────────────

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>');
  return ctx;
}
