import { useState, useCallback, useMemo, useEffect, useRef, type ReactNode } from 'react';
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
import { parse } from './buildStubOutput';
import {
  type StubFormData, INPUT_MODES, HEADER_MODES, GRPC_CODES,
  empty, fromInit, buildBody, collectJsonErrors,
} from './buildStubBody';

/* ── Types ── */

type Props = Readonly<{ initial?: Record<string, unknown>; onSaved?: () => void }>;

// Stable React keys for id-less editable rows (UI-only; never sent to the API).
let rowKeySeed = 0;
const nextRowKey = () => `row-${rowKeySeed++}`;

// Preview pane body — extracted so the JSX stays a flat conditional (no nested ternary).
function renderPreview(busy: boolean, error: string | null, yaml: string | null): ReactNode {
  if (busy) {
    return <span style={{ display: 'flex', alignItems: 'center', gap: 5, color: 'var(--text-muted)', fontSize: 11 }}><Loader2 size={11} className="animate-spin" /> Validating…</span>;
  }
  if (error) {
    return <span style={{ display: 'flex', alignItems: 'flex-start', gap: 5, color: 'var(--error)', fontSize: 11 }}><AlertCircle size={12} style={{ flexShrink: 0, marginTop: 1 }} /><span>{error}</span></span>;
  }
  if (yaml) {
    return <span>{highlightYaml(yaml)}</span>;
  }
  return <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>Fill in service/method to see preview</span>;
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
    const h = (e: BeforeUnloadEvent) => { if (dirty) e.preventDefault(); };
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
  const jsonErrors = useMemo(() => collectJsonErrors(f), [f]);

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

  const altCount = f.inputsAlt.length ? ` (${f.inputsAlt.length})` : '';
  const sequenceLabel = isReqStreamMethod ? `Request message sequence${altCount}` : `Alternative matchers${altCount}`;
  const previewBody = renderPreview(validBusy, validErr, yaml);

  /* ── Render ── */

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 420px', gap: 16, alignItems: 'start' }}>
      {/* ── Left: form ── */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {/* Header */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, paddingBottom: 8, borderBottom: '1px solid var(--border)' }}>
          <button type="button" onClick={leave} className="btn btn-ghost" style={{ fontSize: 11 }}><ArrowLeft size={13} /> Back</button>
          <span style={{ fontSize: 14, fontWeight: 600, flex: 1 }}>{initId ? 'Edit Stub' : 'Create Stub'}</span>
          <button type="button" onClick={handleSubmit} disabled={sub || !f.service || !f.method || jsonErrors.length > 0} className="btn btn-primary" style={{ fontSize: 11 }}>
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
              <button type="button" onClick={handleGenerate} className="btn" style={{ fontSize: 11, padding: '2px 7px' }}><Sparkles size={10} /> Generate</button>
            )}
          </div>
          <Tabs modes={INPUT_MODES} mode={inpMode} onChange={setInpMode} />
          <MatcherMode mode={inpMode} value={f} onChange={patch} prefix="input" anyOfItems={f.inputAnyOf} onAnyOfChange={(v) => patch({ inputAnyOf: v })} />
        </Section>

        {/* Response */}
        <Section label="Response">
          <div style={{ display: 'flex', gap: 3, marginBottom: 4, flexWrap: 'wrap' }}>
            {['data', 'stream'].map((m) => (
              <button type="button" key={m} onClick={() => setOutMode(m as any)} className={`btn ${outMode === m ? 'btn-primary' : ''}`} style={{ fontSize: 11, padding: '2px 8px' }}>{m}</button>
            ))}
            <div style={{ flex: 1 }} />
            {methodSchema?.responseSchema && (
              <button type="button" onClick={handleGenerateResponse} className="btn" style={{ fontSize: 11, padding: '2px 8px' }}><Sparkles size={10} /> Generate</button>
            )}
            <button type="button" onClick={() => patch({ outputError: f.outputError ? '' : 'error' })} className={`btn ${f.outputError ? 'btn-danger' : ''}`} style={{ fontSize: 11, padding: '2px 8px' }}>Error</button>
            <button type="button" onClick={() => patch({ outputDelay: f.outputDelay ? '' : '500ms' })} className="btn" style={{ fontSize: 11, padding: '2px 8px' }}>Delay</button>
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
          label={sequenceLabel}
          defaultOpen={isReqStreamMethod && f.inputsAlt.length > 0}
        >
          <MessageSequenceEditor items={f.inputsAlt} onChange={(v) => patch({ inputsAlt: v })} streaming={isReqStreamMethod} />
        </Collapse>

        <Collapse label="Effects">
          <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4 }}>Side effects triggered on match.</div>
          {f.effects.length === 0 && <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4 }}>None.</div>}
          {f.effects.map((e, i) => (
            <div key={e._k} style={{ padding: 8, borderRadius: 5, border: '1px solid var(--border)', marginBottom: 4, position: 'relative' }}>
              <button type="button" onClick={() => patch({ effects: f.effects.filter((_, j) => j !== i) })} className="btn btn-ghost" style={{ position: 'absolute', top: 4, right: 4, padding: '1px 5px' }}><X size={11} /></button>
              <select value={e.action} onChange={(ev) => { const n = [...f.effects]; n[i] = { ...n[i], action: ev.target.value as any }; patch({ effects: n }); }} className="input" style={{ fontSize: 11, width: 90, marginBottom: 3 }}>
                <option value="upsert">Upsert</option><option value="delete">Delete</option>
              </select>
              {e.action === 'delete' && <input value={e.id || ''} onChange={(ev) => { const n = [...f.effects]; n[i] = { ...n[i], id: ev.target.value }; patch({ effects: n }); }} placeholder="Stub UUID to delete" className="input" style={{ fontFamily: 'monospace', marginTop: 2, fontSize: 11 }} />}
              {e.action === 'upsert' && <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 1 }}>Full stub JSON upserted on match.<MonacoEditor value={e.stub || ''} onChange={(v) => { const n = [...f.effects]; n[i] = { ...n[i], stub: v }; patch({ effects: n }); }} height={70} /></div>}
            </div>
          ))}
          <button type="button" onClick={() => patch({ effects: [...f.effects, { action: 'upsert', _k: nextRowKey() }] })} className="btn btn-ghost" style={{ fontSize: 11 }}><Plus size={9} /> Add</button>
        </Collapse>

        {/* Bottom save */}
        <button type="button" onClick={handleSubmit} disabled={sub || !f.service || !f.method || jsonErrors.length > 0} className="btn btn-primary" style={{ alignSelf: 'flex-start', fontSize: 11 }}>
          <Save size={12} /> {sub ? 'Saving…' : 'Save Stub'}
        </button>
      </div>

      {/* ── Right: preview ── */}
      <div style={{ position: 'sticky', top: 0 }}>
        <div style={{ borderRadius: 6, border: '1px solid var(--border)', overflow: 'hidden', background: 'var(--bg)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '6px 10px', borderBottom: '1px solid var(--border)', background: 'var(--bg-secondary)', fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.3px' }}>
            <span style={{ flex: 1 }}>Preview</span>
            {yaml && <button type="button" onClick={() => navigator.clipboard.writeText(yaml)} className="btn btn-ghost" style={{ fontSize: 11, padding: '1px 5px' }}><Copy size={9} /></button>}
          </div>
          <pre style={{ margin: 0, padding: 10, fontSize: 11, lineHeight: 1.5, fontFamily: 'var(--mono)', overflow: 'auto', maxHeight: 'calc(100vh - 140px)', background: 'var(--bg)', whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
            {previewBody}
          </pre>
        </div>
        {f.service && f.method && (
          <div style={{ marginTop: 6, display: 'flex', gap: 4 }}>
            <button type="button" onClick={() => {
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

function Section({ label, children }: Readonly<{ label?: string; children: React.ReactNode }>) {
  return (
    <div style={{ borderRadius: 6, border: '1px solid var(--border)', background: 'var(--bg-secondary)' }}>
      {label && <div style={{ padding: '6px 10px', borderBottom: '1px solid var(--border)', fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.3px' }}>{label}</div>}
      <div style={{ padding: 10 }}>{children}</div>
    </div>
  );
}

function Label({ children }: Readonly<{ children: React.ReactNode }>) {
  return <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.2px', marginBottom: 1 }}>{children}</div>;
}

const lbl: React.CSSProperties = { fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.2px' };

function Collapse({ label, children, defaultOpen = false }: Readonly<{ label: string; children: React.ReactNode; defaultOpen?: boolean }>) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div style={{ borderRadius: 6, border: '1px solid var(--border)', background: open ? 'var(--bg-secondary)' : 'transparent', overflow: 'hidden' }}>
      <button type="button" onClick={() => setOpen(!open)} style={{ display: 'flex', alignItems: 'center', gap: 5, cursor: 'pointer', userSelect: 'none', padding: '6px 10px', background: 'none', border: 'none', font: 'inherit', color: 'inherit', textAlign: 'left', width: '100%' }}>
        {open ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
        <span style={{ fontSize: 11, fontWeight: 500, color: 'var(--text-secondary)' }}>{label}</span>
      </button>
      {open && <div style={{ padding: '0 10px 8px' }}>{children}</div>}
    </div>
  );
}

function Tabs({ modes, mode, onChange }: Readonly<{ modes: readonly string[]; mode: string; onChange: (m: string) => void }>) {
  return (
    <div style={{ display: 'flex', gap: 2, marginBottom: 4, flexWrap: 'wrap' }}>
      {modes.map((m) => (
        <button type="button" key={m} onClick={() => onChange(m)} className={`btn ${mode === m ? 'btn-primary' : ''}`} style={{ fontSize: 11, padding: '1px 8px' }}>{m}</button>
      ))}
    </div>
  );
}

function MatcherMode({ mode, value, onChange, prefix, anyOfItems, onAnyOfChange }: Readonly<{
  mode: string; value: StubFormData; onChange: (p: any) => void;
  prefix: 'input' | 'headers'; anyOfItems: any[]; onAnyOfChange: (v: any) => void;
}>) {
  const key = (k: string) => prefix === 'input'
    ? `input${k.charAt(0).toUpperCase() + k.slice(1)}`
    : `headers${k.charAt(0).toUpperCase() + k.slice(1)}`;

  if (mode === 'anyOf') {
    const modes = prefix === 'headers' ? ['equals', 'contains', 'matches'] : ['equals', 'contains', 'matches', 'glob'];
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        {anyOfItems.length === 0 && <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Add options below.</div>}
        {anyOfItems.map((item, i) => (
          <div key={item._k} style={{ display: 'flex', gap: 3, alignItems: 'flex-start' }}>
            <select value={item.type} onChange={(e) => { const n = [...anyOfItems]; n[i] = { ...n[i], type: e.target.value }; onAnyOfChange(n); }} className="input" style={{ width: 80, fontSize: 11, marginTop: 3 }}>
              {modes.map((m) => <option key={m} value={m}>{m}</option>)}
            </select>
            <div style={{ flex: 1 }}><MonacoEditor value={item.value} onChange={(v) => { const n = [...anyOfItems]; n[i] = { ...n[i], value: v }; onAnyOfChange(n); }} height={80} /></div>
            {prefix === 'input' && (
              <label style={{ fontSize: 11, display: 'flex', alignItems: 'center', gap: 1, color: 'var(--text-muted)', marginTop: 5, flexShrink: 0 }}><input type="checkbox" checked={item.ignoreArrayOrder} onChange={() => { const n = [...anyOfItems]; n[i] = { ...n[i], ignoreArrayOrder: !item.ignoreArrayOrder }; onAnyOfChange(n); }} /> order</label>
            )}
            <button type="button" onClick={() => onAnyOfChange(anyOfItems.filter((_: any, j: number) => j !== i))} className="btn btn-ghost" style={{ padding: '1px 5px', marginTop: 1 }}><X size={11} /></button>
          </div>
        ))}
        <button type="button" onClick={() => onAnyOfChange([...anyOfItems, { type: 'equals', value: '{\n  \n}', ignoreArrayOrder: false, _k: nextRowKey() }])} className="btn btn-ghost" style={{ fontSize: 11, alignSelf: 'flex-start' }}><Plus size={9} /> Add</button>
      </div>
    );
  }

  const val = (value as any)[key(mode)];
  return <MonacoEditor value={val || ''} onChange={(v) => onChange({ [key(mode)]: v })} height={110} />;
}
