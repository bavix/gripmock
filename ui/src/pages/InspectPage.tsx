import { useState, useEffect, useRef, useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useInspect } from '../hooks/useInspect';
import { useStub } from '../hooks/useStubs';
import { useStore } from '../lib/store';
import { MethodSelect } from '../components/shared/MethodSelect';
import { MonacoEditor } from '../components/json/MonacoEditor';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Bug, ArrowRight, CheckCircle2, XCircle, MinusCircle, Eye, Search, RotateCcw, Fingerprint, Crosshair, Trophy, AlertCircle } from 'lucide-react';
import { colors } from '../lib/theme';
import { requestMessages, responseMessages, prettyJson, hasContent, evalMatcherFields, type FieldRule } from '../lib/stub';
import type { InspectReport, InspectCandidate, InspectStage, Stub } from '../lib/types';

function jsonErr(s: string): string | null {
  const t = (s ?? '').trim();
  if (!t) return null;
  try { JSON.parse(t); return null; } catch (e) { return (e as Error).message; }
}

function parseObject(text: string): unknown {
  try { return JSON.parse(text || '{}'); } catch { return {}; }
}

function mismatches(input: Record<string, unknown>, matcher?: Parameters<typeof evalMatcherFields>[1]): FieldRule[] | null {
  const rows = evalMatcherFields(input, matcher).filter((r) => !r.ok);
  return rows.length ? rows : null;
}

const STAGE_LABEL: Record<string, string> = {
  id: 'Stub ID lookup',
  service_method: 'Service / method',
  fallback_method: 'Method fallback',
  session: 'Session scope',
  times: 'Times limit',
  headers: 'Header matcher',
  input: 'Input matcher',
  selected: 'Selection',
};

const REASON_TEXT: Record<string, string> = {
  id: 'stub ID did not match',
  service: 'different service',
  method: 'different method',
  session: 'not visible in this session',
  times: 'call limit exhausted',
  headers: 'header matcher did not match',
  input: 'input matcher did not match',
  id_lookup: 'skipped — pinned by stub ID',
  not_selected: 'another stub was selected',
};
const reasonText = (r: string) => REASON_TEXT[r] ?? r;

// Selection order: matched wins; otherwise higher priority, then specificity, then score.
function rankValue(c: InspectCandidate): number[] {
  return [c.matched ? 1 : 0, (c.excludedBy?.length ?? 0) === 0 ? 1 : 0, c.priority, c.specificity, c.score];
}
function byRank(a: InspectCandidate, b: InspectCandidate): number {
  const ra = rankValue(a), rb = rankValue(b);
  for (let i = 0; i < ra.length; i++) if (rb[i] !== ra[i]) return rb[i] - ra[i];
  return 0;
}

