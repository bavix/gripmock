import { describe, it, expect, vi, afterEach } from 'vitest';
import { randomHex, newSessionId } from './ids';

afterEach(() => {
  vi.restoreAllMocks();
});

describe('randomHex', () => {
  it('returns the requested number of hex characters', () => {
    expect(randomHex()).toMatch(/^[0-9a-f]{8}$/);
    expect(randomHex(4)).toMatch(/^[0-9a-f]{4}$/);
    expect(randomHex(7)).toMatch(/^[0-9a-f]{7}$/);
  });

  it('does not repeat itself', () => {
    const ids = new Set(Array.from({ length: 200 }, () => randomHex()));
    expect(ids.size).toBe(200);
  });

  it('never touches crypto.randomUUID', () => {
    const randomUUID = vi.fn(() => { throw new TypeError('not a secure context'); });
    vi.spyOn(globalThis, 'crypto', 'get').mockReturnValue({
      ...globalThis.crypto,
      randomUUID,
    } as Crypto);

    expect(randomHex()).toMatch(/^[0-9a-f]{8}$/);
    expect(randomUUID).not.toHaveBeenCalled();
  });

  it('still produces an id where getRandomValues is unavailable', () => {
    vi.spyOn(globalThis, 'crypto', 'get').mockReturnValue(undefined as unknown as Crypto);

    expect(randomHex()).toMatch(/^[0-9a-f]{8}$/);
  });
});

describe('newSessionId', () => {
  it('is prefixed so it reads as a session in the UI', () => {
    expect(newSessionId()).toMatch(/^sess-[0-9a-f]{8}$/);
  });
});
