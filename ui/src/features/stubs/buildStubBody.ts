// Pure stub-form <-> API payload builders, extracted from StubForm so they can
// be unit-tested without pulling in Monaco (which StubForm imports).
import { parse, hasKeys, buildOutput, type OutputMode } from './buildStubOutput';
import { outputTemplate } from '../../lib/stub';
import type { Stub } from '../../lib/types';

export interface StubFormData {
  service: string; method: string; priority: number; times: number;
  inputEquals: string; inputContains: string; inputMatches: string; inputGlob: string;
  inputIgnoreArrayOrder: boolean;
  inputAnyOf: { type: string; value: string; ignoreArrayOrder: boolean; _k?: string }[];
  inputsAlt: { type: string; value: string; ignoreArrayOrder: boolean; _k?: string }[];
  headersEquals: string; headersContains: string; headersMatches: string;
  headersAnyOf: { type: string; value: string; _k?: string }[];
  outputData: string; outputStream: string; outputTemplate: string;
  outputError: string; outputCode: number; outputDelay: string;
  outputHeaders: string; outputDetails: string;
  effects: { action: 'upsert' | 'delete'; id?: string; stub?: string; _k?: string }[];
}

export const INPUT_MODES = ['equals', 'contains', 'matches', 'glob', 'anyOf'] as const;
export const HEADER_MODES = ['equals', 'contains', 'matches', 'anyOf'] as const;
// First truthy matcher key from `kinds`, else `fallback`. Replaces nested ternaries.
function pickKind(a: Record<string, unknown>, kinds: readonly string[], fallback: string): string {
  return kinds.find((k) => a[k]) ?? fallback;
}

// The tab an existing matcher should open on, so editing a `contains` stub does not
// start on an empty `equals` editor.
export function matcherMode(matcher: unknown, modes: readonly string[]): string {
  const m = (matcher || {}) as Record<string, unknown>;
  if (Array.isArray(m.anyOf) && m.anyOf.length > 0) return 'anyOf';

  return pickKind(m, modes.filter((k) => k !== 'anyOf'), 'equals');
}

export function empty(): StubFormData {
  return {
    service: '', method: '', priority: 0, times: 0,
    inputEquals: '{}', inputContains: '{}', inputMatches: '{}', inputGlob: '{}', inputIgnoreArrayOrder: false, inputAnyOf: [], inputsAlt: [],
    headersEquals: '{}', headersContains: '{}', headersMatches: '{}', headersAnyOf: [],
    outputData: '{\n  \n}', outputStream: '', outputTemplate: '', outputError: '', outputCode: 0, outputDelay: '', outputHeaders: '{\n  \n}', outputDetails: '',
    effects: [],
  };
}

function isBadJson(s: string): boolean {
  const t = (s ?? '').trim();
  if (!t || t === '{}' || t === '[]') return false;
  try { JSON.parse(t); return false; } catch { return true; }
}

// Editors read by buildBody that must contain valid JSON.
const JSON_FIELDS: [keyof StubFormData, string][] = [
  ['inputEquals', 'input equals'], ['inputContains', 'input contains'], ['inputMatches', 'input matches'], ['inputGlob', 'input glob'],
  ['headersEquals', 'headers equals'], ['headersContains', 'headers contains'], ['headersMatches', 'headers matches'],
  ['outputData', 'response data'], ['outputStream', 'response stream'], ['outputHeaders', 'response headers'], ['outputDetails', 'error details'],
];

export function fromInit(init: Record<string, unknown>): StubFormData {
  const i = (init.input || {}) as Record<string, unknown>;
  const h = (init.headers || {}) as Record<string, unknown>;
  const o = (init.output || {}) as Record<string, unknown>;
  const e = (init.effects || []) as { action: 'upsert' | 'delete'; id?: string; stub?: string }[];
  const opts = (init.options || {}) as Record<string, unknown>;
  const ser = (a: unknown) => a ? JSON.stringify(a, null, 2) : '{}';
  return {
    service: (init.service as string) || '', method: (init.method as string) || '',
    priority: (init.priority as number) ?? 0, times: (opts.times as number) ?? 0,
    inputEquals: ser((i as any).equals), inputContains: ser((i as any).contains),
    inputMatches: ser((i as any).matches), inputGlob: ser((i as any).glob),
    inputIgnoreArrayOrder: !!(i as any).ignoreArrayOrder,
    inputAnyOf: ((i as any).anyOf || []).map((a: any, idx: number) => {
      const k = pickKind(a, ['equals', 'contains', 'matches'], 'glob');
      return { type: k, value: ser(a[k]), ignoreArrayOrder: !!a.ignoreArrayOrder, _k: `ia${idx}` };
    }),
    inputsAlt: ((init.inputs as any[]) || []).map((a: any, idx: number) => {
      // Preserve the matcher kind of each message (was previously flattened to equals).
      const k = pickKind(a, ['equals', 'contains', 'matches', 'glob'], 'equals');
      return { type: k, value: JSON.stringify(a[k] ?? {}, null, 2), ignoreArrayOrder: !!a.ignoreArrayOrder, _k: `in${idx}` };
    }),
    headersEquals: ser((h as any).equals), headersContains: ser((h as any).contains),
    headersMatches: ser((h as any).matches),
    headersAnyOf: ((h as any).anyOf || []).map((a: any, idx: number) => {
      const k = pickKind(a, ['equals', 'contains'], 'matches');
      return { type: k, value: ser(a[k]), _k: `ha${idx}` };
    }),
    outputData: o.data !== undefined ? JSON.stringify(o.data, null, 2) : '{\n  \n}',
    outputStream: (o as any).stream ? JSON.stringify((o as any).stream, null, 2) : '',
    outputTemplate: outputTemplate({ output: o } as unknown as Stub),
    outputError: (o as any).error || '', outputCode: (o as any).code ?? 0,
    outputDelay: (o as any).delay || '',
    outputHeaders: (o as any).headers ? JSON.stringify((o as any).headers, null, 2) : '{\n  \n}',
    outputDetails: (o as any).details ? JSON.stringify((o as any).details, null, 2) : '',
    effects: e.map((x, idx) => ({ action: x.action, id: x.id || '', stub: x.stub ? JSON.stringify(x.stub, null, 2) : '', _k: `ef${idx}` })),
  };
}

