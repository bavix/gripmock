import { useNavigate } from 'react-router-dom';
import { useMemo } from 'react';
import { useStore } from '../lib/store';
import { useSessions } from '../hooks/useSessions';
import { useStubs } from '../hooks/useStubs';
import { colors } from '../lib/theme';
import { Fingerprint, Globe, Plus, Copy, History, ListOrdered, ShieldCheck } from 'lucide-react';
import { useToast } from '../components/shared/Toast';

export function SessionPage() {
  const navigate = useNavigate();
  const toast = useToast();
  const session = useStore((s) => s.session);
  const setSession = useStore((s) => s.setSession);
  const trackSession = useStore((s) => s.trackSession);
  const recent = useStore((s) => s.recentSessions);
  const { data: backend } = useSessions();
  const { data: stubs } = useStubs();

  // Session-scoped stub counts (stub.session === id).
  const stubCount = useMemo(() => {
    const m: Record<string, number> = {};
    for (const s of stubs ?? []) if (s.session) m[s.session] = (m[s.session] ?? 0) + 1;
    return m;
  }, [stubs]);

  // Union of local recent + server-reported sessions, de-duplicated.
  const all = Array.from(new Set([...(recent ?? []), ...(backend?.sessions ?? [])]));

  const activate = (s: string) => { setSession(s); trackSession(s); };
  const newSession = () => { const id = `sess-${crypto.randomUUID().slice(0, 8)}`; activate(id); toast.show(`Switched to ${id}`); };
  const copy = (s: string) => { navigator.clipboard.writeText(s); toast.show('Copied session ID'); };

  return (
    <div className="page-enter" style={{ display: 'flex', flexDirection: 'column', gap: 12, maxWidth: 720 }}>
      <h1>Session Scope</h1>

      {/* What a session affects */}
      <div style={{ display: 'flex', gap: 8, padding: '10px 14px', borderRadius: 'var(--radius-lg)', border: '1px solid var(--border)', background: 'var(--bg-secondary)', fontSize: 12.5, color: 'var(--text-secondary)', lineHeight: 1.5 }}>
        <ShieldCheck size={16} style={{ color: colors.accent, flexShrink: 0, marginTop: 1 }} />
        <span>
          The active session sends an <code>X-Gripmock-Session</code> header on every UI request. It scopes
          <strong> session-specific stubs</strong>, and the counters/records used by <strong>Verify</strong>, <strong>Inspect</strong> and <strong>History</strong>.
          <em style={{ color: 'var(--text-muted)' }}> Global</em> means no header — you see only global stubs and calls.
        </span>
      </div>

      {/* Active scope */}
      <div className="card">
        <div className="card-header">Active scope</div>
        <div className="card-body" style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
          {session
            ? <><Fingerprint size={16} style={{ color: colors.accent }} /><code style={{ color: 'var(--accent-text)', fontSize: 13 }}>{session}</code>
                <button className="icon-btn" style={{ width: 26, height: 26 }} onClick={() => copy(session)} title="Copy"><Copy size={13} /></button></>
            : <><Globe size={16} style={{ color: colors.success }} /><span style={{ color: colors.success, fontWeight: 600 }}>Global (no session)</span></>}
          <div style={{ flex: 1 }} />
          {session && <button className="btn btn-sm" onClick={() => setSession(null)}><Globe size={13} /> Use Global</button>}
          <button className="btn btn-sm btn-primary" onClick={newSession}><Plus size={13} /> New session</button>
        </div>
        {session && (
          <div style={{ display: 'flex', gap: 6, padding: '0 14px 12px', flexWrap: 'wrap' }}>
            <button className="btn btn-ghost btn-sm" onClick={() => navigate('/history')}><History size={12} /> History in this scope</button>
            <button className="btn btn-ghost btn-sm" onClick={() => navigate('/stubs')}><ListOrdered size={12} /> Stubs in this scope</button>
          </div>
        )}
      </div>

      {/* Known sessions */}
      <div className="card">
        <div className="card-header">Known sessions ({all.length})</div>
        <div style={{ padding: all.length === 0 ? 0 : 10 }}>
          {all.length === 0
            ? <div className="empty" style={{ padding: 26 }}>
                <Globe size={24} />
                <span>No sessions yet. Create one above, or they appear when gRPC calls carry an <code>X-Gripmock-Session</code> header.</span>
              </div>
            : <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                {all.map((s) => {
                  const active = session === s;
                  return (
                    <span key={s} className="hover-row" style={{
                      display: 'inline-flex', alignItems: 'center', gap: 5, padding: '5px 6px 5px 10px', borderRadius: 999,
                      border: `1px solid ${active ? 'var(--accent)' : 'var(--border)'}`, background: active ? 'var(--accent-bg)' : 'var(--bg)',
                    }}>
                      <button onClick={() => activate(s)} title="Activate" style={{ border: 'none', background: 'none', cursor: 'pointer', fontFamily: 'var(--mono)', fontSize: 12, color: active ? 'var(--accent-text)' : 'var(--text)', padding: 0 }}>{s}</button>
                      {stubCount[s] > 0 && <span className="badge" style={{ background: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }} title="session-scoped stubs">{stubCount[s]}</span>}
                      <button className="icon-btn" style={{ width: 20, height: 20 }} onClick={() => copy(s)} title="Copy"><Copy size={11} /></button>
                    </span>
                  );
                })}
              </div>}
        </div>
      </div>
    </div>
  );
}
