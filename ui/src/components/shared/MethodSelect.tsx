import { useState, useRef, useEffect, useMemo, useId } from 'react';
import { useServices, useServiceMethods } from '../../hooks/useServices';
import { Search, ChevronDown, Loader2 } from 'lucide-react';

interface MethodSelectProps {
  service: string;
  method: string;
  onServiceChange: (s: string) => void;
  onMethodChange: (m: string) => void;
}

export function MethodSelect({ service, method, onServiceChange, onMethodChange }: Readonly<MethodSelectProps>) {
  const listboxId = useId();
  const { data: allServices } = useServices();
  const { data: serviceMethods, isFetching: methodsLoading } = useServiceMethods(service || null);
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const ref = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => { if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false); };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  useEffect(() => { if (open) { setSearch(''); setTimeout(() => inputRef.current?.focus(), 50); } }, [open]);

  // Scroll to selected item when dropdown opens
  useEffect(() => {
    if (!open || !listRef.current) return;
    const selected = listRef.current.querySelector('[data-selected="true"]');
    if (selected) selected.scrollIntoView({ block: 'nearest' });
  }, [open]);

  // Show methods from the service list. For the selected service, merge in
  // schemas from the dedicated endpoint (if loaded) but never replace the list.
  const grouped = useMemo(() => {
    if (!allServices) return [];
    const q = search.toLowerCase();
    return allServices
      .map((s) => {
        const base = s.methods || [];
        let methods = base;
        if (s.id === service && serviceMethods) {
          const byName = new Map(serviceMethods.map((m) => [m.name, m]));
          methods = base.map((m) => byName.get(m.name) || m);
        }
        return {
          id: s.id,
          methods: methods.filter((m) => (m.name || '').toLowerCase().includes(q)),
        };
      })
      .filter((s) => s.id.toLowerCase().includes(q) || s.methods.length > 0);
  }, [allServices, serviceMethods, service, search]);

  const displayLabel = service && method ? `${service}/${method}` : service || method || '';

  // Flat list for keyboard navigation across the grouped options.
  const flat = useMemo(() => grouped.flatMap((g) => g.methods.map((m) => ({ service: g.id, method: m.name }))), [grouped]);
  const [active, setActive] = useState(0);
  useEffect(() => { setActive(0); }, [search, open]);
  const flatIndex = (svc: string, m: string) => flat.findIndex((o) => o.service === svc && o.method === m);
  const choose = (o: { service: string; method: string }) => { onServiceChange(o.service); onMethodChange(o.method); setOpen(false); };
  const onKey = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); setActive((a) => Math.min(a + 1, flat.length - 1)); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setActive((a) => Math.max(a - 1, 0)); }
    else if (e.key === 'Enter') { e.preventDefault(); if (flat[active]) choose(flat[active]); }
    else if (e.key === 'Escape') { e.preventDefault(); setOpen(false); }
  };
  useEffect(() => {
    if (!open) return;
    listRef.current?.querySelector('[data-active="true"]')?.scrollIntoView({ block: 'nearest' });
  }, [active, open]);

  return (
    <div ref={ref} style={{ position: 'relative' }}>
      <div
        onClick={() => setOpen((v) => !v)}
        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setOpen((v) => !v); } }}
        role="combobox" tabIndex={0} aria-expanded={open} aria-controls={listboxId} aria-haspopup="listbox" aria-label="Service and method"
        className="input"
        style={{
          display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer',
          padding: '8px 12px', fontSize: 13,
          fontFamily: displayLabel ? 'ui-monospace, monospace' : undefined,
          color: displayLabel ? 'var(--text)' : 'var(--text-muted)',
        }}>
        <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {displayLabel || 'Select service/method...'}
        </span>
        <ChevronDown size={13} style={{ flexShrink: 0, color: 'var(--text-muted)', opacity: 0.6 }} />
      </div>

      {open && (
        <div style={{
          position: 'absolute', top: '100%', left: 0, right: 0, zIndex: 200,
          marginTop: 2, maxHeight: 400, overflow: 'auto',
          border: '1px solid var(--border)', borderRadius: 6,
          background: 'var(--bg)', boxShadow: '0 6px 20px rgba(0,0,0,0.25)',
        }}>
          <div style={{
            position: 'sticky', top: 0, zIndex: 1,
            padding: '6px 8px', borderBottom: '1px solid var(--border)',
            background: 'var(--bg)',
          }}>
            <div style={{
              display: 'flex', alignItems: 'center', gap: 4,
              borderRadius: 4, padding: '4px 8px',
              border: '1px solid var(--border)',
            }}>
              <Search size={12} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
              <input ref={inputRef} value={search} onChange={(e) => setSearch(e.target.value)} onKeyDown={onKey}
                placeholder="Filter… (↑↓ to navigate, ↵ to select)"
                style={{ border: 'none', outline: 'none', background: 'none', color: 'var(--text)', fontSize: 12, width: '100%' }} />
            </div>
          </div>

          <div ref={listRef} id={listboxId} role="listbox" aria-label="Methods">
            {!allServices && (
              <div style={{ padding: 16, textAlign: 'center', fontSize: 12, color: 'var(--text-muted)', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6 }}>
                <Loader2 size={12} className="animate-spin" /> Loading...
              </div>
            )}
            {allServices && grouped.length === 0 && (
              <div style={{ padding: 16, textAlign: 'center', fontSize: 12, color: 'var(--text-muted)' }}>No services found</div>
            )}
            {methodsLoading && service && (
              <div style={{ padding: '4px 10px', fontSize: 11, color: 'var(--text-muted)', display: 'flex', alignItems: 'center', gap: 4, justifyContent: 'center' }}>
                <Loader2 size={10} className="animate-spin" /> Loading methods...
              </div>
            )}

            {grouped.map((g) => (
              <div key={g.id}>
                <div style={{
                  padding: '4px 10px', fontSize: 11, fontWeight: 600, textTransform: 'uppercase',
                  letterSpacing: '0.5px', color: 'var(--text-muted)',
                  background: 'var(--bg-secondary)', borderBottom: '1px solid var(--border)',
                  position: 'sticky', top: 41, zIndex: 1,
                }}>
                  {g.id}
                </div>
                {g.methods.map((m) => {
                  const isSelected = service === g.id && method === m.name;
                  const isActive = flatIndex(g.id, m.name) === active;
                  return (
                    <div key={m.name} onClick={() => choose({ service: g.id, method: m.name })}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); choose({ service: g.id, method: m.name }); } }}
                      role="option" tabIndex={0} aria-selected={isSelected}
                      data-selected={isSelected} data-active={isActive}
                      onMouseEnter={() => setActive(flatIndex(g.id, m.name))}
                      style={{
                        display: 'flex', alignItems: 'center', gap: 6,
                        padding: '6px 10px', fontSize: 12, cursor: 'pointer',
                        background: isActive ? 'var(--accent-bg)' : 'transparent',
                        color: isSelected || isActive ? 'var(--accent-text)' : 'var(--text)',
                        fontFamily: 'monospace',
                        borderBottom: '1px solid var(--border)',
                      }}>
                      <span style={{ flex: 1 }}>{m.name}</span>
                      {isSelected && <span style={{ fontSize: 10, color: 'var(--accent-text)' }}>✓</span>}
                      <StreamBadge type={m.methodType} />
                    </div>
                  );
                })}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function StreamBadge({ type }: Readonly<{ type: string }>) {
  const cfg: Record<string, { label: string; color: string }> = {
    unary: { label: 'U', color: '#3b82f6' },
    client_streaming: { label: 'CS', color: '#f59e0b' },
    server_streaming: { label: 'SS', color: '#a855f7' },
    bidi_streaming: { label: 'BD', color: '#ef4444' },
  };
  const c = cfg[type] || { label: '?', color: 'var(--text-muted)' };
  return <span style={{ fontSize: 11, padding: '1px 5px', borderRadius: 3, fontWeight: 700, background: `${c.color}18`, color: c.color }}>{c.label}</span>;
}
