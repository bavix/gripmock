import { describe, it, expect } from 'vitest';
import { bucketCalls } from './buckets';
import type { CallRecord } from '../../lib/types';

const rec = (msAgo: number, code?: number, now = 1_700_000_000_000): CallRecord => ({
  timestamp: new Date(now - msAgo).toISOString(),
  ...(code !== undefined ? { code } : {}),
} as CallRecord);

const NOW = 1_700_000_000_000;

describe('bucketCalls', () => {
  it('splits ok vs error counts per minute', () => {
    const out = bucketCalls([rec(10_000), rec(20_000, 5), rec(30_000)], 60_000, NOW);
    const last = out[out.length - 1];
    expect(last.ok + out.reduce((n, b) => n + b.ok, 0) - last.ok).toBeGreaterThanOrEqual(2);
    expect(out.reduce((n, b) => n + b.errors, 0)).toBe(1);
    expect(out.reduce((n, b) => n + b.ok, 0)).toBe(2);
  });

  it('fills quiet buckets with zeros (contiguous)', () => {
    const out = bucketCalls([rec(5 * 60_000)], 60_000, NOW);
    expect(out.length).toBe(6); // 5 minutes ago .. now inclusive
    expect(out.some((b) => b.ok === 0 && b.errors === 0)).toBe(true);
  });

  it('caps at maxBuckets', () => {
    const out = bucketCalls([rec(120 * 60_000)], 60_000, NOW, 30);
    expect(out.length).toBe(30);
  });

  it('empty input yields a single zero bucket for now', () => {
    const out = bucketCalls([], 60_000, NOW);
    expect(out.length).toBe(1);
    expect(out[0]).toMatchObject({ ok: 0, errors: 0 });
  });

  it('treats code 0 / missing code as ok', () => {
    const out = bucketCalls([rec(1000, 0), rec(2000)], 60_000, NOW);
    expect(out.reduce((n, b) => n + b.ok, 0)).toBe(2);
  });
});

import { latencyStats } from './buckets';

describe('latencyStats', () => {
  const rec = (elapsedMs?: number): CallRecord => ({ timestamp: new Date(1_700_000_000_000).toISOString(), ...(elapsedMs !== undefined ? { elapsedMs } : {}) } as CallRecord);

  it('empty / no-elapsed records → zeros', () => {
    expect(latencyStats([])).toEqual({ count: 0, avg: 0, p95: 0, max: 0 });
    expect(latencyStats([rec(), rec()])).toMatchObject({ count: 0 });
  });

  it('avg, p95 (nearest-rank), max over measured records', () => {
    const rs = [10, 20, 30, 40, 100].map(rec);
    const s = latencyStats(rs);
    expect(s.count).toBe(5);
    expect(s.avg).toBe(40); // (200)/5
    expect(s.max).toBe(100);
    expect(s.p95).toBe(100); // ceil(0.95*5)-1 = 4 → last
  });

  it('ignores records without elapsedMs', () => {
    const s = latencyStats([rec(50), rec(), rec(150)]);
    expect(s.count).toBe(2);
    expect(s.max).toBe(150);
  });
});
