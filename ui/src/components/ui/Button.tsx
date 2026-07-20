import type { ReactNode, CSSProperties } from 'react';

interface BtnProps {
  children: ReactNode;
  onClick?: () => void;
  variant?: 'primary' | 'danger' | 'ghost' | 'default';
  size?: 'sm' | 'md';
  disabled?: boolean;
  style?: CSSProperties;
  title?: string;
  className?: string;
}

export function Button({ children, onClick, variant = 'default', size = 'sm', disabled, style, title, className }: BtnProps) {
  const cls = ['btn', variant !== 'default' ? `btn-${variant}` : '', className].filter(Boolean).join(' ');
  const sizeStyle = size === 'sm' ? { padding: '4px 10px', fontSize: 12 } : { padding: '6px 14px', fontSize: 13 };
  return (
    <button className={cls} onClick={onClick} disabled={disabled} title={title} style={{ ...sizeStyle, ...style }}>
      {children}
    </button>
  );
}
