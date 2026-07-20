import { useParams, useNavigate } from 'react-router-dom';
import { useStub, useStubs } from '../hooks/useStubs';
import { useHistory } from '../hooks/useHistory';
import { useServices } from '../hooks/useServices';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useToast } from '../components/shared/Toast';
import { api } from '../lib/api';
import { ArrowLeft, Edit3, Copy, Play, Hash, Bug, Trash2, Files, AlertTriangle, Trophy } from 'lucide-react';
import { colors } from '../lib/theme';
import { prettyJson, methodPeers, shadowers, stubRequestExample, streamKind, serviceRefMatches,
  requestMessages, responseMessages, isRequestStream, isResponseStream, matcherEntries, hasContent, MATCHER_COLORS } from '../lib/stub';
import { toYaml } from '../features/stubs/toYaml';
import { stashClone } from '../lib/clone';
import { FileCode } from 'lucide-react';
import { useMemo } from 'react';

export function StubShow() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const toast = useToast();
  const qc = useQueryClient();
  const { data: stub, isLoading, error } = useStub(id!);
  const { data: history } = useHistory();
  const { data: allStubs } = useStubs();
  const { data: svcList } = useServices();

  const del = useMutation({
    mutationFn: async () => { await api.post('/stubs/batchDelete', [id]); },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['stubs'] }); toast.show('Stub deleted'); navigate('/stubs'); },
  });

  const usage = useMemo(() => {
    if (!history || !stub) return null;
    const calls = history.filter((h) => h.stubId === stub.id);
    if (calls.length === 0) return null;
    return {
      total: calls.length,
      first: new Date(calls[calls.length - 1].timestamp),
      last: new Date(calls[0].timestamp),
    };
  }, [history, stub]);

  if (isLoading) return <div style={{ padding: 24, color: 'var(--text-muted)' }}>Loading...</div>;
  if (error || !stub) return <div style={{ padding: 24, color: 'var(--error)' }}>Stub not found.</div>;

  const copyId = () => {
    navigator.clipboard.writeText(stub.id);
    toast.show('Copied to clipboard');
  };

  const out = stub.output;
  const isFile = stub.source === 'file';
  const example = stubRequestExample(stub);
  const inputExample = example.payload;
  const peers = allStubs ? methodPeers(stub, allStubs) : [];
  const shadows = allStubs ? shadowers(stub, allStubs) : [];
  const methodType = svcList?.find((s) => serviceRefMatches(stub.service, s.id, s.name))?.methods?.find((m) => m.name === stub.method)?.methodType;
  const stream = streamKind(methodType);
  // Method-type-aware request/response model.
  const reqMsgs = requestMessages(stub);
  const resMsgs = responseMessages(stub);
  const reqStream = isRequestStream(methodType) || (stub.inputs?.length ?? 0) > 0;
  const resStream = isResponseStream(methodType) || (stub.output?.stream?.length ?? 0) > 0;
  const isError = !!out.error || (out.code ?? 0) > 0;
  const headerEntries = matcherEntries(stub.headers);
  const testHref = `/stubs/test?service=${encodeURIComponent(stub.service)}&method=${encodeURIComponent(stub.method)}&id=${encodeURIComponent(stub.id)}&payload=${encodeURIComponent(example.payload)}&headers=${encodeURIComponent(example.headers)}`;

  return (
    <div className="page-enter" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <button onClick={() => navigate('/stubs')} className="btn btn-ghost" style={{ fontSize: 12 }}><ArrowLeft size={14} /> Back</button>
        <div style={{ flex: 1 }} />
        {!isFile && <button onClick={() => navigate(`/stubs/${stub.id}/edit`)} className="btn"><Edit3 size={13} /> Edit</button>}
        <button onClick={() => { stashClone(stub); navigate('/stubs/create?clone=1'); }} className="btn"><Files size={13} /> Clone</button>
        <button onClick={() => { navigator.clipboard.writeText(toYaml(stub)); toast.show('YAML copied'); }} className="btn" title="Copy stub as YAML"><FileCode size={13} /> YAML</button>
        <button onClick={() => navigate(testHref)} className="btn" title="Send this request and see which stub matches"><Play size={13} /> Test</button>
        <button onClick={() => navigate(`/inspect?service=${encodeURIComponent(stub.service)}&method=${encodeURIComponent(stub.method)}&id=${encodeURIComponent(stub.id)}&payload=${encodeURIComponent(inputExample)}`)} className="btn" title="Diagnose why this stub matches or loses"><Bug size={13} /> Inspect</button>
        {!isFile && <button onClick={() => { if (confirm('Delete this stub?')) del.mutate(); }} className="btn" style={{ color: colors.error }}><Trash2 size={13} /> Delete</button>}
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span className="badge" style={{ background: `${stream.color}1e`, color: stream.color }} title={stream.full}>{stream.label}</span>
        <h1 style={{ fontSize: 16, fontWeight: 600, margin: 0 }}>{stub.service}/{stub.method}</h1>
        <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{stream.full}</span>
      </div>

      <div className="card">
        <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: 6, fontSize: 12 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--text-muted)' }}>
            <Hash size={12} />
            <code style={{ color: 'var(--text)', cursor: 'pointer', fontSize: 11 }} onClick={copyId}>{stub.id}</code>
            <Copy size={11} style={{ cursor: 'pointer', color: 'var(--text-muted)' }} onClick={copyId} />
          </div>
          <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', color: 'var(--text-muted)' }}>
            <span>Priority: <strong style={{ color: 'var(--text)' }}>{stub.priority}</strong></span>
            <span>Times: <strong style={{ color: 'var(--text)' }}>{stub.options?.times ?? '∞'}</strong></span>
            <span>Source: <strong style={{ color: 'var(--text)' }}>{stub.source || '—'}</strong></span>
          </div>
          {usage ? (
            <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', fontSize: 11, color: 'var(--text-muted)', borderTop: '1px solid var(--border)', paddingTop: 6, marginTop: 2 }}>
              <span>Matched <strong style={{ color: colors.success }}>{usage.total}</strong> times</span>
              <span>First: <strong style={{ color: 'var(--text)' }}>{usage.first.toLocaleString()}</strong></span>
              <span>Last: <strong style={{ color: 'var(--text)' }}>{usage.last.toLocaleString()}</strong></span>
            </div>
          ) : (
            <div style={{ fontSize: 11, color: 'var(--text-muted)', borderTop: '1px solid var(--border)', paddingTop: 6, marginTop: 2 }}>
              {stub.used
                ? <span style={{ color: colors.success }}>Matched at least once (details beyond history retention)</span>
                : 'Never matched'}
            </div>
          )}
        </div>
      </div>

      {peers.length > 0 && (
        <div style={{ padding: '10px 12px', borderRadius: 'var(--radius-lg)', border: `1px solid ${shadows.length ? colors.warning + '55' : 'var(--border)'}`, background: shadows.length ? 'var(--warning-bg)' : 'var(--bg-secondary)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 6 }}>
            {shadows.length > 0 && <AlertTriangle size={14} style={{ color: colors.warning }} />}
            <span className="section-title" style={{ color: shadows.length ? colors.warning : 'var(--text-muted)' }}>
              {shadows.length > 0
                ? `Possibly shadowed — ${shadows.length} higher-priority stub${shadows.length > 1 ? 's' : ''} on this method`
                : `${peers.length} other stub${peers.length > 1 ? 's' : ''} on this method`}
            </span>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
            {peers.map((p) => {
              const higher = p.priority > stub.priority;
              return (
                <div key={p.id} onClick={() => navigate(`/stubs/${p.id}`)} className="hover-row"
                  style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12, padding: '3px 6px', borderRadius: 4, cursor: 'pointer' }}>
                  {higher && <Trophy size={12} style={{ color: colors.warning }} />}
                  <code style={{ color: 'var(--text-muted)', fontSize: 11 }}>{p.id.slice(0, 8)}</code>
                  <span className="badge" style={{ background: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}>P{p.priority}</span>
                  <span style={{ color: 'var(--text-muted)', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{higher ? 'evaluated before this' : 'evaluated after this'}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* ── REQUEST ── */}
      <SectionCard
        title={reqStream ? `Request stream · ${reqMsgs.length} message${reqMsgs.length === 1 ? '' : 's'}` : 'Request'}
        hint={reqStream ? 'Client streams these messages; the stub matches them in order.' : (methodType ? stream.full : undefined)}
        color={colors.accent}
      >
        {headerEntries.length > 0 && (
          <div style={{ marginBottom: 10 }}>
            <FieldLabel>Header matcher</FieldLabel>
            <MatcherRules entries={headerEntries} />
          </div>
        )}
        {reqMsgs.length === 0
          ? <Empty>Matches any request payload — no input constraints.</Empty>
          : reqMsgs.map((m, i) => (
              <MessageBlock key={i} index={reqStream ? i + 1 : undefined} label={reqStream ? 'Message' : 'Input matcher'}>
                {matcherEntries(m).length === 0
                  ? <Empty>Matches anything.</Empty>
                  : <MatcherRules entries={matcherEntries(m)} />}
                {m.ignoreArrayOrder && <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 4 }}>· ignores array order</div>}
              </MessageBlock>
            ))}
      </SectionCard>

      {/* ── RESPONSE ── */}
      <SectionCard
        title={isError ? 'Error response' : resStream ? `Response stream · ${resMsgs.length} message${resMsgs.length === 1 ? '' : 's'}` : 'Response'}
        hint={resStream && !isError ? 'Server streams these messages back in order.' : undefined}
        color={isError ? colors.error : '#06b6d4'}
      >
        {isError ? (
          <div style={{ padding: '9px 12px', borderRadius: 'var(--radius)', border: `1px solid ${colors.error}55`, background: 'var(--error-bg)', fontSize: 13 }}>
            <strong style={{ color: colors.error }}>{grpcName(out.code)} </strong>
            <span style={{ color: 'var(--text-muted)' }}>(code {out.code ?? 0})</span>
            {out.error && <div style={{ marginTop: 4, color: 'var(--text)', fontFamily: 'var(--mono)', fontSize: 12 }}>{out.error}</div>}
          </div>
        ) : resMsgs.length === 0 ? (
          <Empty>Empty response.</Empty>
        ) : (
          resMsgs.map((r, i) => (
            <MessageBlock key={i} index={resStream ? i + 1 : undefined} label={resStream ? 'Message' : 'Data'}>
              <pre className="json-block">{prettyJson(r) || JSON.stringify(r, null, 2)}</pre>
            </MessageBlock>
          ))
        )}
        {out.delay && <div style={{ fontSize: 12.5, color: 'var(--text-muted)', marginTop: 8 }}>Delay <code style={{ color: colors.warning, background: 'var(--warning-bg)', padding: '1px 6px', borderRadius: 3 }}>{out.delay}</code></div>}
        {hasContent(out.headers) && <div style={{ marginTop: 8 }}><FieldLabel>Response headers</FieldLabel><pre className="json-block">{prettyJson(out.headers)}</pre></div>}
        {hasContent(out.details) && <div style={{ marginTop: 8 }}><FieldLabel>Error details (protobuf Any)</FieldLabel><pre className="json-block">{prettyJson(out.details)}</pre></div>}
      </SectionCard>

      {stub.effects && stub.effects.length > 0 && (
        <SectionCard title={`Effects · ${stub.effects.length}`} hint="Side effects applied on match." color={colors.warning}>
          <pre className="json-block">{prettyJson(stub.effects)}</pre>
        </SectionCard>
      )}
    </div>
  );
}

const GRPC_NAMES: Record<number, string> = { 0: 'OK', 1: 'Canceled', 2: 'Unknown', 3: 'InvalidArgument', 4: 'DeadlineExceeded', 5: 'NotFound', 6: 'AlreadyExists', 7: 'PermissionDenied', 8: 'ResourceExhausted', 9: 'FailedPrecondition', 10: 'Aborted', 11: 'OutOfRange', 12: 'Unimplemented', 13: 'Internal', 14: 'Unavailable', 15: 'DataLoss', 16: 'Unauthenticated' };
const grpcName = (c?: number) => GRPC_NAMES[c ?? 0] ?? `code ${c}`;

function FieldLabel({ children }: { children: React.ReactNode }) {
  return <div className="section-title" style={{ marginBottom: 4 }}>{children}</div>;
}
function Empty({ children }: { children: React.ReactNode }) {
  return <div style={{ fontSize: 12.5, color: 'var(--text-muted)', fontStyle: 'italic' }}>{children}</div>;
}

function SectionCard({ title, hint, color, children }: { title: string; hint?: string; color: string; children: React.ReactNode }) {
  return (
    <div style={{ borderRadius: 'var(--radius-lg)', border: '1px solid var(--border)', background: 'var(--bg-secondary)', overflow: 'hidden' }}>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, padding: '9px 14px', borderBottom: '1px solid var(--border)', borderLeft: `3px solid ${color}` }}>
        <span style={{ fontSize: 13, fontWeight: 650 }}>{title}</span>
        {hint && <span style={{ fontSize: 11.5, color: 'var(--text-muted)' }}>{hint}</span>}
      </div>
      <div style={{ padding: 12, display: 'flex', flexDirection: 'column', gap: 8 }}>{children}</div>
    </div>
  );
}

function MessageBlock({ index, label, children }: { index?: number; label: string; children: React.ReactNode }) {
  return (
    <div style={{ border: '1px solid var(--border)', borderRadius: 'var(--radius)', background: 'var(--bg)', overflow: 'hidden' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '5px 10px', borderBottom: '1px solid var(--border)', fontSize: 11.5, color: 'var(--text-muted)' }}>
        {index !== undefined && <span style={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'center', minWidth: 18, height: 18, borderRadius: 5, background: 'var(--bg-tertiary)', color: 'var(--text-secondary)', fontWeight: 700, fontSize: 10.5 }}>{index}</span>}
        <span style={{ textTransform: 'uppercase', letterSpacing: 0.4, fontWeight: 650 }}>{label}</span>
      </div>
      <div style={{ padding: 10, display: 'flex', flexDirection: 'column', gap: 6 }}>{children}</div>
    </div>
  );
}

function MatcherRules({ entries }: { entries: { kind: string; value: unknown }[] }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      {entries.map((e, i) => (
        <div key={i}>
          <span style={{ fontSize: 10.5, fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.3, padding: '1px 6px', borderRadius: 4, background: `${MATCHER_COLORS[e.kind] || '#64748b'}1e`, color: MATCHER_COLORS[e.kind] || '#64748b' }}>{e.kind}</span>
          <pre className="json-block" style={{ marginTop: 3 }}>{prettyJson(e.value) || JSON.stringify(e.value, null, 2)}</pre>
        </div>
      ))}
    </div>
  );
}
