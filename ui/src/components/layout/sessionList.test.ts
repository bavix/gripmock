import { describe, it, expect } from 'vitest';
import { sessionOptions } from './sessionList';

describe('sessionOptions', () => {
  it('keeps recents first and appends server-known sessions', () => {
    expect(sessionOptions(['team-a'], ['team-b', 'team-a'], null)).toEqual(['team-a', 'team-b']);
  });

  it('always lists the active session, even when neither source knows it', () => {
    expect(sessionOptions([], [], 'typed-by-hand')).toEqual(['typed-by-hand']);
    expect(sessionOptions(['team-a'], ['team-b'], 'team-b')[0]).toBe('team-b');
  });

  it('drops blanks and duplicates and honours the limit', () => {
    expect(sessionOptions(['  ', 'a', 'a'], ['b'], null)).toEqual(['a', 'b']);
    expect(sessionOptions(['a', 'b', 'c'], [], null, 2)).toEqual(['a', 'b']);
  });
});
