'use client';

// Central auth context for Shule360.
//
// Owns both session scopes:
//   - staff    (admin dashboard + teacher portal)  -> `token`, `staff`
//   - guardian (parent portal)                     -> `guardianToken`, `guardian`
//
// Sessions are HttpOnly cookies issued by the Go API at login and carried
// same-origin on every request (next.config.js proxies /api/v1/* to the API).
// JavaScript never touches the tokens: on mount the provider asks the API
// "who am I?" (GET /auth/me, GET /auth/guardian/me) and hydrates the profile
// from the *verified* server response. Only the non-secret profile (name,
// email, role) is cached in localStorage for an instant first paint — the
// cookie is the source of truth.
//
// The Go API re-validates the JWT signature on every request; the edge proxy
// (web/proxy.ts) is a UX fast-path, not the security boundary.

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { useRouter } from 'next/navigation';
import { API_BASE } from '@/lib/api';

// --- Types ---

export interface StaffUser {
  id: string;
  tenant_id: string;
  full_name: string;
  email: string;
  role: string;
  phone?: string;
}

export interface GuardianUser {
  id: string;
  tenant_id: string;
  full_name: string;
  phone: string;
  email?: string;
}

interface AuthContextValue {
  /** Staff JWT (admin + teacher portals). Held in memory only, when available. */
  token: string;
  staff: StaffUser | null;
  /** Guardian JWT (parent portal). Held in memory only, when available. */
  guardianToken: string;
  guardian: GuardianUser | null;
  /** True once the initial who-am-i hydration has run. */
  ready: boolean;
  loginStaff: (email: string, password: string) => Promise<StaffUser>;
  logoutStaff: (redirectTo?: string) => void;
  loginGuardian: (phone: string, pin: string, tenantId: string) => Promise<GuardianUser>;
  logoutGuardian: () => void;
}

// --- Profile cache & legacy cleanup ---

const STAFF_PROFILE_KEY = 'staff';
const GUARDIAN_PROFILE_KEY = 'guardian';

// Cookies used by pre-cookie-auth builds (JS-readable JWT mirrors). They are
// cleared on every mount — the httpOnly shule360_session cookies replaced them.
const LEGACY_COOKIE_NAMES = ['shule360_token', 'shule360_guardian_token'];
const LEGACY_STORAGE_KEYS = ['token', 'guardian_token'];

function clearLegacySessions() {
  if (typeof window === 'undefined') return;
  for (const key of LEGACY_STORAGE_KEYS) {
    window.localStorage.removeItem(key);
  }
  for (const name of LEGACY_COOKIE_NAMES) {
    document.cookie = `${name}=; path=/; max-age=0; SameSite=Lax`;
  }
}

function cacheProfile<T>(key: string, value: T | null) {
  if (typeof window === 'undefined') return;
  if (value) {
    window.localStorage.setItem(key, JSON.stringify(value));
  } else {
    window.localStorage.removeItem(key);
  }
}

function readCachedProfile<T>(key: string): T | null {
  if (typeof window === 'undefined') return null;
  try {
    const raw = window.localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as T) : null;
  } catch {
    return null;
  }
}

/** Best-effort POST that swallows network/HTTP errors (used by logout). */
async function postAndIgnore(path: string) {
  try {
    await fetch(`${API_BASE}${path}`, { method: 'POST' });
  } catch {
    // Logging out must never fail because the API is unreachable.
  }
}

