import type { CSSProperties } from 'react';

// Inline-style palette. Kept in hex so the `${colors.x}18` alpha-suffix trick
// works. Values mirror the CSS custom properties in index.css so inline and
// class-based styling stay visually consistent.
export const colors = {
  accent: '#5570e6',
  success: '#1fa650',
  error: '#e5484d',
  warning: '#d97706',
} as const;

// Mirrors the .btn CSS class so `style={btn(...)}` call-sites match `className="btn"`.
export function btn(variant: 'primary' | 'ghost' | 'default' = 'default', size: 'sm' | 'md' = 'md'): CSSProperties {
  const pad = size === 'sm' ? { padding: '4px 9px' } : { padding: '6px 12px' };
  const fs = size === 'sm' ? 11.5 : 12.5;
  const base: CSSProperties = {
    display: 'inline-flex', alignItems: 'center', justifyContent: 'center', gap: size === 'sm' ? 4 : 6,
    fontWeight: 550, fontSize: fs, borderRadius: 'var(--radius)', cursor: 'pointer',
    transition: 'background 0.12s, border-color 0.12s, color 0.12s', userSelect: 'none', whiteSpace: 'nowrap',
    outline: 'none', textDecoration: 'none', lineHeight: 1.35,
    border: '1px solid var(--border-strong)',
    ...pad,
  };
  if (variant === 'primary') return { ...base, background: 'var(--accent)', color: '#fff', borderColor: 'var(--accent)', boxShadow: 'var(--shadow-sm)' };
  if (variant === 'ghost') return { ...base, background: 'transparent', color: 'var(--text-secondary)', borderColor: 'transparent' };
  return { ...base, background: 'var(--bg-elevated)', color: 'var(--text)' };
}
