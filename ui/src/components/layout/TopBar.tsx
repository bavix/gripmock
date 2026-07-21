import { useState, useRef, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Menu, FlaskConical, Sun, Moon, Fingerprint, Globe, Settings } from 'lucide-react';
import { useStore } from '../../lib/store';
import { api, getApiUrl, setApiUrl, resetApiUrl } from '../../lib/api';
import { useFocusTrap } from '../../hooks/useFocusTrap';
import { colors, btn } from '../../lib/theme';
import type { Dashboard } from '../../lib/types';

interface TopBarProps {
  onToggleSidebar: () => void;
}

export function TopBar({ onToggleSidebar }: Readonly<TopBarProps>) {
  const theme = useStore((s) => s.theme);
  const setTheme = useStore((s) => s.setTheme);
  const session = useStore((s) => s.session);
  const setSession = useStore((s) => s.setSession);
  const trackSession = useStore((s) => s.trackSession);
  const recentSessions = useStore((s) => s.recentSessions);
  const [showMenu, setShowMenu] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [apiUrlInput, setApiUrlInput] = useState(getApiUrl());
  const menuRef = useRef<HTMLDivElement>(null);
  const settingsRef = useFocusTrap<HTMLDivElement>(showSettings, () => setShowSettings(false));

  const { data: dash } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => api.get<Dashboard>('/dashboard'),
    refetchInterval: 30_000,
  });

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setShowMenu(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const sessionBorderColor = session ? `${colors.accent}50` : 'var(--border)';

  return (
    <header style={{
      height: 44, borderBottom: '1px solid var(--border)',
      display: 'flex', alignItems: 'center', padding: '0 8px', gap: 6,
      background: 'var(--bg-secondary)', flexShrink: 0,
    }}>
      <button type="button" onClick={onToggleSidebar} style={iconBtn} title="Toggle sidebar">
        <Menu size={16} />
      </button>

      <FlaskConical size={16} color={colors.accent} />
      <span style={{ fontWeight: 600, fontSize: 13 }}>GripMock</span>
      {dash?.version && (
        <span style={{ fontSize: 11, color: 'var(--text-muted)', fontWeight: 400 }}>{/^\d/.test(dash.version) ? 'v' : ''}{dash.version}</span>
      )}

      <HealthDot ready={dash?.ready} />

      <div style={{ flex: 1 }} />

      <div ref={menuRef} style={{ position: 'relative' }}>
        <button type="button" onClick={() => setShowMenu((v) => !v)}
          style={{
            display: 'flex', alignItems: 'center', gap: 4,
            padding: '3px 8px', fontSize: 11, borderRadius: 5,
            border: `1px solid ${sessionBorderColor}`,
            background: session ? `${colors.accent}10` : 'transparent',
            color: session ? colors.accent : 'var(--text-muted)',
            cursor: 'pointer', fontWeight: 500,
          }}>
          {session ? <Fingerprint size={12} /> : <Globe size={12} />}
          <span style={{ maxWidth: 100, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {session ? session.slice(0, 10) : 'Global'}
          </span>
        </button>

        {showMenu && (
          <div style={{
            position: 'absolute', top: '100%', right: 0, zIndex: 100,
            minWidth: 180, marginTop: 4, padding: 4,
            background: 'var(--bg-primary)', border: '1px solid var(--border)',
            borderRadius: 6, boxShadow: '0 4px 16px rgba(0,0,0,0.2)',
          }}>
            <div role="button" tabIndex={0} onClick={() => { setSession(null); setShowMenu(false); }}
              onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setSession(null); setShowMenu(false); } }}
              style={{
                display: 'flex', alignItems: 'center', gap: 6, padding: '6px 10px',
                fontSize: 12, cursor: 'pointer', borderRadius: 4,
                color: !session ? colors.accent : 'var(--text-muted)',
                fontWeight: !session ? 600 : 400,
                background: !session ? `${colors.accent}08` : 'transparent',
              }}>
              <Globe size={12} /> Global
            </div>
            {recentSessions.map((s) => (
              <div key={s} role="button" tabIndex={0} onClick={() => { setSession(s); trackSession(s); setShowMenu(false); }}
                onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setSession(s); trackSession(s); setShowMenu(false); } }}
                style={{
                  display: 'flex', alignItems: 'center', gap: 6, padding: '6px 10px',
                  fontSize: 12, cursor: 'pointer', borderRadius: 4,
                  color: session === s ? colors.accent : 'var(--text-primary)',
                  fontWeight: session === s ? 600 : 400,
                  background: session === s ? `${colors.accent}10` : 'transparent',
                  fontFamily: 'monospace',
                }}>
                <Fingerprint size={12} /> {s.slice(0, 12)}
              </div>
            ))}
          </div>
        )}
      </div>

      <button type="button" onClick={() => setShowSettings(true)} style={iconBtn} title="Connection settings">
        <Settings size={14} />
      </button>

      <button type="button" onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')} style={iconBtn} title="Toggle theme">
        {theme === 'dark' ? <Sun size={14} /> : <Moon size={14} />}
      </button>

      {showSettings && (
        <div style={{ position: 'fixed', inset: 0, zIndex: 200, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <button type="button" aria-label="Close settings" onClick={() => setShowSettings(false)}
            style={{ position: 'fixed', inset: 0, border: 'none', padding: 0, cursor: 'default', background: 'rgba(0,0,0,0.4)' }} />
          <div ref={settingsRef} role="dialog" aria-modal="true" aria-label="Connection settings" tabIndex={-1}
            style={{
            position: 'relative', zIndex: 1,
            width: 380, padding: 20, borderRadius: 8, background: 'var(--bg-primary)', border: '1px solid var(--border)',
            boxShadow: '0 8px 32px rgba(0,0,0,0.3)', display: 'flex', flexDirection: 'column', gap: 12,
          }}>
            <h2 style={{ margin: 0, fontSize: 15, fontWeight: 600 }}>Connection Settings</h2>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              <label htmlFor="api-url-input" style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase' }}>API URL</label>
              <input id="api-url-input" value={apiUrlInput} onChange={(e) => setApiUrlInput(e.target.value)}
                placeholder="/api or http://host:port/api"
                style={{ padding: '8px 10px', fontSize: 12, borderRadius: 5, border: '1px solid var(--border)', background: 'var(--bg-primary)', color: 'var(--text-primary)', outline: 'none', fontFamily: 'monospace' }} />
            </div>
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <button type="button" onClick={() => { resetApiUrl(); setApiUrlInput(getApiUrl()); setShowSettings(false); }} style={btn('ghost', 'sm')}>Reset</button>
              <button type="button" onClick={() => setShowSettings(false)} style={btn('default', 'sm')}>Cancel</button>
              <button type="button" onClick={() => { setApiUrl(apiUrlInput); setShowSettings(false); window.location.reload(); }} style={btn('primary', 'sm')}>Save</button>
            </div>
          </div>
        </div>
      )}
    </header>
  );
}

function HealthDot({ ready }: Readonly<{ ready?: boolean }>) {
  const readyColor = ready ? colors.success : colors.error;
  const dotColor = ready === undefined ? '#64748b' : readyColor;
  return (
    <span style={{
      width: 6, height: 6, borderRadius: '50%', display: 'inline-block',
      background: dotColor,
    }} />
  );
}

const iconBtn = {
  display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
  width: 28, height: 28, borderRadius: 6,
  border: 'none', background: 'transparent', cursor: 'pointer',
  color: 'var(--text-secondary)',
} as const;
