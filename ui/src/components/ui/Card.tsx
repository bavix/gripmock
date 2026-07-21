import type { ReactNode, CSSProperties } from 'react';

interface CardProps {
  children: ReactNode;
  style?: CSSProperties;
  className?: string;
  onClick?: () => void;
}

export function Card({ children, style, className, onClick }: Readonly<CardProps>) {
  const cls = ['card', onClick ? 'card-clickable' : '', className ?? ''].filter(Boolean).join(' ');
  if (onClick) {
    return (
      <div
        className={cls}
        style={style}
        role="button"
        tabIndex={0}
        onClick={onClick}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            onClick();
          }
        }}
      >
        {children}
      </div>
    );
  }
  return <div className={cls} style={style}>{children}</div>;
}

export function CardHeader({ children, style }: Readonly<{ children: ReactNode; style?: CSSProperties }>) {
  return <div className="card-header" style={style}>{children}</div>;
}

export function CardBody({ children, style }: Readonly<{ children: ReactNode; style?: CSSProperties }>) {
  return <div className="card-body" style={style}>{children}</div>;
}

export function SectionTitle({ children, style }: Readonly<{ children: string; style?: CSSProperties }>) {
  return <div className="section-title" style={style}>{children}</div>;
}
