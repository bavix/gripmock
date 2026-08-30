import { useState, useRef, useEffect } from 'react';
import { Menu, FlaskConical, Sun, Moon, Fingerprint, Globe, Settings, Check } from 'lucide-react';
import { useStore } from '../../lib/store';
import { getApiUrl, setApiUrl, resetApiUrl } from '../../lib/api';
import { useFocusTrap } from '../../hooks/useFocusTrap';
import { useReadiness } from '../../hooks/useReadiness';
import { useSessions } from '../../hooks/useSessions';
import { colors, btn } from '../../lib/theme';
import { HealthDot } from '../shared/HealthDot';
import { versionLabel } from '../../lib/format';
import { sessionOptions } from './sessionList';

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
  const [sessionInput, setSessionInput] = useState('');
  const menuRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const sessionInputRef = useRef<HTMLInputElement>(null);
  const settingsRef = useFocusTrap<HTMLDivElement>(showSettings, () => setShowSettings(false));

  const { dash, ready, offline, label: healthLabel } = useReadiness();
  const { data: liveSessions } = useSessions();

  const closeMenu = (restoreFocus = true) => {
    setShowMenu(false);
    setSessionInput('');
    if (restoreFocus) triggerRef.current?.focus();
  };

  useEffect(() => {
    if (!showMenu) return;

    const onPointer = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) closeMenu(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') closeMenu();
    };

    document.addEventListener('mousedown', onPointer);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onPointer);
      document.removeEventListener('keydown', onKey);
    };
  }, [showMenu]);

  useEffect(() => {
    if (showMenu) sessionInputRef.current?.focus();
  }, [showMenu]);

  const pick = (id: string | null) => {
    setSession(id);
    if (id) trackSession(id);
    closeMenu();
  };

  const options = sessionOptions(recentSessions, liveSessions?.sessions ?? [], session);
  const errors = dash?.historyErrors ?? 0;

  return (
    <header style={{
      height: 44, borderBottom: '1px solid var(--border)',
      display: 'flex', alignItems: 'center', padding: '0 8px', gap: 8,
      background: 'var(--bg-secondary)', flexShrink: 0,
    }}>
      <button type="button" onClick={onToggleSidebar} className="icon-btn" title="Toggle sidebar" aria-label="Toggle sidebar">
        <Menu size={16} />
      </button>

      <div style={{ display: 'flex', alignItems: 'center', gap: 6, minWidth: 0 }}>
        <FlaskConical size={16} color={colors.accent} />
        <span style={{ fontWeight: 600, fontSize: 13 }}>GripMock</span>
        {dash?.version && (
          <span style={{ fontSize: 11, color: 'var(--text-muted)', fontWeight: 400 }}>{versionLabel(dash.version)}</span>
        )}
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }} title={healthLabel}>
          <HealthDot ready={ready} />
          {offline && <span style={{ fontSize: 11, color: colors.error, fontWeight: 500 }}>offline</span>}
        </span>
      </div>

      <div style={{ flex: 1 }} />

      <div className="topbar-counters" style={{ fontSize: 11, color: 'var(--text-muted)' }}
        title={`${dash?.totalStubs ?? 0} stubs loaded, ${dash?.totalHistory ?? 0} calls recorded${errors > 0 ? `, ${errors} of them failed` : ''}`}>
        <span><b style={{ color: 'var(--text-secondary)', fontWeight: 600 }}>{dash?.totalStubs ?? 0}</b> stubs</span>
        <span><b style={{ color: 'var(--text-secondary)', fontWeight: 600 }}>{dash?.totalHistory ?? 0}</b> calls</span>
        {errors > 0 && (
          <span style={{ color: colors.error, fontWeight: 600 }}>{errors} errs</span>
        )}
      </div>

      <div ref={menuRef} style={{ position: 'relative' }}>
        <button type="button" ref={triggerRef}
          onClick={() => (showMenu ? closeMenu() : setShowMenu(true))}
          aria-haspopup="dialog" aria-expanded={showMenu}
          title={session ? `Session ${session}` : 'Global scope — no session header sent'}
          style={{
            display: 'flex', alignItems: 'center', gap: 4,
            padding: '3px 8px', fontSize: 11, borderRadius: 5,
            border: `1px solid ${session ? `${colors.accent}50` : 'var(--border)'}`,
            background: session ? `${colors.accent}10` : 'transparent',
            color: session ? colors.accent : 'var(--text-muted)',
            cursor: 'pointer', fontWeight: 500,
          }}>
          {session ? <Fingerprint size={12} /> : <Globe size={12} />}
          <span style={{ maxWidth: 110, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {session ?? 'Global'}
          </span>
        </button>

        {showMenu && (
          <div role="dialog" aria-label="Session scope" style={{
            position: 'absolute', top: '100%', right: 0, zIndex: 100,
            minWidth: 220, marginTop: 4, padding: 4,
            background: 'var(--bg-primary)', border: '1px solid var(--border)',
            borderRadius: 6, boxShadow: '0 4px 16px rgba(0,0,0,0.2)',
          }}>
            <input ref={sessionInputRef} value={sessionInput} aria-label="Session id"
              onChange={(e) => setSessionInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key !== 'Enter') return;
                // Focus returns to the trigger during pick(); without this the
                // Enter default action lands on that button and reopens the menu.
                e.preventDefault();
                const id = sessionInput.trim();
                if (id) pick(id);
              }}
              placeholder="Session id, Enter to switch"
              style={{
                width: '100%', boxSizing: 'border-box', marginBottom: 4,
                padding: '6px 8px', fontSize: 12, fontFamily: 'monospace',
                borderRadius: 4, border: '1px solid var(--border)',
                background: 'var(--bg-secondary)', color: 'var(--text-primary)', outline: 'none',
              }} />

            <SessionRow icon={<Globe size={12} />} label="Global" active={!session} onPick={() => pick(null)} />

            {options.map((id) => (
              <SessionRow key={id} icon={<Fingerprint size={12} />} label={id} mono
                active={session === id} onPick={() => pick(id)} />
            ))}
          </div>
        )}
      </div>

      <button type="button" onClick={() => setShowSettings(true)} className="icon-btn" title="Connection settings" aria-label="Connection settings">
        <Settings size={14} />
      </button>

      <button type="button" onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')} className="icon-btn"
        title={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
        aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}>
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

function SessionRow({ icon, label, active, mono, onPick }: Readonly<{
  icon: React.ReactNode; label: string; active: boolean; mono?: boolean; onPick: () => void;
}>) {
  return (
    <button type="button" aria-current={active ? 'true' : undefined} onClick={onPick} title={label}
      style={{
        display: 'flex', alignItems: 'center', gap: 6, width: '100%',
        padding: '6px 10px', fontSize: 12, cursor: 'pointer', borderRadius: 4,
        border: 'none', textAlign: 'left',
        color: active ? colors.accent : 'var(--text-primary)',
        fontWeight: active ? 600 : 400,
        background: active ? `${colors.accent}10` : 'transparent',
        fontFamily: mono ? 'monospace' : 'inherit',
      }}>
      {icon}
      <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{label}</span>
      {active && <Check size={12} />}
    </button>
  );
}
