import type { CSSProperties } from 'react';

// Inline-style palette. Kept in hex so the `${colors.x}18` alpha-suffix trick
// works. Values mirror the CSS custom properties in index.css so inline and
// class-based styling stay visually consistent.
export const colors = {
  accent: '#5570e6',
  accentHover: '#3f57d4',
  success: '#1fa650',
  successBg: 'rgba(31,166,80,0.14)',
  error: '#e5484d',
  errorBg: 'rgba(229,72,77,0.14)',
  warning: '#d97706',
  warningBg: 'rgba(217,119,6,0.14)',
} as const;

export const css = {
  flexCenter: { display: 'flex', alignItems: 'center', justifyContent: 'center' } as const,
  flexBetween: { display: 'flex', alignItems: 'center', justifyContent: 'space-between' } as const,
  truncate: { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } as const,
  mono: { fontFamily: 'var(--mono)' } as const,
  label: { fontSize: 11, fontWeight: 650, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.6px' } as CSSProperties,
  sectionHeader: { fontSize: 11, fontWeight: 650, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.6px' } as CSSProperties,
  badge: (bg: string, fg: string): CSSProperties => ({
    display: 'inline-flex', alignItems: 'center', gap: 4,
    fontSize: 11, fontWeight: 600, padding: '1px 7px', borderRadius: 5,
    background: bg, color: fg,
  }),
};

// Mirrors the .btn CSS class so `style={btn(...)}` call-sites match `className="btn"`.
export function btn(variant: 'primary' | 'danger' | 'ghost' | 'default' = 'default', size: 'sm' | 'md' = 'md'): CSSProperties {
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
  if (variant === 'danger') return { ...base, background: 'var(--error)', color: '#fff', borderColor: 'var(--error)' };
  if (variant === 'ghost') return { ...base, background: 'transparent', color: 'var(--text-secondary)', borderColor: 'transparent' };
  return { ...base, background: 'var(--bg-elevated)', color: 'var(--text)' };
}

export const inputStyle: CSSProperties = {
  padding: '7px 11px', fontSize: 13, borderRadius: 'var(--radius)',
  border: '1px solid var(--border-strong)', background: 'var(--bg)',
  color: 'var(--text)', outline: 'none',
};
