/**
 * Pure timer helpers for reservation TTL display (server expires_at is source of truth).
 */
export function remainingSeconds(expiresAtIso: string, nowMs: number = Date.now()): number {
  const expiresMs = new Date(expiresAtIso).getTime();
  const diff = Math.ceil((expiresMs - nowMs) / 1000);
  return Math.max(0, diff);
}

export function isExpired(expiresAtIso: string, nowMs: number = Date.now()): boolean {
  return remainingSeconds(expiresAtIso, nowMs) === 0;
}

export function formatCountdown(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${s.toString().padStart(2, '0')}`;
}
