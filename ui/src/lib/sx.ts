import type { CSSProperties } from 'react';

export const sx = {
  card: (): CSSProperties => ({
    borderRadius: 8,
    border: '1px solid var(--border)',
    background: 'var(--bg-secondary)',
    overflow: 'hidden',
  }),

  cardBody: (): CSSProperties => ({
    padding: 12,
  }),

  cardHeader: (): CSSProperties => ({
    padding: '8px 12px',
    borderBottom: '1px solid var(--border)',
    fontSize: 10,
    fontWeight: 600,
    color: 'var(--text-muted)',
    textTransform: 'uppercase',
    letterSpacing: '0.4px',
    display: 'flex',
    alignItems: 'center',
    gap: 6,
  }),

  input: (w?: number): CSSProperties => ({
    padding: '6px 10px',
    fontSize: 12,
    borderRadius: 6,
    border: '1px solid var(--border)',
    background: 'var(--bg-primary)',
    color: 'var(--text-primary)',
    outline: 'none',
    width: w ?? '100%',
  }),

  pageTitle: (): CSSProperties => ({
    fontSize: 16,
    fontWeight: 600,
    margin: 0,
  }),

  sectionTitle: (): CSSProperties => ({
    fontSize: 10,
    fontWeight: 600,
    color: 'var(--text-muted)',
    textTransform: 'uppercase',
    letterSpacing: '0.4px',
    marginBottom: 6,
  }),

  label: (): CSSProperties => ({
    fontSize: 10,
    fontWeight: 600,
    color: 'var(--text-muted)',
    textTransform: 'uppercase',
    letterSpacing: '0.3px',
    display: 'block',
    marginBottom: 2,
  }),

  flexRow: (gap = 8): CSSProperties => ({
    display: 'flex',
    alignItems: 'center',
    gap,
    flexWrap: 'wrap',
  }),

  chip: (fg: string, bg = `${fg}18`): CSSProperties => ({
    fontSize: 9,
    padding: '1px 6px',
    borderRadius: 4,
    fontWeight: 600,
    background: bg,
    color: fg,
    whiteSpace: 'nowrap',
  }),

  jsonBlock: (maxHeight = 200): CSSProperties => ({
    fontSize: 11,
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
    padding: 10,
    borderRadius: 6,
    border: '1px solid var(--border)',
    background: 'var(--bg-primary)',
    overflow: 'auto',
    maxHeight,
    margin: 0,
    lineHeight: 1.5,
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
  }),

  hoverRow: (): CSSProperties => ({
    cursor: 'pointer',
    transition: 'background 0.1s ease',
  }),
};
