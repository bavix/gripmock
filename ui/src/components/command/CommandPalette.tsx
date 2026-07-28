import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, LayoutDashboard, ListOrdered, Layers, History, Bug, ShieldCheck, Plus, ArrowRight, Hash, CornerDownLeft, Waypoints } from 'lucide-react';
import { useStubs } from '../../hooks/useStubs';
import { useServices } from '../../hooks/useServices';
import { compactPreview } from '../../lib/stub';
import { colors } from '../../lib/theme';
import type { Stub } from '../../lib/types';

const PAGES = [
  { id: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { id: '/stubs', icon: ListOrdered, label: 'Stubs' },
  { id: '/services', icon: Layers, label: 'Services' },
  { id: '/history', icon: History, label: 'History' },
  { id: '/inspect', icon: Bug, label: 'Inspect' },
  { id: '/verify', icon: ShieldCheck, label: 'Verify' },
  { id: '/stubs/create', icon: Plus, label: 'New Stub' },
];

type Item =
  | { kind: 'page'; id: string; label: string; icon: typeof LayoutDashboard }
  | { kind: 'method'; id: string; service: string; method: string }
  | { kind: 'stub'; id: string; stub: Stub };

export function CommandPalette() {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [active, setActive] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const { data: stubs } = useStubs();
  const { data: services } = useServices();

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setOpen((v) => !v); setQuery(''); setActive(0);
      }
      if (e.key === 'Escape') setOpen(false);
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  useEffect(() => { if (open) setTimeout(() => inputRef.current?.focus(), 40); }, [open]);

  // Client-side search: pages by label, stubs by id/service/method/payload.
  const items = useMemo<Item[]>(() => {
    const q = query.trim().toLowerCase();
    const pages: Item[] = PAGES
      .filter((p) => !q || p.label.toLowerCase().includes(q))
      .map((p) => ({ kind: 'page' as const, id: p.id, label: p.label, icon: p.icon }));
    if (q.length < 1) return pages;
    const methodItems: Item[] = [];
    for (const svc of services ?? []) {
      for (const m of svc.methods ?? []) {
        if (`${svc.id}/${m.name}`.toLowerCase().includes(q)) methodItems.push({ kind: 'method', id: `${svc.id}/${m.name}`, service: svc.id, method: m.name });
        if (methodItems.length >= 12) break;
      }
    }
    const matchStub = (s: Stub) =>
      [s.id, s.service, s.method, compactPreview(s.input, 9999), compactPreview(s.output, 9999)]
        .some((f) => f.toLowerCase().includes(q));
    const stubItems: Item[] = (stubs ?? []).filter(matchStub).slice(0, 20)
      .map((s) => ({ kind: 'stub' as const, id: s.id, stub: s }));
    return [...pages, ...methodItems, ...stubItems];
  }, [query, stubs, services]);

  useEffect(() => { setActive(0); }, [query]);

  const go = useCallback((item: Item) => {
    setOpen(false);
    if (item.kind === 'page') navigate(item.id);
    else if (item.kind === 'method') navigate(`/stubs?service=${encodeURIComponent(item.service)}&method=${encodeURIComponent(item.method)}`);
    else navigate(`/stubs/${item.id}`);
  }, [navigate]);

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); setActive((a) => Math.min(a + 1, items.length - 1)); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setActive((a) => Math.max(a - 1, 0)); }
    else if (e.key === 'Enter') { e.preventDefault(); if (items[active]) go(items[active]); }
  };

  useEffect(() => {
    if (!open) return;
    const el = listRef.current?.querySelector(`[data-idx="${active}"]`);
    (el as HTMLElement | null)?.scrollIntoView({ block: 'nearest' });
  }, [active, open]);

  if (!open) return null;

  const pageItems = items.filter((i) => i.kind === 'page');
  const methodItems = items.filter((i) => i.kind === 'method');
  const stubItems = items.filter((i) => i.kind === 'stub');

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 300,
      display: 'flex', alignItems: 'flex-start', justifyContent: 'center', paddingTop: '12vh',
    }}>
      <button type="button" aria-label="Close command palette" onClick={() => setOpen(false)}
        style={{ position: 'fixed', inset: 0, border: 'none', padding: 0, cursor: 'default', background: 'rgba(6,10,20,0.55)' }} />
      <div role="dialog" aria-modal="true" aria-label="Command palette" style={{
        position: 'relative', zIndex: 1,
        width: 560, maxWidth: '92vw', background: 'var(--bg-elevated)', borderRadius: 'var(--radius-xl)',
        border: '1px solid var(--border)', boxShadow: 'var(--shadow-lg)', overflow: 'hidden',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '12px 16px', borderBottom: '1px solid var(--border)' }}>
          <Search size={16} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
          <input ref={inputRef} value={query} onChange={(e) => setQuery(e.target.value)} onKeyDown={onKeyDown}
            role="combobox" aria-expanded="true" aria-controls="cmdk-list" aria-activedescendant={items[active] ? `cmdk-opt-${active}` : undefined}
            placeholder="Jump to a page or search stubs by ID, endpoint, or payload…"
            style={{ flex: 1, border: 'none', outline: 'none', fontSize: 14.5, background: 'transparent', color: 'var(--text)' }} />
          <span className="kbd">ESC</span>
        </div>

        <div ref={listRef} id="cmdk-list" role="listbox" style={{ maxHeight: 380, overflow: 'auto', padding: 6 }}>
          {pageItems.length > 0 && <div style={groupLabel}>Pages</div>}
          {pageItems.map((item) => {
            const idx = items.indexOf(item);
            const Icon = (item as Extract<Item, { kind: 'page' }>).icon;
            return (
              <Row key={item.id} item={item} idx={idx} active={active} onSelect={go} onHover={setActive}>
                <Icon size={15} style={{ color: colors.accent, flexShrink: 0 }} />
                <span style={{ flex: 1 }}>{(item as Extract<Item, { kind: 'page' }>).label}</span>
              </Row>
            );
          })}

          {methodItems.length > 0 && <div style={groupLabel}>Methods ({methodItems.length})</div>}
          {methodItems.map((item) => {
            const idx = items.indexOf(item);
            const m = item as Extract<Item, { kind: 'method' }>;
            return (
              <Row key={item.id} item={item} idx={idx} active={active} onSelect={go} onHover={setActive}>
                <Waypoints size={14} style={{ color: colors.accent, flexShrink: 0 }} />
                <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}><span style={{ color: 'var(--text-muted)' }}>{m.service}/</span>{m.method}</span>
                <ArrowRight size={13} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
              </Row>
            );
          })}

          {stubItems.length > 0 && <div style={groupLabel}>Stubs ({stubItems.length})</div>}
          {stubItems.map((item) => {
            const idx = items.indexOf(item);
            const s = (item as Extract<Item, { kind: 'stub' }>).stub;
            return (
              <Row key={item.id} item={item} idx={idx} active={active} onSelect={go} onHover={setActive}>
                <Hash size={13} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
                <code style={{ fontSize: 11, color: 'var(--text-muted)', flexShrink: 0 }}>{s.id.slice(0, 8)}</code>
                <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{s.service}/{s.method}</span>
                <ArrowRight size={13} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
              </Row>
            );
          })}

          {items.length === 0 && (
            <div style={{ padding: 22, textAlign: 'center', color: 'var(--text-muted)', fontSize: 13 }}>
              No matches for “{query}”.
            </div>
          )}
        </div>

        <div style={{ padding: '8px 14px', borderTop: '1px solid var(--border)', display: 'flex', gap: 14, fontSize: 11.5, color: 'var(--text-muted)' }}>
          <span style={hint}><span className="kbd">↑</span><span className="kbd">↓</span> Navigate</span>
          <span style={hint}><span className="kbd"><CornerDownLeft size={11} /></span> Open</span>
          <span style={hint}><span className="kbd">Esc</span> Close</span>
        </div>
      </div>
    </div>
  );
}

function Row({ item, idx, active, onSelect, onHover, children }: Readonly<{
  item: Item; idx: number; active: number;
  onSelect: (item: Item) => void; onHover: (idx: number) => void; children: React.ReactNode;
}>) {
  return (
    <div data-idx={idx} id={`cmdk-opt-${idx}`} role="option" aria-selected={idx === active} tabIndex={0}
      onClick={() => onSelect(item)} onMouseEnter={() => onHover(idx)} onFocus={() => onHover(idx)}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onSelect(item); } }}
      style={{
        display: 'flex', alignItems: 'center', gap: 8, padding: '7px 10px', borderRadius: 'var(--radius)',
        cursor: 'pointer', fontSize: 13, background: idx === active ? 'var(--accent-bg)' : 'transparent',
        color: idx === active ? 'var(--accent-text)' : 'var(--text)',
      }}>
      {children}
    </div>
  );
}

const groupLabel: React.CSSProperties = { padding: '6px 10px 3px', fontSize: 11, fontWeight: 650, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px' };
const hint: React.CSSProperties = { display: 'inline-flex', alignItems: 'center', gap: 4 };
