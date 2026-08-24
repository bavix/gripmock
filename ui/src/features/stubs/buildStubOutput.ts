// Pure helpers for turning the StubForm's output-* fields into a stub Output
// object. Extracted from StubForm so they are unit-testable without pulling in
// the Monaco-backed component module.

export interface StubOutputFields {
  outputData: string;
  outputStream: string;
  outputTemplate: string;
  outputHeaders: string;
  outputError: string;
  outputCode: number;
  outputDelay: string;
  outputDetails: string;
}

export type OutputMode = 'data' | 'stream';

export function parse(s: string): unknown {
  try { return s ? JSON.parse(s) : null; } catch { return null; }
}

export const hasKeys = (v: unknown): boolean => !!v && Object.keys(v as object).length > 0;

// NOTE: data/stream/details use `!= null` (not truthiness) so a valid scalar
// response of 0, false or "" is not silently dropped.
// `template` marks the chosen slot as a Go template: the text lives in data or
// stream, exactly where the literal payload would.
export function buildOutput(f: StubOutputFields, outMode: OutputMode, isTemplate = false): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  if (isTemplate && f.outputTemplate.trim()) {
    out.template = true;
    out[outMode] = f.outputTemplate;
  }
  if (!isTemplate && outMode === 'data') { const d = parse(f.outputData); if (d != null) out.data = d; }
  if (!isTemplate && outMode === 'stream') { const s = parse(f.outputStream); if (s != null) out.stream = s; }
  const oh = parse(f.outputHeaders); if (hasKeys(oh)) out.headers = oh;
  if (f.outputError) { out.error = f.outputError; out.code = f.outputCode; }
  if (f.outputDelay) out.delay = f.outputDelay;
  const dd = parse(f.outputDetails); if (dd != null) out.details = dd;
  return Object.keys(out).length > 0 ? out : { data: {} };
}
