import { useState, useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useInfiniteHistory } from '../hooks/useHistory';
import { useStore } from '../lib/store';
import { Search, Copy, Fingerprint, Globe, Bug, ExternalLink } from 'lucide-react';
import { colors } from '../lib/theme';
import { DataTable } from '../components/table/DataTable';
import type { ColumnDef } from '@tanstack/react-table';
import type { CallRecord } from '../lib/types';

// Success when there is no gRPC error code (API omits code 0 on success).
const isOk = (r: CallRecord) => !r.code || r.code === 0;

// gRPC status code → canonical name.
const GRPC_CODE: Record<number, string> = {
  0: 'OK', 1: 'Canceled', 2: 'Unknown', 3: 'InvalidArgument', 4: 'DeadlineExceeded',
  5: 'NotFound', 6: 'AlreadyExists', 7: 'PermissionDenied', 8: 'ResourceExhausted',
  9: 'FailedPrecondition', 10: 'Aborted', 11: 'OutOfRange', 12: 'Unimplemented',
  13: 'Internal', 14: 'Unavailable', 15: 'DataLoss', 16: 'Unauthenticated',
};
const codeName = (c?: number) => (c == null ? '' : GRPC_CODE[c] ?? String(c));

function grpcurl(r: CallRecord): string {
  const msgs = r.requests?.length ? r.requests : (r.request ? [r.request] : [{}]);
  const data = msgs.length === 1 ? JSON.stringify(msgs[0]) : msgs.map((m) => JSON.stringify(m)).join('\n');
  return `grpcurl -plaintext -d '${data}' localhost:4770 ${r.service}/${r.method}`;
}

