import { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { useCreateStub, useUpdateStub, useStubs } from '../../hooks/useStubs';
import { useServiceMethod } from '../../hooks/useServices';
import { MethodSelect } from '../../components/shared/MethodSelect';
import { MonacoEditor } from '../../components/json/MonacoEditor';
import { Save, Plus, X, ChevronDown, ChevronRight, ArrowLeft, Sparkles, Copy, Loader2, AlertCircle, Play, Trophy } from 'lucide-react';
import { api } from '../../lib/api';
import { colors } from '../../lib/theme';
import { shadowers, isRequestStream } from '../../lib/stub';
import { MessageSequenceEditor } from './MessageSequenceEditor';
import { toYaml } from './toYaml';
import { generateSample } from './generateSample';
import { highlightYaml } from './highlightYaml';

/* ── Types ── */

interface StubFormData {
  service: string; method: string; priority: number; times: number;
  inputEquals: string; inputContains: string; inputMatches: string; inputGlob: string;
  inputIgnoreArrayOrder: boolean;
  inputAnyOf: { type: string; value: string; ignoreArrayOrder: boolean }[];
  inputsAlt: { type: string; value: string; ignoreArrayOrder: boolean }[];
  headersEquals: string; headersContains: string; headersMatches: string;
  headersAnyOf: { type: string; value: string }[];
  outputData: string; outputStream: string;
  outputError: string; outputCode: number; outputDelay: string;
  outputHeaders: string; outputDetails: string;
  effects: { action: 'upsert' | 'delete'; id?: string; stub?: string }[];
}

interface Props { initial?: Record<string, unknown>; onSaved?: () => void; }

/* ── Constants ── */

const INPUT_MODES = ['equals', 'contains', 'matches', 'glob', 'anyOf'] as const;
const HEADER_MODES = ['equals', 'contains', 'matches', 'anyOf'] as const;
const GRPC_CODES = [
  { value: 0, label: 'OK' }, { value: 1, label: 'Canceled' }, { value: 2, label: 'Unknown' },
  { value: 3, label: 'InvalidArgument' }, { value: 4, label: 'DeadlineExceeded' }, { value: 5, label: 'NotFound' },
  { value: 6, label: 'AlreadyExists' }, { value: 7, label: 'PermissionDenied' }, { value: 8, label: 'ResourceExhausted' },
  { value: 9, label: 'FailedPrecondition' }, { value: 10, label: 'Aborted' }, { value: 11, label: 'OutOfRange' },
  { value: 12, label: 'Unimplemented' }, { value: 13, label: 'Internal' }, { value: 14, label: 'Unavailable' },
  { value: 15, label: 'DataLoss' }, { value: 16, label: 'Unauthenticated' },
];

function empty(): StubFormData { return {
  service: '', method: '', priority: 0, times: 0,
  inputEquals: '{}', inputContains: '{}', inputMatches: '{}', inputGlob: '{}', inputIgnoreArrayOrder: false, inputAnyOf: [], inputsAlt: [],
  headersEquals: '{}', headersContains: '{}', headersMatches: '{}', headersAnyOf: [],
  outputData: '{\n  \n}', outputStream: '', outputError: '', outputCode: 0, outputDelay: '', outputHeaders: '{\n  \n}', outputDetails: '',
  effects: [],
}; }

function parse(s: string): unknown { try { return s ? JSON.parse(s) : null; } catch { return null; } }

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

function fromInit(init: Record<string, unknown>): StubFormData {
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
    inputAnyOf: ((i as any).anyOf || []).map((a: any) => {
      const k = a.equals ? 'equals' : a.contains ? 'contains' : a.matches ? 'matches' : 'glob';
      return { type: k, value: ser(a[k]), ignoreArrayOrder: !!a.ignoreArrayOrder };
    }),
    inputsAlt: ((init.inputs as any[]) || []).map((a: any) => {
      // Preserve the matcher kind of each message (was previously flattened to equals).
      const k = a.equals ? 'equals' : a.contains ? 'contains' : a.matches ? 'matches' : a.glob ? 'glob' : 'equals';
      return { type: k, value: JSON.stringify(a[k] ?? {}, null, 2), ignoreArrayOrder: !!a.ignoreArrayOrder };
    }),
    headersEquals: ser((h as any).equals), headersContains: ser((h as any).contains),
    headersMatches: ser((h as any).matches),
    headersAnyOf: ((h as any).anyOf || []).map((a: any) => {
      const k = a.equals ? 'equals' : a.contains ? 'contains' : 'matches';
      return { type: k, value: ser(a[k]) };
    }),
    outputData: o.data !== undefined ? JSON.stringify(o.data, null, 2) : '{\n  \n}',
    outputStream: (o as any).stream ? JSON.stringify((o as any).stream, null, 2) : '',
    outputError: (o as any).error || '', outputCode: (o as any).code ?? 0,
    outputDelay: (o as any).delay || '',
    outputHeaders: (o as any).headers ? JSON.stringify((o as any).headers, null, 2) : '{\n  \n}',
    outputDetails: (o as any).details ? JSON.stringify((o as any).details, null, 2) : '',
    effects: e.map((x) => ({ action: x.action, id: x.id || '', stub: x.stub ? JSON.stringify(x.stub, null, 2) : '' })),
  };
}