// Headers matcher (built before input, conventional order).
export function buildHeaders(f: StubFormData): Record<string, unknown> | undefined {
  const hd: Record<string, unknown> = {};
  const add = (k: string, v: unknown) => { if (hasKeys(v)) hd[k] = v; };
  add('equals', parse(f.headersEquals));
  add('contains', parse(f.headersContains));
  add('matches', parse(f.headersMatches));
  if (f.headersAnyOf.length) {
    hd.anyOf = f.headersAnyOf.map((a) => ({ [a.type]: parse(a.value) })).filter((a) => Object.values(a)[0]);
  }
  return Object.keys(hd).length > 0 ? hd : undefined;
}

export function buildInput(f: StubFormData): Record<string, unknown> {
  const inp: Record<string, unknown> = {};
  const add = (k: string, v: unknown) => { if (hasKeys(v)) inp[k] = v; };
  if (f.inputIgnoreArrayOrder) inp.ignoreArrayOrder = true;
  add('equals', parse(f.inputEquals));
  add('contains', parse(f.inputContains));
  add('matches', parse(f.inputMatches));
  add('glob', parse(f.inputGlob));
  if (f.inputAnyOf.length) {
    inp.anyOf = f.inputAnyOf.map((a) => {
      const it: Record<string, unknown> = { [a.type]: parse(a.value) };
      if (a.ignoreArrayOrder) it.ignoreArrayOrder = true;
      return it;
    }).filter((a) => hasKeys(Object.values(a)[0]));
  }
  return inp;
}

// inputs[]: ordered request messages (client/bidi streaming) or alternative
// matchers (unary). Each keeps its own matcher kind.
export function buildInputs(f: StubFormData): Record<string, unknown>[] {
  return f.inputsAlt.map((a) => {
    const v = parse(a.value);
    if (!hasKeys(v)) return null;
    return { [a.type || 'equals']: v, ...(a.ignoreArrayOrder ? { ignoreArrayOrder: true } : {}) };
  }).filter((a): a is Record<string, unknown> => a !== null);
}

export function buildEffects(f: StubFormData): Record<string, unknown>[] {
  return f.effects.map((e) => {
    if (e.action === 'delete') return { action: 'delete', id: e.id };
    const stub = parse(e.stub || '{}');
    return { action: 'upsert', stub: stub || {} };
  });
}

export function buildBody(f: StubFormData, initId: string | undefined, outMode: OutputMode, isTemplate = false): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (initId) body.id = initId;
  body.service = f.service;
  body.method = f.method;
  body.priority = f.priority;
  body.options = { times: f.times };

  const hd = buildHeaders(f);
  if (hd) body.headers = hd;

  const inp = buildInput(f);
  if (Object.keys(inp).length > (f.inputIgnoreArrayOrder ? 1 : 0)) body.input = inp;
  else if (!f.inputsAlt.length) body.input = { equals: {} };

  const alts = buildInputs(f);
  if (alts.length > 0) body.inputs = alts;

  body.output = buildOutput(f, outMode, isTemplate);

  if (f.effects.length) body.effects = buildEffects(f);

  return body;
}

// Editors whose invalid JSON would be silently nulled by buildBody.
export function collectJsonErrors(f: StubFormData): string[] {
  const out: string[] = [];
  for (const [k, label] of JSON_FIELDS) if (isBadJson(f[k] as string)) out.push(label);
  f.inputAnyOf.forEach((a, i) => { if (isBadJson(a.value)) out.push(`input anyOf #${i + 1}`); });
  f.headersAnyOf.forEach((a, i) => { if (isBadJson(a.value)) out.push(`header anyOf #${i + 1}`); });
  f.inputsAlt.forEach((a, i) => { if (isBadJson(a.value)) out.push(`alt input #${i + 1}`); });
  return out;
}
