import { useNavigate } from 'react-router-dom';
import { useMemo, lazy, Suspense } from 'react';
import { useDashboard } from '../hooks/useDashboard';
import { useRecentHistory } from '../hooks/useHistory';
import { useCopy } from '../hooks/useCopy';
import { latencyStats } from '../features/dashboard/buckets';

// Charts are code-split (vendor-charts chunk) and only load on the dashboard.
const CallsTrend = lazy(() => import('../features/dashboard/CallsTrend'));
const CoverageDonut = lazy(() => import('../features/dashboard/CoverageDonut'));
import { getApiUrl } from '../lib/api';
import { colors } from '../lib/theme';
import {
  Activity, ListOrdered, History, AlertTriangle, Layers,
  Clock, Plus, Search, CheckCircle2, ShieldCheck, Copy, Plug,
  WifiOff, RefreshCw, FlaskConical, Cpu, Users, FileUp, ArrowRight, Timer,
} from 'lucide-react';
import type { CallRecord } from '../lib/types';

const callOk = (c: CallRecord) => !c.code || c.code === 0;

function MetaItem({ icon: Icon, label, value }: Readonly<{ icon: typeof Clock; label: string; value: string }>) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 7, minWidth: 0 }}>
      <Icon size={14} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 10, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.4px', lineHeight: 1.2 }}>{label}</div>
        <div style={{ fontSize: 12.5, color: 'var(--text-secondary)', fontWeight: 500, whiteSpace: 'nowrap' }}>{value}</div>
      </div>
    </div>
  );
}

// Compact uptime: "45s" / "12m" / "3h 20m" / "2d 4h".
function fmtUptime(sec: number): string {
  if (sec < 60) return `${sec}s`;
  const m = Math.floor(sec / 60) % 60, h = Math.floor(sec / 3600) % 24, d = Math.floor(sec / 86400);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

function Endpoint({ proto, addr }: Readonly<{ proto: string; addr: string }>) {
  const { copied, copy } = useCopy();
  // 0.0.0.0/empty host is not dialable from a browser — show localhost.
  const shown = addr.replace(/^(0\.0\.0\.0|\[::\]|):/, 'localhost:').replace(/^:/, 'localhost:');
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 7 }}>
      <span style={{ fontSize: 10.5, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.4px' }}>{proto}</span>
      <button type="button" onClick={() => copy(shown)} title="Click to copy"
        style={{ display: 'inline-flex', alignItems: 'center', gap: 5, cursor: 'pointer', fontFamily: 'var(--mono)', fontSize: 12, fontWeight: 600, color: 'var(--text)', background: 'var(--bg-tertiary)', border: 'none', padding: '2px 8px', borderRadius: 4, userSelect: 'all', textAlign: 'inherit' }}>
        {shown}
        {copied ? <CheckCircle2 size={11} style={{ color: colors.success }} /> : <Copy size={10} style={{ color: 'var(--text-muted)' }} />}
      </button>
    </span>
  );
}

