import { useStore } from '../../lib/store';
import { getApiUrl } from '../../lib/api';
import { colors } from '../../lib/theme';
import { HealthDot } from '../shared/HealthDot';
import { useReadiness } from '../../hooks/useReadiness';

// Identity, health and counters live in the top bar; this line carries what the
// top bar has no room for — where the UI is pointed and what the last probe said.
export function StatusBar() {
  const session = useStore((s) => s.session);
  const { ready, offline, label } = useReadiness();

  return (
    <footer style={{
      height: 24, borderTop: '1px solid var(--border)',
      display: 'flex', alignItems: 'center', padding: '0 10px', gap: 10,
      fontSize: 11, color: 'var(--text-muted)', background: 'var(--bg-secondary)', flexShrink: 0,
    }}>
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
        <HealthDot ready={offline ? false : ready} />
        <span style={{ color: offline ? colors.error : 'var(--text-muted)' }}>{label}</span>
      </span>

      <span style={{ fontFamily: 'monospace' }} title="API base URL (change it in connection settings)">{getApiUrl()}</span>

      <div style={{ flex: 1 }} />

      <span title={session ? 'Requests carry this session header' : 'Requests carry no session header'}>
        scope: {session ?? 'global'}
      </span>
    </footer>
  );
}
