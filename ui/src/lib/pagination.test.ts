import { describe, it, expect } from 'vitest';
import { nextPageOffset } from './pagination';

describe('nextPageOffset', () => {
  it('returns the loaded count while more remain', () => {
    expect(nextPageOffset(60, 60, 500)).toBe(60);
    expect(nextPageOffset(60, 120, 500)).toBe(120);
  });

  it('stops exactly at the total (boundary)', () => {
    expect(nextPageOffset(60, 500, 500)).toBeUndefined();
  });

  it('stops when loaded overshoots the total', () => {
    expect(nextPageOffset(60, 540, 500)).toBeUndefined();
  });

  it('stops on an empty page even if the total says more', () => {
    // Guards the offset loop when total shrank mid-scroll (concurrent deletes).
    expect(nextPageOffset(0, 100, 500)).toBeUndefined();
  });

  it('handles a zero total', () => {
    expect(nextPageOffset(0, 0, 0)).toBeUndefined();
  });

  it('continues past a later page whose total is understated (X-Total-Count missing)', () => {
    // firstPageTotal is authoritative (500); a stale per-page total would stop early.
    expect(nextPageOffset(60, 120, 500)).toBe(120);
  });
});
