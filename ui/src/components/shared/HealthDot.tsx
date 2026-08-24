import { colors } from '../../lib/theme';

export function HealthDot({ ready }: Readonly<{ ready?: boolean }>) {
  const dotColor = ready === undefined ? 'var(--text-muted)' : (ready ? colors.success : colors.error);
  return (
    <span style={{
      width: 6, height: 6, borderRadius: '50%', display: 'inline-block',
      background: dotColor,
    }} />
  );
}