function buildBody(f: StubFormData, initId: string | undefined, outMode: 'data' | 'stream'): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (initId) body.id = initId;
  body.service = f.service;
  body.method = f.method;
  body.priority = f.priority;
  body.options = { times: f.times };

  // Headers matcher (before input, conventional order)
  const hd: Record<string, unknown> = {};
  addHd('equals', parse(f.headersEquals));
  addHd('contains', parse(f.headersContains));
  addHd('matches', parse(f.headersMatches));
  if (f.headersAnyOf.length) {
    hd.anyOf = f.headersAnyOf.map((a) => ({ [a.type]: parse(a.value) })).filter((a) => Object.values(a)[0]);
  }
  if (Object.keys(hd).length > 0) body.headers = hd;

  const inp: Record<string, unknown> = {};
  if (f.inputIgnoreArrayOrder) inp.ignoreArrayOrder = true;
  addInp('equals', parse(f.inputEquals));
  addInp('contains', parse(f.inputContains));
  addInp('matches', parse(f.inputMatches));
  addInp('glob', parse(f.inputGlob));
  if (f.inputAnyOf.length) {
    inp.anyOf = f.inputAnyOf.map((a) => {
      const it: Record<string, unknown> = { [a.type]: parse(a.value) };
      if (a.ignoreArrayOrder) it.ignoreArrayOrder = true;
      return it;
    }).filter((a) => { const v = Object.values(a)[0]; return v && Object.keys(v as object).length; });
  }
  if (Object.keys(inp).length > (f.inputIgnoreArrayOrder ? 1 : 0)) body.input = inp;
  else if (!f.inputsAlt.length) body.input = { equals: {} };

  // inputs[]: ordered request messages (client/bidi streaming) or alternative
  // matchers (unary). Each keeps its own matcher kind.
  const alts = f.inputsAlt.map((a) => {
    const v = parse(a.value);
    if (!v || !Object.keys(v as object).length) return null;
    return { [a.type || 'equals']: v, ...(a.ignoreArrayOrder ? { ignoreArrayOrder: true } : {}) };
  }).filter(Boolean);
  if (alts.length > 0) body.inputs = alts;

  const out: Record<string, unknown> = {};
  if (outMode === 'data') { const d = parse(f.outputData); if (d) out.data = d; }
  if (outMode === 'stream') { const s = parse(f.outputStream); if (s) out.stream = s; }
  const oh = parse(f.outputHeaders); if (oh && Object.keys(oh as object).length) out.headers = oh;
  if (f.outputError) { out.error = f.outputError; out.code = f.outputCode; }
  if (f.outputDelay) out.delay = f.outputDelay;
  const dd = parse(f.outputDetails); if (dd) out.details = dd;
  body.output = Object.keys(out).length > 0 ? out : { data: {} };

  if (f.effects.length) {
    body.effects = f.effects.map((e) => {
      if (e.action === 'delete') return { action: 'delete', id: e.id };
      const stub = parse(e.stub || '{}');
      return { action: 'upsert', stub: stub || {} };
    });
  }

  function addInp(k: string, v: unknown) { if (v && Object.keys(v as object).length) inp[k] = v; }
  function addHd(k: string, v: unknown) { if (v && Object.keys(v as object).length) hd[k] = v; }

  return body;
}

/* ── Component ── */

