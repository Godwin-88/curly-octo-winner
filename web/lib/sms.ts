/**
 * How an SMS is billed. Mirrors api/internal/comms/sms/segments.go so the
 * composer can show the unit count while the user types; the API recomputes it
 * on every send and is the authority.
 *
 * GSM-7 text: 160 characters in one unit, 153 per unit once split.
 * Anything else (a curly quote pasted from Word, an emoji) makes the whole
 * message Unicode: 70 per unit, 67 once split.
 */

export const MAX_SMS_UNITS = 3;

const GSM7_BASIC =
  '@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !"#¤%&\'()*+,-./0123456789:;<=>?' +
  '¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà';
const GSM7_EXTENDED = '\f^{}\\[~]|€';

export interface Segments {
  units: number;
  encoding: 'GSM-7' | 'Unicode';
  length: number;
  perUnit: number;
}

export function countSegments(message: string): Segments {
  if (message === '') return { units: 0, encoding: 'GSM-7', length: 0, perUnit: 160 };

  let septets = 0;
  let gsm = true;
  for (const char of message) {
    if (GSM7_BASIC.includes(char)) septets += 1;
    else if (GSM7_EXTENDED.includes(char)) septets += 2;
    else {
      gsm = false;
      break;
    }
  }
  if (gsm) {
    return septets <= 160
      ? { units: 1, encoding: 'GSM-7', length: septets, perUnit: 160 }
      : { units: Math.ceil(septets / 153), encoding: 'GSM-7', length: septets, perUnit: 153 };
  }

  // String.length counts UTF-16 code units, which is what UCS-2 bills.
  const length = message.length;
  return length <= 70
    ? { units: 1, encoding: 'Unicode', length, perUnit: 70 }
    : { units: Math.ceil(length / 67), encoding: 'Unicode', length, perUnit: 67 };
}

/** "86 characters · 1 SMS unit", with a warning once the limit is passed. */
export function describeSegments(message: string): string {
  const seg = countSegments(message);
  if (seg.units === 0) return '0 characters';
  const unicode = seg.encoding === 'Unicode' ? ' (contains special characters, so fewer fit per unit)' : '';
  const over = seg.units > MAX_SMS_UNITS ? ` — over the limit of ${MAX_SMS_UNITS} units, shorten it` : '';
  const personalised = /\{\{\s*[a-zA-Z_]+\s*\}\}/.test(message) ? ' before names are filled in' : '';
  return `${seg.length} characters · ${seg.units} SMS unit${seg.units === 1 ? '' : 's'}${personalised}${unicode}${over}`;
}
