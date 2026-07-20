import { describe, it, expect } from 'vitest';
import { isUuid, extractPayload, extractServiceMethod } from './useSearch';

describe('isUuid', () => {
  it('accepts full and dashless UUIDs and 8+ hex prefixes', () => {
    expect(isUuid('51c50050-ec27-4dae-a583-a32ca71a1dd5')).toBe(true);
    expect(isUuid('51c50050ec274daea583a32ca71a1dd5')).toBe(true);
    expect(isUuid('deadbeef')).toBe(true);
  });

  it('rejects non-hex and short strings', () => {
    expect(isUuid('GetProduct')).toBe(false);
    expect(isUuid('dead')).toBe(false);
    expect(isUuid('pkg.Svc/Method')).toBe(false);
  });
});

describe('extractPayload', () => {
  it('parses embedded JSON object', () => {
    expect(extractPayload('find {"a": 1} now')).toEqual({ a: 1 });
  });

  it('returns undefined for invalid or absent JSON', () => {
    expect(extractPayload('{broken')).toBeUndefined();
    expect(extractPayload('no json here')).toBeUndefined();
  });
});

describe('extractServiceMethod', () => {
  it('splits service/method on slash', () => {
    expect(extractServiceMethod('pkg.Svc/GetProduct')).toEqual({ service: 'pkg.Svc', method: 'GetProduct' });
  });

  it('splits dotted form: last segment is the method', () => {
    expect(extractServiceMethod('ecommerce.EcommerceService.GetProduct'))
      .toEqual({ service: 'ecommerce.EcommerceService', method: 'GetProduct' });
  });

  it('empty for plain words', () => {
    expect(extractServiceMethod('hello world')).toEqual({});
  });
});