export function StubForm({ initial, onSaved }: Props) {
  const navigate = useNavigate();
  const create = useCreateStub();
  const update = useUpdateStub();
  const [f, setF] = useState<StubFormData>(() => initial ? fromInit(initial) : empty());
  const [sub, setSub] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [outMode, setOutMode] = useState<'data' | 'stream'>('data');
  const [inpMode, setInpMode] = useState('equals');
  const [hdrMode, setHdrMode] = useState('equals');
  const { data: methodSchema } = useServiceMethod(f.service || null, f.method || null);
  const { data: allStubs } = useStubs();
  const initId = (initial as any)?.id;
  const isStreamMethod = methodSchema?.methodType === 'server_streaming' || methodSchema?.methodType === 'bidi_streaming';
  const isReqStreamMethod = isRequestStream(methodSchema?.methodType);

  // Stubs on the same method with higher priority — they'd be matched first.
  const shadows = useMemo(() => {
    if (!allStubs || !f.service || !f.method) return [];
    return shadowers({ id: initId ?? '', service: f.service, method: f.method, priority: f.priority } as any, allStubs);
  }, [allStubs, f.service, f.method, f.priority, initId]);

  useEffect(() => {
    if (!initial) return;
    setF(fromInit(initial));
    const o = (initial.output || {}) as Record<string, unknown>;
    if ((o as any).stream) setOutMode('stream');
  }, [initial]);

  // For a server/bidi-streaming method the response is a stream — default to it
  // (only when creating; don't override an explicit edit choice).
  useEffect(() => {
    if (initial) return;
    setOutMode(isStreamMethod ? 'stream' : 'data');
  }, [isStreamMethod, initial]);

  const patch = useCallback((p: Partial<StubFormData>) => setF((v) => ({ ...v, ...p })), []);

  const initialFormRef = useRef<string>(JSON.stringify(initial ? fromInit(initial) : empty()));
  useEffect(() => { initialFormRef.current = JSON.stringify(initial ? fromInit(initial) : empty()); }, [initial]);
  const dirty = JSON.stringify(f) !== initialFormRef.current;
  useEffect(() => {
    const h = (e: BeforeUnloadEvent) => { if (dirty) { e.preventDefault(); e.returnValue = ''; } };
    window.addEventListener('beforeunload', h);
    return () => window.removeEventListener('beforeunload', h);
  }, [dirty]);
  const leave = () => { if (!dirty || confirm('Discard unsaved changes?')) (onSaved ? onSaved() : navigate('/stubs')); };

  const handleGenerate = () => {
    if (!methodSchema?.requestSchema) return;
    patch({ inputEquals: JSON.stringify(generateSample(methodSchema.requestSchema), null, 2) });
    setInpMode('equals');
  };

  const handleGenerateResponse = () => {
    if (!methodSchema?.responseSchema) return;
    const sample = generateSample(methodSchema.responseSchema);
    if (outMode === 'stream') patch({ outputStream: JSON.stringify([sample], null, 2) });
    else patch({ outputData: JSON.stringify(sample, null, 2) });
  };

  // Catch invalid JSON in any editor BEFORE buildBody silently nulls it
  // (which would otherwise produce an unintended match-anything stub).
  const jsonErrors = useMemo(() => {
    const out: string[] = [];
    for (const [k, label] of JSON_FIELDS) if (isBadJson(f[k] as string)) out.push(label);
    f.inputAnyOf.forEach((a, i) => { if (isBadJson(a.value)) out.push(`input anyOf #${i + 1}`); });
    f.headersAnyOf.forEach((a, i) => { if (isBadJson(a.value)) out.push(`header anyOf #${i + 1}`); });
    f.inputsAlt.forEach((a, i) => { if (isBadJson(a.value)) out.push(`alt input #${i + 1}`); });
    return out;
  }, [f]);

  const handleSubmit = async () => {
    if (jsonErrors.length > 0) { setErr(`Invalid JSON in: ${jsonErrors.join(', ')}`); return; }
    setSub(true); setErr(null);
    try {
      const body = buildBody(f, initId, outMode);
      if (initId) await update.mutateAsync(body as any);
      else await create.mutateAsync(body);
      if (onSaved) onSaved(); else navigate('/stubs');
    } catch (e) { setErr((e as Error).message); }
    finally { setSub(false); }
  };

  /* ── Validation / Preview ── */

  const [validJson, setValidJson] = useState<unknown>(null);
  const [validErr, setValidErr] = useState<string | null>(null);
  const [validBusy, setValidBusy] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const bodySnapshot = useMemo(() => buildBody(f, initId, outMode), [f, initId, outMode]);

  useEffect(() => {
    if (!f.service || !f.method) { setValidJson(null); setValidErr(null); return; }
    clearTimeout(timer.current);
    setValidBusy(true);
    timer.current = setTimeout(async () => {
      try {
        const res = await api.post('/stubs/validate', [bodySnapshot]);
        setValidJson(res);
        setValidErr(null);
      } catch (e) { setValidJson(null); setValidErr((e as Error).message); }
      setValidBusy(false);
    }, 400);
    return () => clearTimeout(timer.current);
  }, [bodySnapshot, f.service, f.method]);

  const yaml = useMemo(() => {
    if (validErr || !validJson) return null;
    return toYaml(validJson);
  }, [validJson, validErr]);

  /* ── Render ── */

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 420px', gap: 16, alignItems: 'start' }}>
      {/* ── Left: form ── */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {/* Header */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, paddingBottom: 8, borderBottom: '1px solid var(--border)' }}>
          <button onClick={leave} className="btn btn-ghost" style={{ fontSize: 11 }}><ArrowLeft size={13} /> Back</button>
          <span style={{ fontSize: 14, fontWeight: 600, flex: 1 }}>{initId ? 'Edit Stub' : 'Create Stub'}</span>
          <button onClick={handleSubmit} disabled={sub || !f.service || !f.method || jsonErrors.length > 0} className="btn btn-primary" style={{ fontSize: 11 }}>
            <Save size={12} /> {sub ? 'Saving…' : 'Save'}
          </button>
        </div>
        {err && <div style={{ padding: '7px 10px', borderRadius: 5, background: 'var(--error-bg)', color: colors.error, fontSize: 12 }}>{err}</div>}
        {jsonErrors.length > 0 && !err && (
          <div style={{ padding: '7px 10px', borderRadius: 5, background: 'var(--warning-bg)', color: colors.warning, fontSize: 12, display: 'flex', alignItems: 'center', gap: 6 }}>
            <AlertCircle size={13} /> Invalid JSON in: {jsonErrors.join(', ')} — fix before saving.
          </div>
        )}
        {shadows.length > 0 && (
          <div style={{ padding: '7px 10px', borderRadius: 5, background: 'var(--warning-bg)', color: colors.warning, fontSize: 12, display: 'flex', alignItems: 'flex-start', gap: 6 }}>
            <Trophy size={13} style={{ flexShrink: 0, marginTop: 1 }} />
            <span>{shadows.length} higher-priority stub{shadows.length > 1 ? 's' : ''} on this method may match first (priority {'>'} {f.priority}). Raise this stub's priority if it should win.</span>
          </div>
        )}

        {/* Service & Method */}
        <Section>
          <MethodSelect service={f.service} method={f.method}
            onServiceChange={(s) => patch({ service: s, method: '' })}
            onMethodChange={(m) => patch({ method: m })} />
        </Section>

        {/* Priority & Times */}
        <Section label="Priority & Times">
          <div style={{ display: 'flex', gap: 10 }}>
            <label style={lbl}>Priority <input type="number" value={f.priority} onChange={(e) => patch({ priority: Number(e.target.value) })} className="input" style={{ width: 70, display: 'block', marginTop: 1 }} /></label>
            <label style={lbl}>Times (0=∞) <input type="number" min={0} value={f.times} onChange={(e) => patch({ times: Number(e.target.value) })} className="input" style={{ width: 70, display: 'block', marginTop: 1 }} /></label>
          </div>
        </Section>

        {/* Input Matcher */}
        <Section label="Input Matcher">
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 11, cursor: 'pointer', color: 'var(--text-secondary)' }}>
              <input type="checkbox" checked={f.inputIgnoreArrayOrder} onChange={(e) => patch({ inputIgnoreArrayOrder: e.target.checked })} /> Ignore array order
            </label>
            <div style={{ flex: 1 }} />
            {methodSchema?.requestSchema && (
              <button onClick={handleGenerate} className="btn" style={{ fontSize: 11, padding: '2px 7px' }}><Sparkles size={10} /> Generate</button>
            )}
          </div>
          <Tabs modes={INPUT_MODES} mode={inpMode} onChange={setInpMode} />
          <MatcherMode mode={inpMode} value={f} onChange={patch} prefix="input" anyOfItems={f.inputAnyOf} onAnyOfChange={(v) => patch({ inputAnyOf: v })} />
        </Section>

        {/* Response */}
        <Section label="Response">
          <div style={{ display: 'flex', gap: 3, marginBottom: 4, flexWrap: 'wrap' }}>
            {['data', 'stream'].map((m) => (
              <button key={m} onClick={() => setOutMode(m as any)} className={`btn ${outMode === m ? 'btn-primary' : ''}`} style={{ fontSize: 11, padding: '2px 8px' }}>{m}</button>
            ))}
            <div style={{ flex: 1 }} />
            {methodSchema?.responseSchema && (
              <button onClick={handleGenerateResponse} className="btn" style={{ fontSize: 11, padding: '2px 8px' }}><Sparkles size={10} /> Generate</button>
            )}
            <button onClick={() => patch({ outputError: f.outputError ? '' : 'error' })} className={`btn ${f.outputError ? 'btn-danger' : ''}`} style={{ fontSize: 11, padding: '2px 8px' }}>Error</button>
            <button onClick={() => patch({ outputDelay: f.outputDelay ? '' : '500ms' })} className="btn" style={{ fontSize: 11, padding: '2px 8px' }}>Delay</button>
          </div>
          {outMode === 'data' && <MonacoEditor value={f.outputData} onChange={(v) => patch({ outputData: v })} height={140} />}
          {outMode === 'stream' && <MonacoEditor value={f.outputStream} onChange={(v) => patch({ outputStream: v })} height={140} />}

          {f.outputError && (
            <div style={{ marginTop: 6, padding: 8, borderRadius: 5, border: '1px solid var(--error)', background: 'var(--errorBg)' }}>
              <div style={{ display: 'flex', gap: 8 }}>
                <div><Label>Code</Label><select value={f.outputCode} onChange={(e) => patch({ outputCode: Number(e.target.value) })} className="input" style={{ width: 160, marginTop: 1, fontSize: 11 }}>
                  {GRPC_CODES.map((c) => <option key={c.value} value={c.value}>{c.label} ({c.value})</option>)}
                </select></div>
                <div style={{ flex: 1 }}><Label>Message</Label><input value={f.outputError} onChange={(e) => patch({ outputError: e.target.value })} placeholder="error description" className="input" style={{ marginTop: 1, fontSize: 11 }} /></div>
              </div>
            </div>
          )}

          {f.outputDelay && (
            <div style={{ display: 'flex', gap: 6, alignItems: 'center', marginTop: 6 }}>
              <Label>Delay:</Label>
              <input value={f.outputDelay} onChange={(e) => patch({ outputDelay: e.target.value })} placeholder="500ms, 2s, 1m30s" className="input" style={{ fontFamily: 'monospace', width: 130, fontSize: 11 }} />
            </div>
          )}

          <div style={{ marginTop: 8 }}><Label>Response Headers</Label><MonacoEditor value={f.outputHeaders} onChange={(v) => patch({ outputHeaders: v })} height={70} /></div>
          <div style={{ marginTop: 8 }}><Label>Error Details (protobuf Any)</Label><MonacoEditor value={f.outputDetails} onChange={(v) => patch({ outputDetails: v })} height={70} /></div>
        </Section>

        {/* Optional sections */}
        <Collapse label="Headers Matcher">
          <Tabs modes={HEADER_MODES} mode={hdrMode} onChange={setHdrMode} />
          <MatcherMode mode={hdrMode} value={f} onChange={patch} prefix="headers" anyOfItems={f.headersAnyOf} onAnyOfChange={(v) => patch({ headersAnyOf: v })} />
        </Collapse>

        <Collapse
          label={isReqStreamMethod ? `Request message sequence${f.inputsAlt.length ? ` (${f.inputsAlt.length})` : ''}` : `Alternative matchers${f.inputsAlt.length ? ` (${f.inputsAlt.length})` : ''}`}
          defaultOpen={isReqStreamMethod && f.inputsAlt.length > 0}
        >
          <MessageSequenceEditor items={f.inputsAlt} onChange={(v) => patch({ inputsAlt: v })} streaming={isReqStreamMethod} />
        </Collapse>

        <Collapse label="Effects">
          <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4 }}>Side effects triggered on match.</div>
          {f.effects.length === 0 && <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4 }}>None.</div>}
          {f.effects.map((e, i) => (
            <div key={i} style={{ padding: 8, borderRadius: 5, border: '1px solid var(--border)', marginBottom: 4, position: 'relative' }}>
              <button onClick={() => patch({ effects: f.effects.filter((_, j) => j !== i) })} className="btn btn-ghost" style={{ position: 'absolute', top: 4, right: 4, padding: '1px 5px' }}><X size={11} /></button>
              <select value={e.action} onChange={(ev) => { const n = [...f.effects]; n[i] = { ...n[i], action: ev.target.value as any }; patch({ effects: n }); }} className="input" style={{ fontSize: 11, width: 90, marginBottom: 3 }}>
                <option value="upsert">Upsert</option><option value="delete">Delete</option>
              </select>
              {e.action === 'delete' && <input value={e.id || ''} onChange={(ev) => { const n = [...f.effects]; n[i] = { ...n[i], id: ev.target.value }; patch({ effects: n }); }} placeholder="Stub UUID to delete" className="input" style={{ fontFamily: 'monospace', marginTop: 2, fontSize: 11 }} />}
              {e.action === 'upsert' && <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 1 }}>Full stub JSON upserted on match.<MonacoEditor value={e.stub || ''} onChange={(v) => { const n = [...f.effects]; n[i] = { ...n[i], stub: v }; patch({ effects: n }); }} height={70} /></div>}
            </div>
          ))}
          <button onClick={() => patch({ effects: [...f.effects, { action: 'upsert' }] })} className="btn btn-ghost" style={{ fontSize: 11 }}><Plus size={9} /> Add</button>
        </Collapse>

        {/* Bottom save */}
        <button onClick={handleSubmit} disabled={sub || !f.service || !f.method || jsonErrors.length > 0} className="btn btn-primary" style={{ alignSelf: 'flex-start', fontSize: 11 }}>
          <Save size={12} /> {sub ? 'Saving…' : 'Save Stub'}
        </button>
      </div>

      {/* ── Right: preview ── */}
      <div style={{ position: 'sticky', top: 0 }}>
        <div style={{ borderRadius: 6, border: '1px solid var(--border)', overflow: 'hidden', background: 'var(--bg)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '6px 10px', borderBottom: '1px solid var(--border)', background: 'var(--bg-secondary)', fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.3px' }}>
            <span style={{ flex: 1 }}>Preview</span>
            {yaml && <button onClick={() => navigator.clipboard.writeText(yaml)} className="btn btn-ghost" style={{ fontSize: 11, padding: '1px 5px' }}><Copy size={9} /></button>}
          </div>
          <pre style={{ margin: 0, padding: 10, fontSize: 11, lineHeight: 1.5, fontFamily: 'var(--mono)', overflow: 'auto', maxHeight: 'calc(100vh - 140px)', background: 'var(--bg)', whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
            {validBusy ? (
              <span style={{ display: 'flex', alignItems: 'center', gap: 5, color: 'var(--text-muted)', fontSize: 11 }}><Loader2 size={11} className="animate-spin" /> Validating…</span>
            ) : validErr ? (
              <span style={{ display: 'flex', alignItems: 'flex-start', gap: 5, color: 'var(--error)', fontSize: 11 }}><AlertCircle size={12} style={{ flexShrink: 0, marginTop: 1 }} /><span>{validErr}</span></span>
            ) : yaml ? (
              <span>{highlightYaml(yaml)}</span>
            ) : (
              <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>Fill in service/method to see preview</span>
            )}
          </pre>
        </div>
        {f.service && f.method && (
          <div style={{ marginTop: 6, display: 'flex', gap: 4 }}>
            <button onClick={() => {
              const payload = (parse(f.inputEquals) ?? parse(f.inputContains) ?? parse(f.inputMatches) ?? {}) as unknown;
              const hdrs = (parse(f.headersEquals) ?? {}) as unknown;
              navigate(`/stubs/test?service=${encodeURIComponent(f.service)}&method=${encodeURIComponent(f.method)}&payload=${encodeURIComponent(JSON.stringify(payload, null, 2))}&headers=${encodeURIComponent(JSON.stringify(hdrs, null, 2))}`);
            }} className="btn" style={{ flex: 1, fontSize: 11, padding: '3px 6px' }} title="Test which stub this request would match"><Play size={10} /> Test match</button>
          </div>
        )}
      </div>
    </div>
  );
}

