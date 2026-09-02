import { describe, it, expect } from 'vitest';
import { computeUsage } from './stubUsage';
import type { CallRecord } from '../lib/types';

const call = (timestamp: string, stubId?: string): CallRecord => ({
  service: 'simple.Gripmock',
  method: 'SayHello',
  stubId,
  timestamp,
  code: 0,
});

const stub = { id: 'stub-1' };

describe('computeUsage', () => {
  it('returns null before the history is loaded or without a stub', () => {
    expect(computeUsage(undefined, stub)).toBeNull();
    expect(computeUsage([], undefined)).toBeNull();
  });

  it('returns null when the stub has never matched', () => {
    expect(computeUsage([call('2024-05-01T10:00:00Z', 'other')], stub)).toBeNull();
  });

  it('reports first and last from an oldest-first history', () => {
    const usage = computeUsage([
      call('2024-05-01T10:00:00Z', 'stub-1'),
      call('2024-05-01T11:00:00Z', 'stub-1'),
      call('2024-05-01T12:00:00Z', 'stub-1'),
    ], stub);

    expect(usage?.total).toBe(3);
    expect(usage?.first.toISOString()).toBe('2024-05-01T10:00:00.000Z');
    expect(usage?.last.toISOString()).toBe('2024-05-01T12:00:00.000Z');
  });

  it('reports the same first and last from a newest-first history', () => {
    const usage = computeUsage([
      call('2024-05-01T12:00:00Z', 'stub-1'),
      call('2024-05-01T11:00:00Z', 'stub-1'),
      call('2024-05-01T10:00:00Z', 'stub-1'),
    ], stub);

    expect(usage?.first.toISOString()).toBe('2024-05-01T10:00:00.000Z');
    expect(usage?.last.toISOString()).toBe('2024-05-01T12:00:00.000Z');
  });

  it('counts only this stub and ignores unmatched calls', () => {
    const usage = computeUsage([
      call('2024-05-01T09:00:00Z'),
      call('2024-05-01T10:00:00Z', 'stub-1'),
      call('2024-05-01T11:00:00Z', 'other'),
      call('2024-05-01T13:00:00Z', 'stub-1'),
    ], stub);

    expect(usage?.total).toBe(2);
    expect(usage?.first.toISOString()).toBe('2024-05-01T10:00:00.000Z');
    expect(usage?.last.toISOString()).toBe('2024-05-01T13:00:00.000Z');
  });

  it('leaves the caller-owned history array untouched', () => {
    const history = [
      call('2024-05-01T10:00:00Z', 'stub-1'),
      call('2024-05-01T12:00:00Z', 'stub-1'),
    ];
    const order = history.map((c) => c.timestamp);

    computeUsage(history, stub);

    expect(history.map((c) => c.timestamp)).toEqual(order);
  });

  it('handles a single call', () => {
    const usage = computeUsage([call('2024-05-01T10:00:00Z', 'stub-1')], stub);

    expect(usage?.total).toBe(1);
    expect(usage?.first.toISOString()).toBe(usage?.last.toISOString());
  });
});
