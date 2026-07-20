import type { ReactNode, CSSProperties } from 'react';

interface CardProps {
  children: ReactNode;
  style?: CSSProperties;
  className?: string;
  onClick?: () => void;
}

export function Card({ children, style, className, onClick }: CardProps) {
  const cls = ['card', onClick ? 'card-clickable' : '', className ?? ''].filter(Boolean).join(' ');
  return <div className={cls} style={style} onClick={onClick}>{children}</div>;
}

export function CardHeader({ children, style }: { children: ReactNode; style?: CSSProperties }) {
  return <div className="card-header" style={style}>{children}</div>;
}

export function CardBody({ children, style }: { children: ReactNode; style?: CSSProperties }) {
  return <div className="card-body" style={style}>{children}</div>;
}

export function SectionTitle({ children, style }: { children: string; style?: CSSProperties }) {
  return <div className="section-title" style={style}>{children}</div>;
}