function LatencyStat({ label, value, warn }: Readonly<{ label: string; value: number; warn?: boolean }>) {
  return (
    <span style={{ display: 'inline-flex', alignItems: 'baseline', gap: 5 }}>
      <span style={{ fontSize: 10.5, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.4px' }}>{label}</span>
      <span style={{ fontSize: 15, fontWeight: 700, color: warn ? colors.warning : 'var(--text)' }}>{value}<span style={{ fontSize: 10.5, fontWeight: 500, color: 'var(--text-muted)', marginLeft: 1 }}>ms</span></span>
    </span>
  );
}

function ago(ts: string): string {
  const s = Math.max(0, Math.floor((Date.now() - new Date(ts).getTime()) / 1000));
  if (s < 60) return `${s}s`;
  if (s < 3600) return `${Math.floor(s / 60)}m`;
  if (s < 86400) return `${Math.floor(s / 3600)}h`;
  return `${Math.floor(s / 86400)}d`;
}

export function Dashboard() {
  const navigate = useNavigate();
  const { data: dash, isLoading, error, refetch } = useDashboard();
  // Bounded server-side feeds on the same 15s cadence: 18 rows for the feed,
  // 500 recent records for the trend chart.
  const { data: history } = useRecentHistory(18, 15_000);
  const { data: trendHistory } = useRecentHistory(500, 15_000);

  const liveFeed = useMemo(() => {
    if (!history) return [];
    return [...history].sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
  }, [history]);

  // Method coverage now comes from the server (scales to thousands of stubs;
  // no full-stub fetch on the client).
  const coverage = { covered: dash?.coveredMethods ?? 0, total: dash?.totalMethods ?? 0 };

  const latency = useMemo(() => latencyStats(trendHistory ?? []), [trendHistory]);

  if (isLoading) {
    return (
      <div className="page-enter" style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        <div className="card" style={{ height: 96, animation: 'pulse 1.5s ease-in-out infinite' }} />
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: 10 }}>
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="card" style={{ height: 78, animation: 'pulse 1.5s ease-in-out infinite', animationDelay: `${i * 0.08}s` }} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="empty" style={{ paddingTop: 70 }}>
        <WifiOff size={40} />
        <h2>Server not reachable</h2>
        <p style={{ maxWidth: 380, lineHeight: 1.6, margin: 0 }}>
          The GripMock API is not responding at <code style={{ background: 'var(--bg-tertiary)', padding: '1px 6px', borderRadius: 3 }}>{getApiUrl()}</code>
        </p>
        <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
          <button onClick={() => refetch()} className="btn btn-primary"><RefreshCw size={13} /> Retry</button>
          <button onClick={() => window.location.reload()} className="btn">Reload</button>
        </div>
      </div>
    );
  }

  if (!dash) return null;

  const uptime = fmtUptime(dash.uptimeSeconds);
  const usedPct = dash.totalStubs > 0 ? Math.round((dash.usedStubs / dash.totalStubs) * 100) : 0;
  const vlabel = `${/^\d/.test(dash.version) ? 'v' : ''}${dash.version}`;

  const stats: { icon: typeof Layers; label: string; value: number; color: string; to?: string }[] = [
    { icon: Layers, label: 'Services', value: dash.totalServices, color: colors.accent, to: '/services' },
    { icon: ListOrdered, label: 'Stubs', value: dash.totalStubs, color: colors.accent, to: '/stubs' },
    { icon: CheckCircle2, label: 'Used', value: dash.usedStubs, color: colors.success, to: '/stubs/used' },
    { icon: Activity, label: 'Unused', value: dash.unusedStubs, color: colors.warning, to: '/stubs/unused' },
    { icon: Users, label: 'Sessions', value: dash.totalSessions, color: '#9333ea', to: '/session' },
    { icon: FileUp, label: 'Descriptors', value: dash.runtimeDescriptors, color: '#0891b2', to: '/descriptors' },
    { icon: History, label: 'Calls', value: dash.totalHistory, color: colors.accent, to: '/history' },
    { icon: AlertTriangle, label: 'Errors', value: dash.historyErrors, color: dash.historyErrors > 0 ? colors.error : 'var(--text-muted)', to: '/history' },
  ];

  return (
    <div className="page-enter" style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      <div style={{
        borderRadius: 'var(--radius-xl)', border: '1px solid var(--border)', overflow: 'hidden',
        background: `linear-gradient(120deg, var(--accent-bg), transparent 55%), var(--bg-secondary)`,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '16px 20px', flexWrap: 'wrap' }}>
          <div style={{ width: 44, height: 44, borderRadius: 12, display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--accent-bg)', color: 'var(--accent-text)', flexShrink: 0 }}>
            <FlaskConical size={22} />
          </div>
          <div style={{ minWidth: 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ fontSize: 19, fontWeight: 650, letterSpacing: '-0.01em' }}>GripMock</span>
              <span className="badge" style={{ background: 'var(--bg-tertiary)', color: 'var(--text-secondary)', fontFamily: 'var(--mono)' }}>{vlabel}</span>
            </div>
            <div style={{ fontSize: 12.5, color: 'var(--text-muted)' }}>gRPC mock server</div>
          </div>
          <div style={{ flex: 1 }} />
          <span style={{
            display: 'inline-flex', alignItems: 'center', gap: 6, padding: '5px 12px', borderRadius: 999, fontWeight: 650, fontSize: 12.5,
            background: dash.ready ? 'var(--success-bg)' : 'var(--error-bg)', color: dash.ready ? colors.success : colors.error,
          }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', background: dash.ready ? colors.success : colors.error }} />
            {dash.ready ? 'Operational' : 'Degraded'}
          </span>
        </div>
        <div style={{ display: 'flex', gap: 28, flexWrap: 'wrap', padding: '10px 20px', borderTop: '1px solid var(--border)' }}>
          <MetaItem icon={Clock} label="Uptime" value={uptime} />
          <MetaItem icon={FlaskConical} label="Runtime" value={dash.goVersion.replace(/^go/, 'Go ')} />
          <MetaItem icon={Cpu} label="Platform" value={`${dash.goos}/${dash.goarch} · ${dash.numCPU} CPU`} />
          <MetaItem icon={History} label="Started" value={new Date(dash.startedAt).toLocaleString()} />
        </div>
      </div>

      {!dash.historyEnabled && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '9px 14px', borderRadius: 'var(--radius)', border: `1px solid ${colors.warning}40`, background: 'var(--warning-bg)', color: colors.warning, fontSize: 12.5 }}>
          <AlertTriangle size={15} /> Call history is disabled — History, the live feed and Verify have no data. Enable with <code>HISTORY_ENABLED=true</code>.
        </div>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: 10 }}>
        {stats.map((s) => (
          <button type="button" key={s.label} onClick={() => s.to && navigate(s.to)} className="card card-hover" style={{ cursor: 'pointer', padding: 14, display: 'flex', alignItems: 'center', gap: 12, font: 'inherit', color: 'inherit', textAlign: 'inherit', width: '100%' }}>
            <span style={{ width: 38, height: 38, borderRadius: 10, display: 'flex', alignItems: 'center', justifyContent: 'center', background: `${s.color}1e`, color: s.color, flexShrink: 0 }}>
              <s.icon size={18} />
            </span>
            <span style={{ display: 'block', minWidth: 0 }}>
              <span style={{ display: 'block', fontSize: 24, fontWeight: 700, lineHeight: 1.1 }}>{s.value}</span>
              <span style={{ display: 'block', fontSize: 11.5, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.4px' }}>{s.label}</span>
            </span>
          </button>
        ))}
      </div>

      {dash.totalStubs > 0 && (
        <div className="card" style={{ padding: '12px 16px', display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ fontSize: 12.5, fontWeight: 600 }}>Stub usage</span>
          <div style={{ flex: 1, height: 8, borderRadius: 999, background: 'var(--bg-tertiary)', overflow: 'hidden' }}>
            <div style={{ width: `${usedPct}%`, height: '100%', borderRadius: 999, background: usedPct > 80 ? colors.warning : colors.success, transition: 'width 0.3s' }} />
          </div>
          <span style={{ fontSize: 12.5, color: 'var(--text-muted)' }}><strong style={{ color: 'var(--text)' }}>{dash.usedStubs}</strong>/{dash.totalStubs} used · {usedPct}%</span>
        </div>
      )}

      {/* Endpoints — where clients connect, per protocol */}
      {(dash.grpcAddr || dash.gatewayAddr || dash.httpAddr) && (
        <div className="card" style={{ padding: '10px 16px', display: 'flex', alignItems: 'center', gap: 22, flexWrap: 'wrap' }}>
          <span style={{ fontSize: 12.5, fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: 6 }}>
            <Plug size={14} style={{ color: colors.accent }} /> Endpoints
          </span>
          {dash.grpcAddr && <Endpoint proto="gRPC" addr={dash.grpcAddr} />}
          {dash.gatewayAddr && <Endpoint proto="ConnectRPC" addr={dash.gatewayAddr} />}
          {dash.gatewayAddr && <Endpoint proto="gRPC-Web" addr={dash.gatewayAddr} />}
          {dash.httpAddr && <Endpoint proto="HTTP / REST" addr={dash.httpAddr} />}
        </div>
      )}

      {/* Latency summary (only when calls have measured durations) */}
      {dash.historyEnabled && latency.count > 0 && (
        <div className="card" style={{ padding: '10px 16px', display: 'flex', alignItems: 'center', gap: 20, flexWrap: 'wrap' }}>
          <span style={{ fontSize: 12.5, fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: 6 }}>
            <Timer size={14} style={{ color: colors.accent }} /> Response time
          </span>
          <LatencyStat label="avg" value={latency.avg} />
          <LatencyStat label="p95" value={latency.p95} warn={latency.p95 > 500} />
          <LatencyStat label="max" value={latency.max} warn={latency.max > 1000} />
          <span style={{ fontSize: 11, color: 'var(--text-muted)', marginLeft: 'auto' }}>over last {latency.count} timed calls</span>
        </div>
      )}

      {dash.historyEnabled && (
        <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 2fr) minmax(0, 1fr)', gap: 10 }}>
          <Suspense fallback={<div className="card" style={{ height: 214, animation: 'pulse 1.5s ease-in-out infinite' }} />}>
            <CallsTrend records={trendHistory ?? []} />
          </Suspense>
          <Suspense fallback={<div className="card" style={{ height: 214, animation: 'pulse 1.5s ease-in-out infinite' }} />}>
            <CoverageDonut covered={coverage.covered} total={coverage.total} />
          </Suspense>
        </div>
      )}

      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        <button onClick={() => navigate('/stubs/create')} className="btn btn-primary"><Plus size={14} /> New stub</button>
        <button onClick={() => navigate('/inspect')} className="btn"><Search size={14} /> Inspect matching</button>
        <button onClick={() => navigate('/verify')} className="btn"><ShieldCheck size={14} /> Verify calls</button>
        <button onClick={() => navigate('/services')} className="btn"><Layers size={14} /> Browse services</button>
      </div>

      <div className="card">
        <div className="card-header" style={{ display: 'flex', alignItems: 'center' }}>
          <span style={{ flex: 1 }}>Latest calls</span>
          {liveFeed.length > 0 && <button onClick={() => navigate('/history')} className="btn btn-ghost btn-sm" style={{ textTransform: 'none', letterSpacing: 0 }}>All history <ArrowRight size={12} /></button>}
        </div>
        <div style={{ maxHeight: 320, overflow: 'auto' }}>
          {liveFeed.length === 0 && (
            <div className="empty" style={{ padding: 32 }}>
              <History size={26} />
              <span>{dash.historyEnabled ? 'No calls yet — send a gRPC request to see it here.' : 'History is disabled.'}</span>
            </div>
          )}
          {liveFeed.map((call, i) => {
            const ok = callOk(call);
            return (
              <button type="button" key={i} onClick={() => call.stubId ? navigate(`/stubs/${call.stubId}`) : navigate('/history')} className="hover-row"
                style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 14px', font: 'inherit', fontSize: 12.5, color: 'inherit', textAlign: 'inherit', width: '100%', background: 'none', border: 'none', borderBottom: '1px solid var(--border)', borderLeft: `3px solid ${ok ? colors.success : colors.error}` }}>
                <span className="badge" style={{ background: ok ? 'var(--success-bg)' : 'var(--error-bg)', color: ok ? colors.success : colors.error, minWidth: 34, justifyContent: 'center' }}>{ok ? 'OK' : (call.code ?? 'ERR')}</span>
                <span style={{ fontWeight: 500, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  <span style={{ color: 'var(--text-muted)' }}>{call.service}/</span>{call.method}
                </span>
                {call.stubId
                  ? <code style={{ fontSize: 11, color: 'var(--accent-text)' }}>{call.stubId.slice(0, 8)}</code>
                  : <span style={{ fontSize: 11, color: colors.error }}>no match</span>}
                {call.elapsedMs != null && (
                  <span style={{ fontSize: 10.5, fontFamily: 'var(--mono)', color: call.elapsedMs > 500 ? colors.warning : 'var(--text-muted)', minWidth: 40, textAlign: 'right' }}>{call.elapsedMs}ms</span>
                )}
                <span style={{ fontSize: 11, color: 'var(--text-muted)', minWidth: 26, textAlign: 'right' }}>{ago(call.timestamp)}</span>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
