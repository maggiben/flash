import { describe, expect, it } from 'vitest';
import { formatCountdown, isExpired, remainingSeconds } from '../utils/reservationTimer';

describe('reservationTimer', () => {
  const now = new Date('2026-05-29T12:00:00.000Z').getTime();

  it('computes remaining seconds from expires_at', () => {
    const expires = '2026-05-29T12:01:30.000Z';
    expect(remainingSeconds(expires, now)).toBe(90);
  });

  it('never returns negative remaining seconds', () => {
    const expires = '2026-05-29T11:59:00.000Z';
    expect(remainingSeconds(expires, now)).toBe(0);
  });

  it('detects expired reservations', () => {
    expect(isExpired('2026-05-29T11:59:59.000Z', now)).toBe(true);
    expect(isExpired('2026-05-29T12:00:01.000Z', now)).toBe(false);
  });

  it('formats countdown as m:ss', () => {
    expect(formatCountdown(90)).toBe('1:30');
    expect(formatCountdown(5)).toBe('0:05');
  });
});
