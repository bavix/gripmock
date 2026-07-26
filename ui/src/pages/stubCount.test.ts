import { describe, it, expect } from 'vitest';
import { stubCountLabel } from './stubCount';

describe('stubCountLabel', () => {
  it('shows "X of Y" when a partial page is loaded (no client filter)', () => {
    expect(stubCountLabel(60, 500, false)).toBe('(60 of 500)');
  });

  it('shows just the total when everything is loaded', () => {
    expect(stubCountLabel(500, 500, false)).toBe('(500)');
  });

  it('reports honest "N matching" under a client-only filter (regression)', () => {
    // Was misleadingly "(60 of 500)" — 60 matches among loaded, not of 500.
    expect(stubCountLabel(60, 500, true)).toBe('(60 matching)');
    expect(stubCountLabel(0, 500, true)).toBe('(0 matching)');
  });
});
