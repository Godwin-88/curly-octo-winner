import { NextRequest, NextResponse } from 'next/server';

// Edge route protection for Shule360 (Next.js 16 proxy convention).
//
// This is a cookie-presence fast-path only: the Go API re-validates every JWT
// signature on each request, so a forged or expired cookie just leads to API
// 401s that the frontend handles by bouncing to sign-in. The goal here is that
// anonymous visitors never see the admin/teacher/parent UI shells at all.
//
// The session cookies (shule360_session / shule360_guardian_session) are
// HttpOnly cookies issued by the Go API and proxied same-origin via
// next.config.js rewrites — the edge runtime can read request cookies, even
// though browser JavaScript cannot.

const STAFF_COOKIE = 'shule360_session';
const GUARDIAN_COOKIE = 'shule360_guardian_session';

// Route prefixes served by the (admin) route group + teacher workspace.
// Both require a staff JWT.
const STAFF_PREFIXES = [
  '/dashboard',
  '/school',
  '/communications',
  '/academic',
  '/learners',
  '/vehicles',
  '/routes',
  '/trips',
  '/finance',
  '/reports',
  '/analytics',
  '/hr',
  '/procurement',
  '/intelligence',
  '/security',
  '/settings',
  '/teacher',
  // The list → view → edit workspace: /w/{group}/{school}/{module}/...
  '/w',
];

export default function proxy(req: NextRequest) {
  const { pathname, search } = req.nextUrl;
  const staffToken = req.cookies.get(STAFF_COOKIE)?.value;
  const guardianToken = req.cookies.get(GUARDIAN_COOKIE)?.value;

  // --- Parent portal (guardian session) ---
  // /parent/dashboard was removed; the dashboard now lives at /parent.
  if (pathname === '/parent/dashboard') {
    return NextResponse.redirect(new URL('/parent', req.url));
  }

  if (pathname === '/parent' || pathname.startsWith('/parent/')) {
    if (pathname === '/parent/login') {
      if (guardianToken) {
        return NextResponse.redirect(new URL('/parent', req.url));
      }
      return NextResponse.next();
    }
    if (!guardianToken) {
      const login = new URL('/parent/login', req.url);
      if (pathname !== '/parent') {
        login.searchParams.set('next', pathname + search);
      }
      return NextResponse.redirect(login);
    }
    return NextResponse.next();
  }

  // --- Public: landing + staff sign-in ---
  if (pathname === '/' || pathname.startsWith('/auth')) {
    if (pathname === '/auth/login' && staffToken) {
      return NextResponse.redirect(new URL('/dashboard', req.url));
    }
    return NextResponse.next();
  }

  // --- Staff area (admin + teacher) ---
  const isStaffArea = STAFF_PREFIXES.some(
    (p) => pathname === p || pathname.startsWith(p + '/')
  );
  if (isStaffArea && !staffToken) {
    const login = new URL('/auth/login', req.url);
    login.searchParams.set('next', pathname + search);
    return NextResponse.redirect(login);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon\\.svg|robots\\.txt).*)'],
};