// --- Context ---

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState('');
  const [staff, setStaff] = useState<StaffUser | null>(null);
  const [guardianToken, setGuardianToken] = useState('');
  const [guardian, setGuardian] = useState<GuardianUser | null>(null);
  const [ready, setReady] = useState(false);
  const router = useRouter();

  // Who-am-i hydration on mount (client only). Sessions live in HttpOnly
  // cookies, so the browser asks the API who it is: each /me endpoint answers
  // 200 + profile for a valid cookie of its scope, or 401 when absent/expired.
  // The cached localStorage profile is painted immediately (optimistic) and
  // then reconciled with the verified server response.
  useEffect(() => {
    let alive = true;
    clearLegacySessions();

    setStaff(readCachedProfile<StaffUser>(STAFF_PROFILE_KEY));
    setGuardian(readCachedProfile<GuardianUser>(GUARDIAN_PROFILE_KEY));

    (async () => {
      const [staffRes, guardianRes] = await Promise.all([
        fetch(`${API_BASE}/auth/me`),
        fetch(`${API_BASE}/auth/guardian/me`),
      ]);

      if (!alive) return;

      if (staffRes.ok) {
        const data = (await staffRes.json().catch(() => null)) as { staff?: StaffUser } | null;
        if (data?.staff) {
          setStaff(data.staff);
          cacheProfile(STAFF_PROFILE_KEY, data.staff);
        }
      } else {
        setStaff(null);
        cacheProfile(STAFF_PROFILE_KEY, null);
      }

      if (guardianRes.ok) {
        const data = (await guardianRes.json().catch(() => null)) as { guardian?: GuardianUser } | null;
        if (data?.guardian) {
          setGuardian(data.guardian);
          cacheProfile(GUARDIAN_PROFILE_KEY, data.guardian);
        }
      } else {
        setGuardian(null);
        cacheProfile(GUARDIAN_PROFILE_KEY, null);
      }

      if (alive) setReady(true);
    })();

    return () => {
      alive = false;
    };
  }, []);

  const loginStaff = useCallback(
    async (email: string, password: string): Promise<StaffUser> => {
      const res = await fetch(`${API_BASE}/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(data.error || 'Login failed');
      }
      const user = data.staff as StaffUser;
      // The Go API set the HttpOnly session cookie on this response; the token
      // stays in memory only (never persisted to JS-accessible storage).
      setToken(data.token);
      setStaff(user);
      cacheProfile(STAFF_PROFILE_KEY, user);
      return user;
    },
    []
  );

  const logoutStaff = useCallback(
    (redirectTo = '/auth/login') => {
      // Server clears the HttpOnly cookie; best effort if the API is down.
      postAndIgnore('/auth/logout');
      setToken('');
      setStaff(null);
      cacheProfile(STAFF_PROFILE_KEY, null);
      router.push(redirectTo);
    },
    [router]
  );

  const loginGuardian = useCallback(
    async (phone: string, pin: string, tenantId: string): Promise<GuardianUser> => {
      const res = await fetch(`${API_BASE}/auth/guardian/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone, pin, tenant_id: tenantId }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(data.error || 'Login failed');
      }
      const user = data.guardian as GuardianUser;
      setGuardianToken(data.token);
      setGuardian(user);
      cacheProfile(GUARDIAN_PROFILE_KEY, user);
      // Remember the school so the picker can pre-select it next visit.
      window.localStorage.setItem('tenant_id', tenantId);
      return user;
    },
    []
  );

  const logoutGuardian = useCallback(() => {
    // Server clears the HttpOnly cookie AND revokes the guardian_sessions rows.
    postAndIgnore('/auth/guardian/logout');
    setGuardianToken('');
    setGuardian(null);
    cacheProfile(GUARDIAN_PROFILE_KEY, null);
    router.push('/parent/login');
  }, [router]);

  const value = useMemo(
    () => ({
      token,
      staff,
      guardianToken,
      guardian,
      ready,
      loginStaff,
      logoutStaff,
      loginGuardian,
      logoutGuardian,
    }),
    [token, staff, guardianToken, guardian, ready, loginStaff, logoutStaff, loginGuardian, logoutGuardian]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// --- Hooks & helpers ---

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within <AuthProvider>');
  }
  return ctx;
}

/**
 * Whether a nav item / page is accessible for the given role.
 * Items without a role restriction are visible to everyone; before hydration
 * (role undefined) everything is rendered so the layout guard can redirect
 * anonymous visitors instead of flashing an empty nav.
 */
export function roleCanAccess(role: string | undefined, allowedRoles?: string[]): boolean {
  if (!allowedRoles || allowedRoles.length === 0) return true;
  if (!role) return true;
  return allowedRoles.includes(role);
}