/* ── Sub-components ── */

function Section({ label, children }: { label?: string; children: React.ReactNode }) {
  return (
    <div style={{ borderRadius: 6, border: '1px solid var(--border)', background: 'var(--bg-secondary)' }}>
      {label && <div style={{ padding: '6px 10px', borderBottom: '1px solid var(--border)', fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.3px' }}>{label}</div>}
      <div style={{ padding: 10 }}>{children}</div>
    </div>
  );
}

function Label({ children }: { children: React.ReactNode }) {
  return <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.2px', marginBottom: 1 }}>{children}</div>;
}

const lbl: React.CSSProperties = { fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.2px' };

function Collapse({ label, children, defaultOpen = false }: { label: string; children: React.ReactNode; defaultOpen?: boolean }) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div style={{ borderRadius: 6, border: '1px solid var(--border)', background: open ? 'var(--bg-secondary)' : 'transparent', overflow: 'hidden' }}>
      <div onClick={() => setOpen(!open)} style={{ display: 'flex', alignItems: 'center', gap: 5, cursor: 'pointer', userSelect: 'none', padding: '6px 10px', background: 'var(--bg-secondary)' }}>
        {open ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
        <span style={{ fontSize: 11, fontWeight: 500, color: 'var(--text-secondary)' }}>{label}</span>
      </div>
      {open && <div style={{ padding: '0 10px 8px' }}>{children}</div>}
    </div>
  );
}

function Tabs({ modes, mode, onChange }: { modes: readonly string[]; mode: string; onChange: (m: string) => void }) {
  return (
    <div style={{ display: 'flex', gap: 2, marginBottom: 4, flexWrap: 'wrap' }}>
      {modes.map((m) => (
        <button key={m} onClick={() => onChange(m)} className={`btn ${mode === m ? 'btn-primary' : ''}`} style={{ fontSize: 11, padding: '1px 8px' }}>{m}</button>
      ))}
    </div>
  );
}

function MatcherMode({ mode, value, onChange, prefix, anyOfItems, onAnyOfChange }: {
  mode: string; value: StubFormData; onChange: (p: any) => void;
  prefix: 'input' | 'headers'; anyOfItems: any[]; onAnyOfChange: (v: any) => void;
}) {
  const key = (k: string) => prefix === 'input'
    ? `input${k.charAt(0).toUpperCase() + k.slice(1)}`
    : `headers${k.charAt(0).toUpperCase() + k.slice(1)}`;

  if (mode === 'anyOf') {
    const modes = prefix === 'headers' ? ['equals', 'contains', 'matches'] : ['equals', 'contains', 'matches', 'glob'];
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        {anyOfItems.length === 0 && <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Add options below.</div>}
        {anyOfItems.map((item, i) => (
          <div key={i} style={{ display: 'flex', gap: 3, alignItems: 'flex-start' }}>
            <select value={item.type} onChange={(e) => { const n = [...anyOfItems]; n[i] = { ...n[i], type: e.target.value }; onAnyOfChange(n); }} className="input" style={{ width: 80, fontSize: 11, marginTop: 3 }}>
              {modes.map((m) => <option key={m} value={m}>{m}</option>)}
            </select>
            <div style={{ flex: 1 }}><MonacoEditor value={item.value} onChange={(v) => { const n = [...anyOfItems]; n[i] = { ...n[i], value: v }; onAnyOfChange(n); }} height={80} /></div>
            {prefix === 'input' && (
              <label style={{ fontSize: 11, display: 'flex', alignItems: 'center', gap: 1, color: 'var(--text-muted)', marginTop: 5, flexShrink: 0 }}><input type="checkbox" checked={item.ignoreArrayOrder} onChange={() => { const n = [...anyOfItems]; n[i] = { ...n[i], ignoreArrayOrder: !item.ignoreArrayOrder }; onAnyOfChange(n); }} /> order</label>
            )}
            <button onClick={() => onAnyOfChange(anyOfItems.filter((_: any, j: number) => j !== i))} className="btn btn-ghost" style={{ padding: '1px 5px', marginTop: 1 }}><X size={11} /></button>
          </div>
        ))}
        <button onClick={() => onAnyOfChange([...anyOfItems, { type: 'equals', value: '{\n  \n}', ignoreArrayOrder: false }])} className="btn btn-ghost" style={{ fontSize: 11, alignSelf: 'flex-start' }}><Plus size={9} /> Add</button>
      </div>
    );
  }

  const val = (value as any)[key(mode)];
  return <MonacoEditor value={val || ''} onChange={(v) => onChange({ [key(mode)]: v })} height={110} />;
}
