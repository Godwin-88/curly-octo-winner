/**
 * Phone helpers for the contact book.
 *
 * This mirrors the API's NormalizePhone (api/internal/comms/contacts/phone.go)
 * so the form can show "will be saved as +254 712 345 678" while the user
 * types, without a round trip. The server stays the authority: this is purely
 * for immediate feedback, and the API re-validates every write.
 */

/** Normalizes a Kenyan number to E.164, or returns null when it is not valid. */
export function normalizePhone(raw: string): string | null {
  const cleaned = raw.replace(/[\s\-()./]/g, '');
  if (!cleaned) return null;

  let digits = cleaned.startsWith('+') ? cleaned.slice(1) : cleaned;

  if (digits.startsWith('254')) {
    digits = digits.slice(3);
  } else if (digits.startsWith('0')) {
    digits = digits.slice(1);
  }

  if (digits.length !== 9) return null;
  if (!/^\d+$/.test(digits)) return null;
  if (digits[0] !== '7') return null;

  return `+254${digits}`;
}

/** True when the number can be saved as-is. */
export function isValidPhone(raw: string): boolean {
  return normalizePhone(raw) !== null;
}

/** Formats a stored E.164 number for reading: +254 712 345 678. */
export function formatPhone(e164: string): string {
  const match = /^\+254(\d{3})(\d{3})(\d{3})$/.exec(e164);
  return match ? `+254 ${match[1]} ${match[2]} ${match[3]}` : e164;
}