export function InspectPage() {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const session = useStore((s) => s.session);
  const inspect = useInspect();
  const [service, setService] = useState(params.get('service') || '');
  const [method, setMethod] = useState(params.get('method') || '');
  const [target, setTarget] = useState(params.get('id') || '');
  const [payload, setPayload] = useState(params.get('payload') || '{\n  "name": "John",\n  "age": 30\n}');
  const [headers, setHeaders] = useState(params.get('headers') || '{\n  "authorization": "Bearer eyJ..."\n}');
  const [result, setResult] = useState<InspectReport | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [showAll, setShowAll] = useState(false);
  const [candFilter, setCandFilter] = useState<'all' | 'qualified' | 'excluded'>('all');
  const autoRan = useRef(false);
  const payloadErr = useMemo(() => jsonErr(payload), [payload]);
  const headersErr = useMemo(() => jsonErr(headers), [headers]);

  const handleInspect = async () => {
    setResult(null); setSelectedId(null); setShowAll(false);
    let p: Record<string, unknown>[] | undefined;
    let h: Record<string, string> | undefined;
    try { const x = JSON.parse(payload); p = Array.isArray(x) ? x : [x]; } catch {}
    try { h = JSON.parse(headers); } catch {}

    // NOTE: we intentionally do NOT send the target id — we want the full
    // candidate set (all stubs on the method) so we can explain ranking.
    const body: Record<string, unknown> = { service, method, headers: h, input: p };
    if (session) body.session = session;

    try {
      const res = await inspect.mutateAsync(body as any);
      setResult(res);
      const tgt = target && res.candidates?.find((c) => c.id === target || c.id.startsWith(target));
      setSelectedId(tgt ? tgt.id : (res.matchedStubId ?? null));
    } catch (err) {
      setResult({ service, method, error: (err as Error).message, candidates: [], stages: [] });
    }
  };

  useEffect(() => {
    if (autoRan.current) return;
    if (params.get('service') && params.get('method')) { autoRan.current = true; handleInspect(); }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const ranked = useMemo(() => [...(result?.candidates ?? [])].sort(byRank), [result]);
  const winner = useMemo(() => ranked.find((c) => c.matched) ?? null, [ranked]);
  const shown = useMemo(() => ranked.filter((c) => {
    if (candFilter === 'qualified') return (c.excludedBy?.length ?? 0) === 0;
    if (candFilter === 'excluded') return (c.excludedBy?.length ?? 0) > 0;
    return true;
  }), [ranked, candFilter]);
  const visible = showAll ? shown : shown.slice(0, 15);
  const selected = ranked.find((c) => c.id === selectedId) ?? null;
  const targetId = target ? (ranked.find((c) => c.id === target || c.id.startsWith(target))?.id ?? null) : null;
  // Fetch the selected stub by id directly — the candidate list can be large
  // and server-paginated, so we can't rely on a preloaded full-stub set.
  const { data: selectedStub } = useStub(selected?.id ?? '');

  return (
    <div className="page-enter" style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <h1 style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <Bug size={18} color={colors.accent} /> Inspect Matching
      </h1>
      <p style={{ fontSize: 12.5, color: 'var(--text-muted)', margin: 0, marginTop: -4 }}>
        Simulate a request and see exactly which stub is selected and why. Set a target stub to diagnose why it does or doesn't win.
      </p>

      <Card>
        <div className="card-header">Request</div>
        <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <MethodSelect service={service} method={method} onServiceChange={(s) => { setService(s); setMethod(''); }} onMethodChange={setMethod} />
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Crosshair size={14} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
            <input value={target} onChange={(e) => setTarget(e.target.value)}
              placeholder="Target stub ID (optional — diagnose why this stub wins or loses)"
              className="input" style={{ flex: 1, fontFamily: 'var(--mono)', fontSize: 12 }} />
            {session && (
              <span className="chip" style={{ background: 'var(--accent-bg)', color: 'var(--accent-text)' }}>
                <Fingerprint size={11} /> {session.slice(0, 10)}
              </span>
            )}
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
            <div>
              <MonacoEditor value={payload} onChange={setPayload} height={110} label="Payload" />
              {payloadErr && <div style={jsonErrStyle}><AlertCircle size={11} /> {payloadErr}</div>}
            </div>
            <div>
              <MonacoEditor value={headers} onChange={setHeaders} height={110} label="Headers" />
              {headersErr && <div style={jsonErrStyle}><AlertCircle size={11} /> {headersErr}</div>}
            </div>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <Button variant="primary" onClick={handleInspect} disabled={!service || !method || inspect.isPending || !!payloadErr || !!headersErr}>
              <Search size={14} /> {inspect.isPending ? 'Inspecting…' : 'Inspect'}
            </Button>
            <Button onClick={() => { setPayload('{\n  "name": "John",\n  "age": 30\n}'); setHeaders('{\n  "authorization": "Bearer eyJ..."\n}'); setResult(null); }}>
              <RotateCcw size={13} /> Reset example
            </Button>
            <Button onClick={() => { setPayload('{}'); setHeaders('{}'); setTarget(''); setResult(null); }}>Clear</Button>
          </div>
        </div>
      </Card>

      {inspect.isError && (
        <div style={banner('var(--error)', colors.error)}>{(inspect.error as Error).message}</div>
      )}

      {result && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {result.error && <div style={banner('var(--warning)', colors.warning)}>{result.error}</div>}

          <ResultHeader result={result} />
          {result.stages && result.stages.length > 0 && <PipelineFlow stages={result.stages} />}

          <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1.1fr)', gap: 10, alignItems: 'start' }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <div className="section-title" style={{ flex: 1 }}>Candidates ({shown.length}{shown.length !== ranked.length ? ` / ${ranked.length}` : ''})</div>
                {ranked.length > 1 && (
                  <div className="tabs">
                    {(['all', 'qualified', 'excluded'] as const).map((k) => (
                      <button key={k} className={`tab ${candFilter === k ? 'active' : ''}`} onClick={() => setCandFilter(k)}>{k}</button>
                    ))}
                  </div>
                )}
              </div>
              {shown.length === 0 && <div className="empty" style={{ padding: 20 }}>No {candFilter !== 'all' ? candFilter : ''} candidates.</div>}
              {visible.map((c) => (
                <CandidateRow key={c.id} rank={ranked.indexOf(c) + 1} candidate={c} selected={selectedId === c.id}
                  isTarget={c.id === targetId} onSelect={() => setSelectedId(c.id)} />
              ))}
              {shown.length > 15 && (
                <Button className="btn-sm" onClick={() => setShowAll(!showAll)} style={{ alignSelf: 'flex-start' }}>
                  {showAll ? 'Show less' : `Show all ${shown.length}`}
                </Button>
              )}
            </div>

            {selected
              ? <Diagnosis candidate={selected} winner={winner} isTarget={selected.id === targetId} navigate={navigate} stub={selectedStub && selectedStub.id === selected.id ? selectedStub : undefined} payloadText={payload} headersText={headers} />
              : <Card><div className="card-body" style={{ color: 'var(--text-muted)', fontSize: 13 }}>Select a candidate to see the criterion-by-criterion diagnosis.</div></Card>}
          </div>
        </div>
      )}
    </div>
  );
}