export function HistoryList() {
  const navigate = useNavigate();
  const session = useStore((s) => s.session);
  const { data, isLoading, hasNextPage, isFetchingNextPage, fetchNextPage } = useInfiniteHistory(100, 10_000);
  const [sp] = useSearchParams();
  const [search, setSearch] = useState(sp.get('q') || '');
  const [sessionTab, setSessionTab] = useState('all');
  const [statusTab, setStatusTab] = useState<'all' | 'ok' | 'err'>('all');

  const total = data?.pages[0]?.total ?? 0;
  // Flatten loaded pages and sort newest-first (windows arrive oldest-first).
  const history = useMemo(() => {
    const rows = data?.pages.flatMap((p) => p.data) ?? [];
    return rows.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
  }, [data]);

  const filtered = useMemo(() => {
    if (!history) return [];
    return history.filter((h) => {
      if (sessionTab === 'mine' && h.session !== session) return false;
      if (sessionTab === 'global' && h.session) return false;
      if (statusTab === 'ok' && !isOk(h)) return false;
      if (statusTab === 'err' && isOk(h)) return false;
      if (!search) return true;
      const q = search.toLowerCase();
      return [h.service, h.method, h.stubId, h.error].some((f) => f?.toLowerCase().includes(q));
    });
  }, [history, search, sessionTab, statusTab, session]);

  const errorCount = useMemo(() => (history ?? []).filter((h) => !isOk(h)).length, [history]);

  const tab = (active: boolean): React.CSSProperties => ({
    padding: '5px 11px', fontSize: 12, borderRadius: 'var(--radius-sm)', cursor: 'pointer', whiteSpace: 'nowrap',
    border: 'none', background: active ? 'var(--bg-elevated)' : 'transparent',
    color: active ? 'var(--text)' : 'var(--text-secondary)', fontWeight: active ? 600 : 500,
    boxShadow: active ? 'var(--shadow-sm)' : undefined,
    display: 'inline-flex', alignItems: 'center', gap: 4,
  });

  const inspectCall = (r: CallRecord) => {
    const payload = r.requests?.[0] ?? r.request ?? {};
    navigate(`/inspect?service=${encodeURIComponent(r.service ?? '')}&method=${encodeURIComponent(r.method ?? '')}&payload=${encodeURIComponent(JSON.stringify(payload))}`);
  };

  const columns = useMemo<ColumnDef<CallRecord>[]>(() => [
    { id: '_bar', header: '', cell: (info) => (
      <span style={{ display: 'inline-block', width: 3, height: 18, borderRadius: 2, background: isOk(info.row.original) ? colors.success : colors.error }} />
    ), size: 6 },
    { id: 'service', header: 'Service', accessorKey: 'service', cell: (info) => <span style={{ fontWeight: 500 }}>{info.getValue() as string}</span> },
    { id: 'method', header: 'Method', accessorKey: 'method' },
    { id: 'stubId', header: 'Stub', accessorKey: 'stubId', cell: (info) => {
      const v = info.getValue() as string;
      return v
        ? <button type="button" onClick={(e) => { e.stopPropagation(); navigate(`/stubs/${v}`); }}
            title={`${v}\nOpen stub`} className="hover-row"
            style={{ font: 'inherit', fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--accent-text)', cursor: 'pointer', padding: '1px 4px', borderRadius: 3, background: 'none', border: 'none', textAlign: 'inherit' }}>{v.slice(0, 8)}</button>
        : <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>no match</span>;
    }},
    { id: 'time', header: 'Time', accessorKey: 'timestamp', cell: (info) => (
      <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{new Date(info.getValue() as string).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</span>
    ), meta: { hideBelow: 1280 }},
    { id: 'elapsed', header: 'Latency', accessorKey: 'elapsedMs', cell: (info) => {
      const ms = info.getValue() as number | undefined;
      return ms != null
        ? <span style={{ fontSize: 11, fontFamily: 'var(--mono)', color: ms > 500 ? colors.warning : 'var(--text-secondary)' }}>{ms} ms</span>
        : <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>—</span>;
    }},
    { id: 'status', header: 'Status', cell: (info) => {
      const r = info.row.original;
      return isOk(r)
        ? <span className="badge" style={{ background: 'var(--success-bg)', color: colors.success }}>OK</span>
        : <span className="badge" style={{ background: 'var(--error-bg)', color: colors.error }} title={r.error || ''}>{codeName(r.code)}</span>;
    }},
    { id: 'cp', header: '', cell: (info) => (
      <button onClick={(e) => { e.stopPropagation(); navigator.clipboard.writeText(grpcurl(info.row.original)); }}
        className="icon-btn" style={{ width: 24, height: 24 }} title="Copy grpcurl"><Copy size={12} /></button>
    ), size: 30 },
  ], [navigate]);

  return (
    <div className="page-enter" style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <h1>History <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>({history.length < total ? `${history.length} of ${total}` : total}{errorCount > 0 ? ` · ${errorCount} errors` : ''})</span></h1>

      <div className="toolbar">
        <div className="search" style={{ flex: '1 1 220px', minWidth: 160 }}>
          <Search size={13} />
          <input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search service, method, stub, error…" className="input" />
        </div>

        <div className="tabs">
          <button style={tab(statusTab === 'all')} onClick={() => setStatusTab('all')}>All</button>
          <button style={tab(statusTab === 'ok')} onClick={() => setStatusTab('ok')}>OK</button>
          <button style={tab(statusTab === 'err')} onClick={() => setStatusTab('err')}>
            Errors{errorCount > 0 && <span style={{ color: colors.error }}>{errorCount}</span>}
          </button>
        </div>

        {session && (
          <div className="tabs" title="History is scoped to the active session on the server; these split its calls from global ones.">
            <button style={tab(sessionTab === 'all')} onClick={() => setSessionTab('all')}>All</button>
            <button style={tab(sessionTab === 'mine')} onClick={() => setSessionTab('mine')}><Fingerprint size={11} /> Mine</button>
            <button style={tab(sessionTab === 'global')} onClick={() => setSessionTab('global')}><Globe size={11} /> Global</button>
          </div>
        )}
      </div>
      {session && (
        <div style={{ fontSize: 11.5, color: 'var(--text-muted)' }}>
          Scoped to session <code style={{ color: 'var(--accent-text)' }}>{session.slice(0, 16)}</code> — the server returns only this session's calls plus global ones.
        </div>
      )}

      <DataTable data={filtered} columns={columns} loading={isLoading} emptyMessage="No calls recorded yet"
        renderExpanded={(r: CallRecord) => (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8, fontSize: 12 }}>
            <div style={{ color: 'var(--text-muted)', display: 'flex', gap: 14, flexWrap: 'wrap', alignItems: 'center' }}>
              <span>Status: <strong style={{ color: isOk(r) ? colors.success : colors.error }}>{isOk(r) ? 'OK' : `${codeName(r.code)} (${r.code})`}</strong></span>
              <span>Session: {r.session ? <code style={{ color: 'var(--accent-text)' }}>{r.session.slice(0, 16)}</code> : <span style={{ color: colors.success }}>Global</span>}</span>
              <span>Stub: {r.stubId ? <button type="button" onClick={() => navigate(`/stubs/${r.stubId}`)} style={{ font: 'inherit', fontFamily: 'var(--mono)', color: 'var(--accent-text)', cursor: 'pointer', background: 'none', border: 'none', padding: 0, textAlign: 'inherit' }}>{r.stubId.slice(0, 12)}</button> : 'no match'}</span>
              <span>{new Date(r.timestamp).toLocaleString()}</span>
            </div>

            {!isOk(r) && r.error && (
              <div style={{ padding: '7px 10px', borderRadius: 'var(--radius)', background: 'var(--error-bg)', border: `1px solid ${colors.error}40`, color: colors.error, fontSize: 12, fontFamily: 'var(--mono)', whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                {r.error}
              </div>
            )}

            <div style={{ display: 'flex', gap: 10 }}>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div className="section-title" style={{ marginBottom: 3 }}>Request {(r.requests?.length ?? 0) > 1 ? `(${r.requests!.length} msgs)` : ''}</div>
                <pre className="json-block">{JSON.stringify(r.requests ?? r.request ?? {}, null, 2)}</pre>
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div className="section-title" style={{ marginBottom: 3 }}>Response {(r.responses?.length ?? 0) > 1 ? `(${r.responses!.length} msgs)` : ''}</div>
                <pre className="json-block">{JSON.stringify(r.responses ?? r.response ?? {}, null, 2)}</pre>
              </div>
            </div>

            <div style={{ display: 'flex', gap: 6 }}>
              <button onClick={() => inspectCall(r)} className="btn btn-sm"><Bug size={12} /> Inspect this call</button>
              {r.stubId && <button onClick={() => navigate(`/stubs/${r.stubId}`)} className="btn btn-sm"><ExternalLink size={12} /> Open stub</button>}
              <button onClick={() => navigator.clipboard.writeText(grpcurl(r))} className="btn btn-sm"><Copy size={12} /> Copy grpcurl</button>
            </div>
          </div>
        )}
      />

      {hasNextPage && (
        <div style={{ display: 'flex', justifyContent: 'center', padding: 4 }}>
          <button onClick={() => fetchNextPage()} disabled={isFetchingNextPage} className="btn btn-sm">
            {isFetchingNextPage ? 'Loading…' : `Load older calls (${total - history.length} more)`}
          </button>
        </div>
      )}
    </div>
  );
}
