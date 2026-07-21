import { useState, useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useServices, useServiceMethod } from '../hooks/useServices';
import { useStubs } from '../hooks/useStubs';
import { useDescriptors } from '../hooks/useDescriptors';
import { api } from '../lib/api';
import { serviceRefMatches } from '../lib/stub';
import { useToast } from '../components/shared/Toast';
import { Search, ChevronDown, ChevronRight, Loader2, Plus, History, Trash2 } from 'lucide-react';
import { colors } from '../lib/theme';
import { DataTable } from '../components/table/DataTable';
import type { ColumnDef } from '@tanstack/react-table';
import type { Service, Method, ProtoFieldSchema, ProtoMessageSchema } from '../lib/types';

export function ServicesList() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const toast = useToast();
  const { data: services, isLoading } = useServices();
  const { data: stubs } = useStubs();
  const { data: descriptors } = useDescriptors();
  const [sp] = useSearchParams();
  const [search, setSearch] = useState(sp.get('q') || '');

  // Services registered at runtime (via descriptor upload) can be unregistered.
  const runtime = useMemo(() => new Set(descriptors?.serviceIDs ?? []), [descriptors]);
  const unregister = useMutation({
    mutationFn: (id: string) => api.delete(`/services/${encodeURIComponent(id)}`),
    onSuccess: (_r, id) => { qc.invalidateQueries({ queryKey: ['services'] }); qc.invalidateQueries({ queryKey: ['descriptors'] }); toast.show(`Unregistered ${id}`); },
    onError: (e) => toast.show((e as Error).message),
  });

  // Stub coverage — a stub's `service` may or may not include the package,
  // so match against both the FQN (id) and the bare service name.
  const stubCountFor = (svc: Service, methodName: string) =>
    (stubs ?? []).filter((s) => s.method === methodName && serviceRefMatches(s.service, svc.id, svc.name)).length;
  const covered = (svc: Service) => svc.methods.filter((mm) => stubCountFor(svc, mm.name) > 0).length;

  const filtered = useMemo(() => {
    if (!services) return [];
    const q = search.toLowerCase();
    return services.filter((s) => [s.id, s.name, s.package].some((f) => f.toLowerCase().includes(q))
      || s.methods.some((mm) => mm.name.toLowerCase().includes(q)));
  }, [services, search]);

  const columns = useMemo<ColumnDef<Service>[]>(() => [
    { id: 'id', header: 'Service', accessorKey: 'id', cell: (info) => <span style={{ fontFamily: 'var(--mono)', fontSize: 12 }}>{info.getValue() as string}</span> },
    { id: 'name', header: 'Name', accessorKey: 'name', cell: (info) => <span style={{ fontWeight: 500 }}>{info.getValue() as string}</span> },
    { id: 'methods', header: 'RPCs', accessorKey: 'methods', cell: (info) => (info.getValue() as Method[]).length },
    { id: 'coverage', header: 'Coverage', cell: (info) => {
      const svc = info.row.original; const c = covered(svc); const total = svc.methods.length;
      const color = c === total ? colors.success : c === 0 ? colors.error : colors.warning;
      return <span className="badge" style={{ background: `${color}1e`, color }} title="methods with at least one stub">{c}/{total} covered</span>;
    }},
    { id: 'streaming', header: 'Stream', cell: (info) => {
      const n = info.row.original.methods.filter((m) => m.clientStreaming || m.serverStreaming).length;
      return n > 0 ? <span className="badge" style={{ background: '#9333ea1e', color: '#9333ea' }} title={`${n} streaming ${n === 1 ? 'method' : 'methods'}`}>{n} streaming</span> : <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>—</span>;
    }},
    // eslint-disable-next-line react-hooks/exhaustive-deps
  ], [stubs]);

  return (
    <div className="page-enter" style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <h1>Services {services && <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>({services.length})</span>}</h1>
      <div className="search" style={{ maxWidth: 300 }}>
        <Search size={13} />
        <input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search service or method…" className="input" />
      </div>

      <DataTable data={filtered} columns={columns} loading={isLoading} emptyMessage="No services found"
        renderExpanded={(svc: Service) => (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10, fontSize: 12 }}>
            <div style={{ color: 'var(--text-muted)', fontSize: 12, padding: '0 2px', display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <span>
                <code style={{ color: 'var(--text)', fontWeight: 600 }}>{svc.name}</code>
                {' · '}Package <code style={{ color: 'var(--text-secondary)' }}>{svc.package || '—'}</code>
                {' · '}{svc.methods.length} RPCs · <span style={{ color: covered(svc) === svc.methods.length ? colors.success : colors.warning }}>{covered(svc)}/{svc.methods.length} covered</span>
              </span>
              {runtime.has(svc.id) && (
                <>
                  <span className="badge" style={{ background: 'var(--accent-bg)', color: 'var(--accent-text)' }}>runtime</span>
                  <button className="btn btn-ghost btn-sm" style={{ color: colors.error }} disabled={unregister.isPending}
                    onClick={() => { if (confirm(`Unregister ${svc.id}?`)) unregister.mutate(svc.id); }}
                    title="Unregister this runtime service"><Trash2 size={12} /> Unregister</button>
                </>
              )}
            </div>
            {svc.methods.map((m, i) => (
              <MethodBlock key={m.id || i} method={m} serviceId={svc.id} navigate={navigate} stubCount={stubCountFor(svc, m.name)} />
            ))}
          </div>
        )}
      />
    </div>
  );
}