const jsonErrStyle: React.CSSProperties = { display: 'flex', alignItems: 'center', gap: 4, marginTop: 3, fontSize: 11, color: colors.error };

function banner(border: string, color: string): React.CSSProperties {
  return { padding: 10, borderRadius: 'var(--radius)', border: `1px solid ${border}`, background: `${color}12`, color, fontSize: 12.5 };
}

function ResultHeader({ result }: { result: InspectReport }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '11px 14px', borderRadius: 'var(--radius-lg)', border: '1px solid var(--border)', background: 'var(--bg-secondary)' }}>
      {result.matchedStubId ? (
        <span style={{ display: 'flex', alignItems: 'center', gap: 7, fontSize: 14, color: colors.success, fontWeight: 650 }}>
          <CheckCircle2 size={18} /> Matched
          <code style={{ fontSize: 11.5, background: 'var(--success-bg)', padding: '2px 8px', borderRadius: 5 }}>{result.matchedStubId.slice(0, 12)}</code>
        </span>
      ) : (
        <span style={{ display: 'flex', alignItems: 'center', gap: 7, fontSize: 14, color: colors.warning, fontWeight: 650 }}>
          <XCircle size={18} /> No stub matched
        </span>
      )}
      {result.similarStubId && (
        <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Nearest: <code>{result.similarStubId.slice(0, 12)}</code></span>
      )}
      {result.fallbackToMethod && <span className="chip" style={{ background: 'var(--accent-bg)', color: 'var(--accent-text)' }}>method fallback</span>}
      <div style={{ flex: 1 }} />
      <span style={{ fontSize: 12, color: 'var(--text-muted)', fontFamily: 'var(--mono)' }}>{result.service}/{result.method}</span>
    </div>
  );
}

function PipelineFlow({ stages }: { stages: InspectStage[] }) {
  return (
    <div style={{ padding: 12, borderRadius: 'var(--radius-lg)', border: '1px solid var(--border)', background: 'var(--bg-secondary)' }}>
      <div className="section-title" style={{ marginBottom: 10 }}>Filter pipeline</div>
      <div style={{ display: 'flex', alignItems: 'center', overflowX: 'auto' }}>
        {stages.map((s, i) => (
          <div key={i} style={{ display: 'flex', alignItems: 'center', flexShrink: 0 }}>
            <div style={{
              display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '6px 12px', borderRadius: 'var(--radius)', minWidth: 58,
              background: s.removed > 0 ? 'var(--warning-bg)' : 'var(--success-bg)',
              border: `1px solid ${s.removed > 0 ? colors.warning : colors.success}30`,
            }}>
              <span style={{ fontSize: 16, fontWeight: 700 }}>{s.after}</span>
              <span style={{ fontSize: 9.5, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: 0.3, marginTop: 1 }}>{STAGE_LABEL[s.name] ?? s.name}</span>
              {s.removed > 0 && <span style={{ fontSize: 10, color: colors.warning, fontWeight: 600 }}>−{s.removed}</span>}
            </div>
            {i < stages.length - 1 && <ArrowRight size={13} style={{ color: 'var(--text-muted)', margin: '0 3px', flexShrink: 0 }} />}
          </div>
        ))}
      </div>
    </div>
  );
}

