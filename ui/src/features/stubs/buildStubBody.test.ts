import { describe, it, expect } from 'vitest';
import {
  empty, fromInit, buildBody, buildInput, buildInputs, buildHeaders, buildEffects, collectJsonErrors,
  matcherMode, INPUT_MODES, HEADER_MODES,
  type StubFormData,
} from './buildStubBody';

const base = (over: Partial<StubFormData> = {}): StubFormData => ({ ...empty(), service: 's', method: 'm', ...over });

describe('buildInput', () => {
  it('collects only non-empty matchers', () => {
    const inp = buildInput(base({ inputEquals: '{"a":1}', inputContains: '{}', inputGlob: '{"g":"x*"}' }));
    expect(inp).toEqual({ equals: { a: 1 }, glob: { g: 'x*' } });
  });

  it('carries ignoreArrayOrder', () => {
    expect(buildInput(base({ inputEquals: '{"a":1}', inputIgnoreArrayOrder: true }))).toEqual({
      ignoreArrayOrder: true, equals: { a: 1 },
    });
  });

  it('builds anyOf preserving each matcher kind, dropping empties', () => {
    const inp = buildInput(base({
      inputAnyOf: [
        { type: 'contains', value: '{"a":1}', ignoreArrayOrder: false },
        { type: 'glob', value: '{}', ignoreArrayOrder: false }, // empty → dropped
      ],
    }));
    expect(inp.anyOf).toEqual([{ contains: { a: 1 } }]);
  });
});

describe('buildInputs (ordered stream messages)', () => {
  it('preserves per-message matcher kind and order', () => {
    const inputs = buildInputs(base({
      inputsAlt: [
        { type: 'equals', value: '{"n":1}', ignoreArrayOrder: false },
        { type: 'contains', value: '{"n":2}', ignoreArrayOrder: true },
      ],
    }));
    expect(inputs).toEqual([
      { equals: { n: 1 } },
      { contains: { n: 2 }, ignoreArrayOrder: true },
    ]);
  });

  it('drops empty messages', () => {
    expect(buildInputs(base({ inputsAlt: [{ type: 'equals', value: '{}', ignoreArrayOrder: false }] }))).toEqual([]);
  });
});

describe('buildBody', () => {
  it('assembles core fields and defaults empty input to {equals:{}}', () => {
    const body = buildBody(base(), undefined, 'data');
    expect(body.service).toBe('s');
    expect(body.method).toBe('m');
    expect(body.options).toEqual({ times: 0 });
    expect(body.input).toEqual({ equals: {} });
    expect(body.inputs).toBeUndefined();
  });

  it('does NOT default input when ordered inputs[] are present', () => {
    const body = buildBody(base({ inputsAlt: [{ type: 'equals', value: '{"n":1}', ignoreArrayOrder: false }] }), undefined, 'data');
    expect(body.input).toBeUndefined();
    expect(body.inputs).toEqual([{ equals: { n: 1 } }]);
  });

  it('does not emit an input carrying only ignoreArrayOrder', () => {
    // ignoreArrayOrder alone (no matcher) must not become body.input — it falls
    // back to the {equals:{}} default.
    const body = buildBody(base({ inputIgnoreArrayOrder: true }), undefined, 'data');
    expect(body.input).toEqual({ equals: {} });
  });

  it('includes id only when editing', () => {
    expect(buildBody(base(), 'abc', 'data').id).toBe('abc');
    expect(buildBody(base(), undefined, 'data').id).toBeUndefined();
  });
});

describe('buildHeaders / buildEffects', () => {
  it('builds header anyOf', () => {
    expect(buildHeaders(base({ headersAnyOf: [{ type: 'equals', value: '{"x":"1"}' }] }))).toEqual({
      anyOf: [{ equals: { x: '1' } }],
    });
  });

  it('returns undefined with no header matchers', () => {
    expect(buildHeaders(base())).toBeUndefined();
  });

  it('builds upsert and delete effects', () => {
    const eff = buildEffects(base({
      effects: [
        { action: 'delete', id: 'id-1' },
        { action: 'upsert', stub: '{"service":"x"}' },
      ],
    }));
    expect(eff).toEqual([
      { action: 'delete', id: 'id-1' },
      { action: 'upsert', stub: { service: 'x' } },
    ]);
  });
});

describe('fromInit → buildBody round-trip', () => {
  it('preserves matcher kinds through a round-trip', () => {
    const initial = {
      service: 'pkg.Svc', method: 'Method', priority: 3,
      input: { contains: { name: 'a' }, glob: { id: 'x*' } },
      output: { data: { ok: true } },
    };
    const body = buildBody(fromInit(initial), undefined, 'data');
    expect(body.service).toBe('pkg.Svc');
    expect(body.priority).toBe(3);
    expect(body.input).toEqual({ contains: { name: 'a' }, glob: { id: 'x*' } });
    expect(body.output).toEqual({ data: { ok: true } });
  });

  it('keeps the template when a template stub is edited and saved', () => {
    const document = '{{ dict "a" 1 }}';
    const initial = {
      id: 'abc', service: 'pkg.Svc', method: 'Method',
      input: { equals: { n: 1 } },
      output: { template: true, data: document },
    };
    const body = buildBody(fromInit(initial), 'abc', 'data', true);
    expect(body.output).toEqual({ template: true, data: document });

    const streamed = { ...initial, output: { template: true, stream: document } };
    expect(buildBody(fromInit(streamed), 'abc', 'stream', true).output).toEqual({ template: true, stream: document });
  });

  it('preserves ordered inputs[] matcher kinds', () => {
    const initial = {
      service: 's', method: 'm',
      inputs: [{ equals: { n: 1 } }, { contains: { n: 2 } }],
      output: { data: {} },
    };
    const body = buildBody(fromInit(initial), undefined, 'data');
    expect(body.inputs).toEqual([{ equals: { n: 1 } }, { contains: { n: 2 } }]);
  });
});

describe('collectJsonErrors', () => {
  it('flags malformed JSON editors by label', () => {
    const errs = collectJsonErrors(base({ inputEquals: '{bad', outputData: '{"ok":true}' }));
    expect(errs).toContain('input equals');
    expect(errs).not.toContain('response data');
  });

  it('is empty for valid/blank editors', () => {
    expect(collectJsonErrors(base())).toEqual([]);
  });
});

describe('matcherMode', () => {
  it('opens on the kind the stub actually uses', () => {
    expect(matcherMode({ contains: { a: 1 } }, INPUT_MODES)).toBe('contains');
    expect(matcherMode({ matches: { a: '.*' } }, INPUT_MODES)).toBe('matches');
    expect(matcherMode({ glob: { a: '*' } }, INPUT_MODES)).toBe('glob');
    expect(matcherMode({ anyOf: [{ equals: { a: 1 } }] }, INPUT_MODES)).toBe('anyOf');
  });

  it('falls back to equals for an empty or unset matcher', () => {
    expect(matcherMode(undefined, INPUT_MODES)).toBe('equals');
    expect(matcherMode({ equals: null, contains: null, matches: null }, INPUT_MODES)).toBe('equals');
  });

  it('never picks a kind the matcher does not offer', () => {
    expect(matcherMode({ glob: { a: '*' } }, HEADER_MODES)).toBe('equals');
  });
});
