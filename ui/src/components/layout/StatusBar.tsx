import { useQuery } from '@tanstack/react-query';
import { Fingerprint, Globe } from 'lucide-react';
import { useStore } from '../../lib/store';
import { api } from '../../lib/api';
import { colors } from '../../lib/theme';
import type { Dashboard } from '../../lib/types';

export function StatusBar() {
  const session = useStore((s) => s.session);
  const { data: dash } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => api.get<Dashboard>('/dashboard'),
    refetchInterval: 30_000,
  });

  // Dedicated readiness probe on a short cadence: catches server-down (a failed
  // request) as "offline" instead of the dashboard query silently going stale.
  const health = useQuery({
    queryKey: ['health', 'readiness'],
    queryFn: () => api.get('/health/readiness'),
    refetchInterval: 10_000,
    retry: false,
  });
  const ready = health.isError ? false : health.isSuccess ? true : dash?.ready;
  const healthLabel = health.isError ? 'Server unreachable' : ready ? 'Ready' : ready === false ? 'Not ready' : 'Checking…';

  return (
    <footer style={{
      height: 24, borderTop: '1px solid var(--border)',
      display: 'flex', alignItems: 'center', padding: '0 10px', gap: 10,
      fontSize: 11, color: 'var(--text-muted)', background: 'var(--bg-secondary)', flexShrink: 0,
    }}>
      <span>{dash?.version ? `${/^\d/.test(dash.version) ? 'v' : ''}${dash.version}` : '?'}</span>
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }} title={healthLabel}>
        <HealthDot ready={ready} />
        {health.isError && <span style={{ color: colors.error }}>offline</span>}
      </span>
      <span>{dash?.totalStubs ?? 0} stubs · {dash?.totalHistory ?? 0} calls</span>
      {dash && dash.historyErrors > 0 && (
        <span style={{ color: colors.error }}>{dash.historyErrors} errs</span>
      )}
      <div style={{ flex: 1 }} />
      {session ? (
        <span style={{ display: 'flex', alignItems: 'center', gap: 3, color: colors.accent }}>
          <Fingerprint size={11} /> {session.slice(0, 12)}
        </span>
      ) : (
        <span style={{ display: 'flex', alignItems: 'center', gap: 3, color: colors.success }}>
          <Globe size={11} /> Global
        </span>
      )}
    </footer>
  );
}

function HealthDot({ ready }: { ready?: boolean }) {
  return (
    <span style={{
      width: 6, height: 6, borderRadius: '50%', display: 'inline-block',
      background: ready === undefined ? '#64748b' : (ready ? colors.success : colors.error),
    }} />
  );
}