function MethodBlock({ method, serviceId, navigate, stubCount }: { method: Method; serviceId: string; navigate: (p: string) => void; stubCount: number }) {
  const [expanded, setExpanded] = useState(false);
  const { data: methodDetail, isFetching } = useServiceMethod(expanded ? serviceId : null, expanded ? method.name : null);

  // Use fetched detail schema, fallback to initial method schema
  const reqSchema = methodDetail?.requestSchema || method.requestSchema;
  const resSchema = methodDetail?.responseSchema || method.responseSchema;

  return (
    <div className="card">
      <div onClick={() => setExpanded(!expanded)} role="button" tabIndex={0} onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setExpanded(!expanded); } }} className="card-header hover-row" style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer', textTransform: 'none', letterSpacing: 0 }}>
        {expanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
        <span style={{ fontWeight: 600, fontSize: 13, color: 'var(--text)' }}>{method.name}</span>
        <span className="badge" style={badgeStyle(method.methodType)}>{method.methodType === 'unary' ? 'U' : method.methodType === 'server_streaming' ? 'SS' : method.methodType === 'client_streaming' ? 'CS' : 'BD'}</span>
        {stubCount > 0
          ? <span className="badge" style={{ background: 'var(--success-bg)', color: colors.success }} title="stubs covering this method">{stubCount} stub{stubCount > 1 ? 's' : ''}</span>
          : <span className="badge" style={{ background: 'var(--error-bg)', color: colors.error }} title="no stubs — this method is uncovered">no stubs</span>}
        <span style={{ fontSize: 11, color: 'var(--text-muted)', fontFamily: 'var(--mono)', marginLeft: 4 }}>
          {method.requestType} → {method.responseType}
        </span>
        <div style={{ flex: 1 }} />
        {isFetching && <Loader2 size={12} className="animate-spin" style={{ color: 'var(--text-muted)' }} />}
        <button onClick={(e) => { e.stopPropagation(); navigate(`/stubs/create?service=${encodeURIComponent(serviceId)}&method=${encodeURIComponent(method.name)}`); }}
          className="btn btn-ghost btn-sm" title="Create a stub for this method"><Plus size={12} /></button>
        <button onClick={(e) => { e.stopPropagation(); navigate(`/history?q=${encodeURIComponent(method.name)}`); }}
          className="btn btn-ghost btn-sm" title="View calls to this method"><History size={12} /></button>
        <button onClick={(e) => { e.stopPropagation(); navigate(`/stubs?service=${encodeURIComponent(serviceId)}&method=${encodeURIComponent(method.name)}`); }}
          className="btn btn-ghost btn-sm">Stubs</button>
      </div>

      {expanded && (
        <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: 8, minHeight: isFetching ? 60 : undefined }}>
          {isFetching ? (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6, padding: 20, color: 'var(--text-muted)', fontSize: 11 }}>
              <Loader2 size={12} className="animate-spin" /> Loading schema...
            </div>
          ) : (
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
              <SchemaView title={`Request: ${reqSchema?.typeName || method.requestType}`} schema={reqSchema} />
              <SchemaView title={`Response: ${resSchema?.typeName || method.responseType}`} schema={resSchema} />
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function SchemaView({ title, schema }: { title: string; schema?: ProtoMessageSchema | null }) {
  return (
    <div className="card" style={{ background: 'var(--bg)' }}>
      <div className="card-header" style={{ fontSize: 11, borderBottom: 'none', paddingBottom: 4, textTransform: 'none', letterSpacing: 0 }}>{title}</div>
      <div className="card-body" style={{ padding: '2px 6px 6px' }}>
        {schema?.fields?.length ? (
          <SchemaFieldsTable fields={schema.fields} depth={0} />
        ) : (
          <span style={{ fontSize: 11, color: 'var(--text-muted)', fontStyle: 'italic' }}>No fields</span>
        )}
        {schema?.recursiveRef && <span className="chip" style={{ background: `${colors.warning}18`, color: colors.warning, fontSize: 11, marginTop: 4 }}>recursive</span>}
      </div>
    </div>
  );
}

function fieldType(f: ProtoFieldSchema): string {
  if (f.map) return `map<${f.mapKeyKind ?? '?'}, ${f.mapValueTypeName ?? f.mapValueKind ?? '?'}>`;
  return f.typeName || f.kind;
}

function SchemaFieldsTable({ fields, depth }: { fields: ProtoFieldSchema[]; depth: number }) {
  return (
    <div>
      {fields.map((f, i) => (
        <div key={f.name || i}>
          <div style={{
            display: 'flex', alignItems: 'center', gap: 5, fontSize: 11.5, flexWrap: 'wrap',
            padding: '3px 2px 3px 0', borderBottom: '1px solid var(--border)',
            marginLeft: depth * 14,
          }}>
            <span style={{ color: 'var(--text)', fontFamily: 'var(--mono)', fontWeight: 500, minWidth: 60 }}>{f.name}</span>
            <span style={{ color: f.map ? '#0891b2' : f.enumValues?.length ? colors.warning : '#5570e6', fontFamily: 'var(--mono)' }}>{fieldType(f)}</span>
            {!f.map && f.typeName && f.typeName !== f.kind && !f.message && <span style={{ color: '#9333ea', fontFamily: 'var(--mono)' }}>{f.kind}</span>}
            <span style={{ color: 'var(--text-muted)' }}>#{f.number}</span>
            {f.cardinality === 'repeated' && <RepeatedBadge />}
            {f.cardinality === 'required' && <RequiredBadge />}
            {f.oneof && <OneofBadge label={f.oneof} />}
            {f.enumValues && f.enumValues.length > 0 && (
              <span style={{ color: 'var(--text-muted)', fontSize: 11, fontStyle: 'italic' }} title={f.enumValues.join(', ')}>
                enum: {f.enumValues.slice(0, 4).join(' | ')}{f.enumValues.length > 4 ? ' …' : ''}
              </span>
            )}
          </div>
          {f.message && f.message.fields?.length > 0 && (
            <SchemaFieldsTable fields={f.message.fields} depth={depth + 1} />
          )}
        </div>
      ))}
    </div>
  );
}

function RepeatedBadge() { return <span className="chip" style={{ background: '#f59e0b18', color: '#f59e0b', fontSize: 11 }}>repeated</span>; }
function RequiredBadge() { return <span className="chip" style={{ background: '#3b82f618', color: '#3b82f6', fontSize: 11 }}>required</span>; }
function OneofBadge({ label }: { label: string }) { return <span className="chip" style={{ background: '#a855f718', color: '#a855f7', fontSize: 11 }}>oneof: {label}</span>; }

const badgeStyle = (type: string) => {
  const m: Record<string, string> = { unary: '#3b82f6', client_streaming: '#f59e0b', server_streaming: '#a855f7', bidi_streaming: '#ef4444' };
  return { background: `${m[type] || '#64748b'}18`, color: m[type] || '#64748b', fontWeight: 700, fontSize: 11 };
};
