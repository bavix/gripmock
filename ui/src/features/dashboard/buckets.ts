import type { CallRecord } from '../../lib/types';

export interface Bucket {
  /** Bucket start, ms since epoch. */
  t: number;
  /** Human label (HH:MM). */
  label: string;
  ok: number;
  errors: number;
}

const isOk = (c: CallRecord) => !c.code || c.code === 0;

export interface LatencyStats {
  count: number; // records that carried an elapsedMs
  avg: number;
  p95: number;
  max: number;
}

// Latency summary over records that reported elapsedMs (older records / sub-ms
// calls omit it). Returns count 0 when nothing is measurable.
export function latencyStats(records: CallRecord[]): LatencyStats {
  const ms = records
    .map((r) => r.elapsedMs)
    .filter((v): v is number => typeof v === 'number' && v >= 0)
    .sort((a, b) => a - b);

  if (ms.length === 0) return { count: 0, avg: 0, p95: 0, max: 0 };

  const sum = ms.reduce((n, v) => n + v, 0);
  // Nearest-rank p95 (clamped to the last index).
  const idx = Math.min(ms.length - 1, Math.ceil(0.95 * ms.length) - 1);

  return {
    count: ms.length,
    avg: Math.round(sum / ms.length),
    p95: ms[Math.max(0, idx)],
    max: ms[ms.length - 1],
  };
}

/**
 * Bucket call records into fixed time windows for the calls/errors trend.
 * Returns a contiguous series from the oldest record (or now-minus-span) to now,
 * so quiet minutes render as zeros instead of gaps.
 */
export function bucketCalls(records: CallRecord[], bucketMs = 60_000, now = Date.now(), maxBuckets = 30): Bucket[] {
  const counts = new Map<number, { ok: number; errors: number }>();
  let oldest = now;
  for (const r of records) {
    const ts = new Date(r.timestamp).getTime();
    if (Number.isNaN(ts)) continue;
    if (ts < oldest) oldest = ts;
    const key = Math.floor(ts / bucketMs) * bucketMs;
    const b = counts.get(key) ?? { ok: 0, errors: 0 };
    if (isOk(r)) b.ok++; else b.errors++;
    counts.set(key, b);
  }

  const end = Math.floor(now / bucketMs) * bucketMs;
  const span = Math.min(maxBuckets - 1, Math.max(0, Math.ceil((end - Math.floor(oldest / bucketMs) * bucketMs) / bucketMs)));
  const out: Bucket[] = [];
  for (let i = span; i >= 0; i--) {
    const t = end - i * bucketMs;
    const b = counts.get(t) ?? { ok: 0, errors: 0 };
    const d = new Date(t);
    out.push({ t, label: `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`, ok: b.ok, errors: b.errors });
  }
  return out;
}
