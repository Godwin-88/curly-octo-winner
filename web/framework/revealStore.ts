import type { Reveal } from './types';

/**
 * A one-time secret on its way to the record it belongs to. Creating a record
 * moves to that record's address, which can remount the section and lose its
 * state; the secret waits here, in memory only, until the record's view shows
 * it. It is never written to storage and is dropped once dismissed.
 */
let pending: { path: string; reveal: Reveal } | undefined;

export function holdReveal(path: string, reveal: Reveal | undefined): void {
  pending = reveal ? { path, reveal } : undefined;
}

export function heldReveal(path: string): Reveal | undefined {
  return pending?.path === path ? pending.reveal : undefined;
}
