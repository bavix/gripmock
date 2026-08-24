import { describe, it, expect } from 'vitest';
import { buildOutput, type StubOutputFields } from './buildStubOutput';

const base: StubOutputFields = {
  outputData: '', outputStream: '', outputTemplate: '', outputHeaders: '{\n  \n}',
  outputError: '', outputCode: 0, outputDelay: '', outputDetails: '',
};

describe('buildOutput', () => {
  // Regression: truthiness checks (`if (d)`) silently dropped scalar 0/false/""
  // output values; the fix uses `!= null`.
  it('keeps scalar 0 / false / "" as data', () => {
    expect(buildOutput({ ...base, outputData: '0' }, 'data')).toEqual({ data: 0 });
    expect(buildOutput({ ...base, outputData: 'false' }, 'data')).toEqual({ data: false });
    expect(buildOutput({ ...base, outputData: '""' }, 'data')).toEqual({ data: '' });
  });

  it('keeps scalar stream and details values', () => {
    expect(buildOutput({ ...base, outputStream: '0' }, 'stream')).toEqual({ stream: 0 });
    expect(buildOutput({ ...base, outputDetails: 'false' }, 'data')).toEqual({ details: false });
  });

  it('falls back to { data: {} } when nothing meaningful is set', () => {
    expect(buildOutput(base, 'data')).toEqual({ data: {} });
  });

  it('puts the template document in the chosen slot', () => {
    const document = '{{ dict "a" 1 }}';
    expect(buildOutput({ ...base, outputTemplate: document }, 'data', true)).toEqual({ template: true, data: document });
    expect(buildOutput({ ...base, outputTemplate: document }, 'stream', true)).toEqual({ template: true, stream: document });
    expect(buildOutput({ ...base, outputTemplate: '   ' }, 'data', true)).toEqual({ data: {} });
  });

  it('drops invalid JSON (parse fails → no field)', () => {
    expect(buildOutput({ ...base, outputData: '{bad' }, 'data')).toEqual({ data: {} });
  });

  it('builds object data with headers, error and code', () => {
    const r = buildOutput(
      { ...base, outputData: '{"a":1}', outputHeaders: '{"h":"v"}', outputError: 'boom', outputCode: 5 },
      'data',
    );
    expect(r).toEqual({ data: { a: 1 }, headers: { h: 'v' }, error: 'boom', code: 5 });
  });

  it('only emits stream in stream mode (not data)', () => {
    expect(buildOutput({ ...base, outputData: '{"a":1}', outputStream: '[1,2]' }, 'stream'))
      .toEqual({ stream: [1, 2] });
  });
});
