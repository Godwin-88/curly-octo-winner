'use client';

// Central auth context for Shule360.
//
// Owns both session scopes:
//   - staff    (admin dashboard + teacher portal)  -> `token`, `staff`
//   - guardian (parent portal)                     -> `guardianToken`, `guardian`
//
// Sessions are hydrated once on mount from localStorage (written at login) and
// mirrored into cookies (`shule360_token`, `shule360_guardian_token`) so the
// edge proxy can route-protect areas without shipping tokens to it in headers.
// The Go API always re-validates the JWT signature on every request; the proxy
// check is a UX fast-path, not the security boundary.

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
  /** Staff JWT (admin + teacher portals). Empty until hydrated or when signed out. */
  token: string;
  staff: StaffUser | null;
  /** Guardian JWT (parent portal). Empty until hydrated or when signed out. */
  guardianToken: string;
  guardian: GuardianUser | null;
  /** True once the initial localStorage hydration has run. */
  ready: boolean;
  loginStaff: (email: string, password: string) => Promise<StaffUser>;
  logoutStaff: (redirectTo?: string) => void;
  loginGuardian: (phone: string, pin: string, tenantId: string) => Promise<GuardianUser>;
  logoutGuardian: () => void;
}

// --- Storage keys (kept identical to the keys written by earlier versions) ---

const STAFF_TOKEN_KEY = 'token';
const STAFF_USER_KEY = 'staff';
const GUARDIAN_TOKEN_KEY = 'guardian_token';
const GUARDIAN_USER_KEY = 'guardian';

// Cookies read by web/proxy.ts (edge route protection).
const STAFF_COOKIE = 'shule360_token';
const GUARDIAN_COOKIE = 'shule360_guardian_token';

// Matches the 24h expiry the Go API issues on both JWT types.
const SESSION_MAX_AGE_SECONDS = 24 * 60 * 60;

function writeSessionCookie(name: string, value: string) {
  if (typeof document === 'undefined') return;
  const secure = typeof window !== 'undefined' && window.location.protocol === 'https:' ? '; Secure' : '';
  document.cookie = `${name}=${encodeURIComponent(value)}; path=/; max-age=${SESSION_MAX_AGE_SECONDS}; SameSite=Lax${secure}`;
}

function clearSessionCookie(name: string) {
  if (typeof document === 'undefined') return;
  document.cookie = `${name}=; path=/; max-age=0; SameSite=Lax`;
}

function safeParse<T>(raw: string | null): T | null {
  if (!raw) return null;
  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
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

  // Hydrate on mount (client only). Running this in an effect instead of a
  // lazy useState initializer keeps the first client render identical to the
  // SSR output (token = '') and avoids hydration mismatches.
  useEffect(() => {
    const t = window.localStorage.getItem(STAFF_TOKEN_KEY) || '';
    const gt = window.localStorage.getItem(GUARDIAN_TOKEN_KEY) || '';
    setToken(t);
    setStaff(safeParse<StaffUser>(window.localStorage.getItem(STAFF_USER_KEY)));
    setGuardianToken(gt);
    setGuardian(safeParse<GuardianUser>(window.localStorage.getItem(GUARDIAN_USER_KEY)));
    // Re-sync cookies in case they expired while localStorage still holds the
    // session, so edge route protection keeps working on the next navigation.
    if (t) writeSessionCookie(STAFF_COOKIE, t);
    if (gt) writeSessionCookie(GUARDIAN_COOKIE, gt);
    setReady(true);
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
      window.localStorage.setItem(STAFF_TOKEN_KEY, data.token);
      window.localStorage.setItem(STAFF_USER_KEY, JSON.stringify(user));
      writeSessionCookie(STAFF_COOKIE, data.token);
      setToken(data.token);
      setStaff(user);
      return user;
    },
    []
  );

  const logoutStaff = useCallback(
    (redirectTo = '/auth/login') => {
      window.localStorage.removeItem(STAFF_TOKEN_KEY);
      window.localStorage.removeItem(STAFF_USER_KEY);
      clearSessionCookie(STAFF_COOKIE);
      setToken('');
      setStaff(null);
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
      window.localStorage.setItem(GUARDIAN_TOKEN_KEY, data.token);
      window.localStorage.setItem(GUARDIAN_USER_KEY, JSON.stringify(user));
      window.localStorage.setItem('tenant_id', tenantId);
      writeSessionCookie(GUARDIAN_COOKIE, data.token);
      setGuardianToken(data.token);
      setGuardian(user);
      return user;
    },
    []
  );

  const logoutGuardian = useCallback(() => {
    window.localStorage.removeItem(GUARDIAN_TOKEN_KEY);
    window.localStorage.removeItem(GUARDIAN_USER_KEY);
    clearSessionCookie(GUARDIAN_COOKIE);
    setGuardianToken('');
    setGuardian(null);
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