function CandidateRow({ rank, candidate, selected, isTarget, onSelect }: { rank: number; candidate: InspectCandidate; selected: boolean; isTarget: boolean; onSelect: () => void }) {
  const excluded = (candidate.excludedBy?.length ?? 0) > 0;
  return (
    <div onClick={onSelect} role="button" tabIndex={0} onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onSelect(); } }} className="hover-row" data-candidate-id={candidate.id}
      style={{
        display: 'flex', alignItems: 'center', gap: 8, padding: '8px 10px', borderRadius: 'var(--radius)', cursor: 'pointer', fontSize: 12,
        border: `1px solid ${selected ? 'var(--accent)' : isTarget ? colors.warning : 'var(--border)'}`,
        background: selected ? 'var(--accent-bg)' : 'var(--bg-secondary)',
      }}>
      <span style={{ fontSize: 11, fontWeight: 700, color: 'var(--text-muted)', width: 16, textAlign: 'right' }}>{rank}</span>
      {candidate.matched && <Trophy size={13} style={{ color: colors.success, flexShrink: 0 }} />}
      {isTarget && !candidate.matched && <Crosshair size={13} style={{ color: colors.warning, flexShrink: 0 }} />}
      <code style={{ fontSize: 11, color: 'var(--text-muted)', flexShrink: 0 }}>{candidate.id.slice(0, 8)}</code>
      <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{candidate.method}</span>
      <span className="badge" style={{ background: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }} title="priority">P{candidate.priority}</span>
      <span className="badge" style={{ background: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }} title="specificity">S{candidate.specificity}</span>
      {candidate.matched
        ? <span className="chip" style={{ background: 'var(--success-bg)', color: colors.success }}>selected</span>
        : excluded
        ? <span className="chip" style={{ background: 'var(--error-bg)', color: colors.error }}>✕ {candidate.excludedBy!.join(', ')}</span>
        : <span className="chip" style={{ background: 'var(--warning-bg)', color: colors.warning }}>outranked</span>}
    </div>
  );
}

