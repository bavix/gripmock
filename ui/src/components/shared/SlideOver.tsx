import { type ReactNode } from 'react';
import { X } from 'lucide-react';
import { useFocusTrap } from '../../hooks/useFocusTrap';

interface SlideOverProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  width?: string;
}

export function SlideOver({ open, onClose, title, children, width = '640px' }: Readonly<SlideOverProps>) {
  const ref = useFocusTrap<HTMLDivElement>(open, onClose);

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 150,
      pointerEvents: open ? 'auto' : 'none',
    }}>
      {open && (
        <div
          role="button"
          tabIndex={0}
          aria-label="Close"
          onClick={onClose}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              onClose();
            }
          }}
          style={{ position: 'absolute', inset: 0, background: 'rgba(0,0,0,0.3)' }}
        />
      )}

      <div ref={ref} role="dialog" aria-modal="true" aria-label={title} tabIndex={-1} style={{
        position: 'absolute', top: 44, right: 0, bottom: 24,
        width: width, maxWidth: '100vw',
        background: 'var(--bg)',
        borderLeft: '1px solid var(--border)',
        boxShadow: '-4px 0 24px rgba(0,0,0,0.12)',
        display: 'flex', flexDirection: 'column',
        transform: open ? 'translateX(0)' : 'translateX(100%)',
        transition: 'transform 0.2s ease',
      }}>
        <div style={{
          display: 'flex', alignItems: 'center', gap: 8,
          padding: '10px 14px', borderBottom: '1px solid var(--border)',
          flexShrink: 0, background: 'var(--bg-secondary)',
        }}>
          <span style={{ fontSize: 13, fontWeight: 600, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{title}</span>
          <button type="button" onClick={onClose} className="btn btn-ghost" style={{ padding: '2px 6px' }} aria-label="Close"><X size={15} /></button>
        </div>

        <div style={{ flex: 1, overflow: 'auto', padding: 14 }}>
          {children}
        </div>
      </div>
    </div>
  );
}
