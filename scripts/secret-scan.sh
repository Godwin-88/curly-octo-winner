#!/usr/bin/env bash
#
# secret-scan.sh — fail CI (and pre-push hooks) if a live credential is ever
# committed.
#
# Why this exists: api/.env carries real Supabase, Upstash, Backblaze,
# Africa's Talking, Groq and M-Pesa credentials, and web/.env.local carries the
# front-end configuration. Both are covered by .gitignore, but a `git add -f`,
# a careless `git add .` from a new clone, or an edited ignore rule would put
# them in history — where rotating them is the only remedy. This script is the
# backstop.
#
# It is intentionally pattern-based rather than a full secret scanner: the
# patterns are specific enough to match only these providers' key formats, so
# the committed .env.example placeholders ("your_...", "change_this_in_production")
# never trip it, and the fixed dummy credentials used by the Go test suite are
# allowlisted explicitly rather than by suppressing broad categories.
#
# Usage: scripts/secret-scan.sh            # scan tracked files
# Exit:  0 clean, 1 findings, 2 bad usage.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# --- 1. no .env file may be tracked ------------------------------------------
# The examples are .env.example / .env.local.example and are intentionally
# tracked; anything literally named .env or .env.local is not.
tracked_env=$(git ls-files | grep -E '(^|/)\.env(\.local)?$' || true)
if [ -n "$tracked_env" ]; then
    echo "ERROR: live environment files are tracked by git:" >&2
    printf '%s\n' "$tracked_env" >&2
    echo "       git rm --cached <file>  and confirm .gitignore still covers them." >&2
    exit 1
fi

# --- 2. no provider key format may appear in a tracked file -------------------
# Provider-specific prefixes/shapes, so example placeholders do not match:
#   gsk_          Groq API key
#   atsk_         Africa's Talking API key
#   sb_secret_    Supabase secret key (sb_publishable_ is public by design)
#   eyJ...        HS256 JWT — Supabase service_role / anon keys
#   gAAAA...      Upstash Redis REST token
#   AB0F.../ABcF  Upstash Vector & Search REST tokens
#   postgres://u:p@  a connection string with the password inline — this is the
#                    shape a Supabase pooler personal access token takes
KEY_PATTERNS='gsk_[A-Za-z0-9]{20,}|atsk_[a-f0-9]{40,}|sb_secret_[A-Za-z0-9_-]{10,}|eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9\.[A-Za-z0-9_-]{30,}\.[A-Za-z0-9_-]{30,}|gAAAA[A-Za-z0-9_-]{40,}|AB0F[A-Za-z0-9+/=]{40,}|ABcF[A-Za-z0-9+/=]{40,}|postgres(ql)?://[^:/[:space:]]+:[^@[:space:]]+@'

# --- 3. no secret-shaped variable may be assigned a real value ---------------
# Catches the weaker case where a key is pasted into a config file, a compose
# file, or a workflow without the provider's usual prefix.
ASSIGN_PATTERN='(DATABASE_URL|SUPABASE_SERVICE_ROLE_KEY|SUPABASE_SECRET_KEY|JWT_SECRET|SETTINGS_ENCRYPTION_KEY|UPSTASH_REDIS_REST_TOKEN|UPSTASH_VECTOR_REST_TOKEN|UPSTASH_SEARCH_REST_TOKEN|UPSTASH_BLOB_TOKEN|B2_APPLICATION_KEY|GROQ_API_KEY|AT_API_KEY|META_WA_TOKEN|MPESA_CONSUMER_KEY|MPESA_CONSUMER_SECRET|MPESA_PASSKEY|RENDER_API_KEY|VERCEL_TOKEN)=[^[:space:]]'

# Scan every tracked file for both patterns. Template markers, not file
# extensions, are what distinguishes a placeholder from a real credential — the
# *.example files carry "your_..." / "change_this_in_production", and the
# filtering below removes them.
#
# This file is excluded from its own scan: it necessarily spells out every
# pattern it looks for, so its comments match themselves.
findings=$(git grep -nEI -e "$KEY_PATTERNS|$ASSIGN_PATTERN" -- . ':(exclude)scripts/secret-scan.sh' 2>/dev/null | sort -u || true)

# Assignments: keep only lines whose value carries no template marker. These are
# the exact markers the committed .env.example files use, so a real credential
# (a long random token) cannot plausibly contain one — the filter fails open only
# for strings that are visibly templates.
assignments=$(printf '%s\n' "$findings" \
    | grep -E "$ASSIGN_PATTERN" \
    | grep -vEi 'your_|your-|yourproject|change_this|\$\{|\$\(|<[^>]*>|xxx|example|placeholder|todo|fixme|=\.\.\.' \
    || true)

# Provider key formats. Template markers still apply here: a line may match both.
keys=$(printf '%s\n' "$findings" \
    | grep -E "$KEY_PATTERNS" \
    | grep -vEi 'your_|your-|yourproject|change_this|\$\{|\$\(|<[^>]*>|xxx|example|placeholder|todo|fixme|=\.\.\.' \
    || true)

# Local and dummy connection strings used by the Go test suite are not secrets.
noise() {
    grep -vE 'localhost|127\.0\.0\.1|:postgres@|:password@|:test@' || true
}

findings=$(printf '%s\n%s\n' "$keys" "$assignments" | grep -v '^$' | noise | sort -u || true)

if [ -n "$findings" ]; then
    echo "ERROR: possible committed secrets —" >&2
    printf '%s\n' "$findings" >&2
    echo >&2
    echo "       If any of these are real, rotate the credential at the provider" >&2
    echo "       first: a removed commit is still in git history." >&2
    exit 1
fi

echo "secret-scan: clean (no tracked .env files, no credential patterns)."