function Diagnosis({ candidate, winner, isTarget, navigate, stub, payloadText, headersText }: { candidate: InspectCandidate; winner: InspectCandidate | null; isTarget: boolean; navigate: (p: string) => void; stub?: Stub; payloadText: string; headersText: string }) {
  const excluded = (candidate.excludedBy?.length ?? 0) > 0;
  const outranked = !candidate.matched && !excluded;

  const headerDiff = useMemo(() => {
    if (candidate.headersMatched || !stub || !hasContent(stub.headers)) return null;
    return mismatches(parseObject(headersText) as Record<string, unknown>, stub.headers as Record<string, unknown>);
  }, [candidate.headersMatched, stub, headersText]);

  // Streaming stubs keep the matcher in inputs[] (input is null) — diff msg #1.
  const diff = useMemo(() => {
    if (candidate.inputMatched || !stub) return null;
    const parsed = parseObject(payloadText);
    const payload = (Array.isArray(parsed) ? (parsed[0] ?? {}) : parsed) as Record<string, unknown>;
    const matcher = hasContent(stub.input) ? stub.input : requestMessages(stub)[0];
    return mismatches(payload, matcher);
  }, [candidate.inputMatched, stub, payloadText]);

  // Prefer the authoritative per-stage events (3 states: passed/failed/skipped);
  // fall back to the boolean checks when events are absent.
  type Check = { label: string; state: 'passed' | 'failed' | 'skipped'; reason: string };
  const checks: Check[] = candidate.events && candidate.events.length > 0
    ? candidate.events.map((e) => ({
        label: STAGE_LABEL[e.stage] ?? e.stage,
        state: e.result === 'passed' ? 'passed' : e.result === 'skipped' ? 'skipped' : 'failed',
        reason: e.reason && e.result !== 'passed' ? reasonText(e.reason) : '',
      }))
    : ([
        { label: 'Session scope', state: candidate.visibleBySession ? 'passed' : 'failed', reason: '' },
        { label: 'Times limit', state: candidate.withinTimes ? 'passed' : 'failed', reason: '' },
        { label: 'Header matcher', state: candidate.headersMatched ? 'passed' : 'failed', reason: '' },
        { label: 'Input matcher', state: candidate.inputMatched ? 'passed' : 'failed', reason: '' },
      ] as Check[]);

  const verdict = candidate.matched
    ? { color: colors.success, text: 'Selected — this stub served the response.' }
    : excluded
    ? { color: colors.error, text: `Excluded at "${candidate.excludedBy!.join(', ')}" — did not qualify.` }
    : { color: colors.warning, text: 'Qualified, but another stub was selected.' };

  return (
    <Card style={{ borderColor: candidate.matched ? colors.success : isTarget ? colors.warning : 'var(--border)' }}>
      <div className="card-header" style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        {isTarget && <Crosshair size={12} style={{ color: colors.warning }} />}
        Diagnosis
      </div>
      <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: 10, fontSize: 12.5 }}>
        <div>
          <div style={{ fontWeight: 600 }}>{candidate.service}/{candidate.method}</div>
          <code style={{ fontSize: 11, color: 'var(--text-muted)' }}>{candidate.id}</code>
        </div>

        <div style={{ fontWeight: 600, color: verdict.color }}>{verdict.text}</div>

        <div style={{ display: 'flex', gap: 14, flexWrap: 'wrap', color: 'var(--text-muted)', fontSize: 11.5 }}>
          <span>priority <strong style={{ color: 'var(--text)' }}>{candidate.priority}</strong></span>
          <span>specificity <strong style={{ color: 'var(--text)' }}>{candidate.specificity}</strong></span>
          <span>score <strong style={{ color: 'var(--text)' }}>{candidate.score.toFixed(3)}</strong></span>
          <span>used <strong style={{ color: 'var(--text)' }}>{candidate.used}/{candidate.times || '∞'}</strong></span>
        </div>

        <div>
          <div className="section-title" style={{ marginBottom: 4 }}>Criteria</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
            {checks.map((c, i) => (
              <div key={i} style={{ display: 'flex', alignItems: 'flex-start', gap: 6, fontSize: 12 }}>
                {c.state === 'passed'
                  ? <CheckCircle2 size={13} color={colors.success} style={{ flexShrink: 0, marginTop: 1 }} />
                  : c.state === 'skipped'
                  ? <MinusCircle size={13} color={'var(--text-muted)'} style={{ flexShrink: 0, marginTop: 1 }} />
                  : <XCircle size={13} color={colors.error} style={{ flexShrink: 0, marginTop: 1 }} />}
                <span style={{ color: c.state === 'passed' ? 'var(--text)' : 'var(--text-muted)', minWidth: 120 }}>{c.label}</span>
                {c.reason && <span style={{ color: c.state === 'skipped' ? 'var(--text-muted)' : colors.error, fontSize: 11.5 }}>{c.reason}</span>}
              </div>
            ))}
          </div>
        </div>

        {headerDiff && <FieldMismatchList title="Header field mismatches" rows={headerDiff} />}
        {diff && <FieldMismatchList title="Input field mismatches" rows={diff} />}

        <StubDefinition stub={stub} candidate={candidate} />

        {outranked && winner && winner.id !== candidate.id && (
          <div style={{ padding: 10, borderRadius: 'var(--radius)', background: 'var(--warning-bg)', border: `1px solid ${colors.warning}40` }}>
            <div style={{ fontWeight: 600, color: colors.warning, marginBottom: 4, display: 'flex', alignItems: 'center', gap: 5 }}>
              <Trophy size={13} /> Lost to {winner.id.slice(0, 8)}
            </div>
            <div style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{whyLost(candidate, winner)}</div>
            <div style={{ display: 'flex', gap: 16, marginTop: 6, fontSize: 11.5 }}>
              <Compare label="priority" a={candidate.priority} b={winner.priority} />
              <Compare label="specificity" a={candidate.specificity} b={winner.specificity} />
              <Compare label="score" a={+candidate.score.toFixed(3)} b={+winner.score.toFixed(3)} />
            </div>
          </div>
        )}

        <Button className="btn-sm" onClick={() => navigate(`/stubs/${candidate.id}`)} style={{ alignSelf: 'flex-start' }}>
          <Eye size={12} /> Open stub
        </Button>
      </div>
    </Card>
  );
}

