import { describe, expect, it } from 'vitest';
import { stubFromCall } from './stubFromCall';
import type { CallRecord } from '../../lib/types';

const base: CallRecord = {
  service: 'greeter.Greeter',
  method: 'SayHello',
  timestamp: '2026-07-29T00:00:00Z',
  code: 0,
};

describe('stubFromCall', () => {
  it('maps a unary call to input.equals + output.data', () => {
    const stub = stubFromCall({ ...base, request: { name: 'a' }, response: { message: 'hi' } });
    expect(stub).toEqual({
      service: 'greeter.Greeter',
      method: 'SayHello',
      input: { equals: { name: 'a' } },
      output: { data: { message: 'hi' } },
    });
  });

  it('uses output.stream for multi-message responses', () => {
    const stub = stubFromCall({ ...base, requests: [{ n: 1 }], responses: [{ m: 1 }, { m: 2 }] });
    expect(stub.output).toEqual({ stream: [{ m: 1 }, { m: 2 }] });
    expect(stub.input).toEqual({ equals: { n: 1 } });
  });

  it('emits ordered inputs[] for multi-message requests (client-stream/bidi)', () => {
    const stub = stubFromCall({ ...base, requests: [{ n: 1 }, { n: 2 }, { n: 3 }], responses: [{ ok: true }] });
    expect(stub.inputs).toEqual([{ equals: { n: 1 } }, { equals: { n: 2 } }, { equals: { n: 3 } }]);
    expect(stub.input).toBeUndefined();
  });

  it('carries response headers into output.headers', () => {
    const stub = stubFromCall({ ...base, response: { ok: true }, responseHeaders: { 'x-trace': 'abc' } });
    expect((stub.output as Record<string, unknown>).headers).toEqual({ 'x-trace': 'abc' });
  });

  it('carries error and code for failed calls', () => {
    const stub = stubFromCall({ ...base, code: 5, error: 'not found' });
    expect(stub.output).toEqual({ data: {}, error: 'not found', code: 5 });
  });

  it('falls back to empty matcher and response when the record is bare', () => {
    const stub = stubFromCall(base);
    expect(stub.input).toEqual({ equals: {} });
    expect(stub.output).toEqual({ data: {} });
  });
});