function FieldMismatchList({ title, rows }: { title: string; rows: FieldRule[] }) {
  return (
    <div>
      <div className="section-title" style={{ marginBottom: 4 }}>{title}</div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
        {rows.map((r) => (
          <div key={`${r.kind}:${r.field}`} style={{ display: 'flex', alignItems: 'baseline', gap: 6, fontSize: 11.5, fontFamily: 'var(--mono)', flexWrap: 'wrap' }}>
            <XCircle size={12} color={colors.error} style={{ alignSelf: 'center', flexShrink: 0 }} />
            <span style={{ color: 'var(--text)', minWidth: 90 }}>{r.field}</span>
            <span className="badge" style={{ background: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }} title="matcher kind">{r.kind}</span>
            <code style={{ color: colors.success }}>{JSON.stringify(r.expected)}</code>
            <span style={{ color: 'var(--text-muted)' }}>got</span>
            <code style={{ color: colors.error }}>{r.actual === undefined ? '(absent)' : JSON.stringify(r.actual)}</code>
          </div>
        ))}
      </div>
    </div>
  );
}

function StubDefinition({ stub, candidate }: { stub?: Stub; candidate: InspectCandidate }) {
  if (!stub) {
    return (
      <div>
        <div className="section-title" style={{ marginBottom: 4 }}>Stub definition</div>
        <div style={{ fontSize: 11.5, color: 'var(--text-muted)', fontStyle: 'italic' }}>
          Loading stub <code>{candidate.id.slice(0, 8)}</code>…
        </div>
      </div>
    );
  }

  const reqMsgs = requestMessages(stub);
  const resMsgs = responseMessages(stub);
  const reqStream = reqMsgs.length > 1;
  const resStream = (stub.output?.stream?.length ?? 0) > 0;
  const isErr = !!stub.output?.error || (stub.output?.code ?? 0) > 0;
  const headersJson = hasContent(stub.headers) ? prettyJson(stub.headers) : '';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      <div className="section-title">Stub definition</div>

      {headersJson && (
        <>
          <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Headers matcher</div>
          <pre className="json-block">{headersJson}</pre>
        </>
      )}

      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>
        {reqStream ? `Request stream · ${reqMsgs.length} msgs` : 'Request matcher'}
      </div>
      {reqMsgs.length === 0
        ? <div style={{ fontSize: 11, color: 'var(--text-muted)', fontStyle: 'italic' }}>any request</div>
        : reqMsgs.map((m, i) => <pre key={i} className="json-block">{reqStream ? `# ${i + 1}\n` : ''}{prettyJson(m) || '{}'}</pre>)}

      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>
        {isErr ? 'Error response' : resStream ? `Response stream · ${resMsgs.length} msgs` : 'Response'}
      </div>
      {isErr
        ? <pre className="json-block" style={{ color: colors.error }}>{`code ${stub.output?.code ?? 0}${stub.output?.error ? '\n' + stub.output.error : ''}`}</pre>
        : resMsgs.length === 0
          ? <div style={{ fontSize: 11, color: 'var(--text-muted)', fontStyle: 'italic' }}>empty</div>
          : resMsgs.map((rm, i) => <pre key={i} className="json-block">{resStream ? `# ${i + 1}\n` : ''}{prettyJson(rm) || '{}'}</pre>)}
    </div>
  );
}

function whyLost(c: InspectCandidate, w: InspectCandidate): string {
  if (w.priority !== c.priority) return `Winner has higher priority (${w.priority} > ${c.priority}). Priority is compared first.`;
  if (w.specificity !== c.specificity) return `Equal priority, but winner is more specific (${w.specificity} > ${c.specificity} matched fields).`;
  if (w.score !== c.score) return `Equal priority and specificity, but winner scored higher (${w.score.toFixed(3)} > ${c.score.toFixed(3)}).`;
  return 'Tie-breaker resolved in the winner\'s favour (insertion order).';
}

function Compare({ label, a, b }: { label: string; a: number; b: number }) {
  const win = b >= a;
  return (
    <span style={{ color: 'var(--text-muted)' }}>
      {label}: <strong style={{ color: a >= b ? colors.success : 'var(--text)' }}>{a}</strong>
      <span style={{ margin: '0 3px' }}>vs</span>
      <strong style={{ color: win ? colors.success : 'var(--text)' }}>{b}</strong>
    </span>
  );
}
