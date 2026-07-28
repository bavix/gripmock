import type { CSSProperties } from 'react';
import { colors } from '../../lib/theme';
import { grpcCodeName } from '../../lib/grpc';
import { isCallOk, type CallRecord } from '../../lib/types';

export function CallStatusBadge({ call, style }: Readonly<{ call: Pick<CallRecord, 'code' | 'error'>; style?: CSSProperties }>) {
  const ok = isCallOk(call);
  return (
    <span className="badge" title={ok ? undefined : call.error || ''}
      style={{ background: ok ? 'var(--success-bg)' : 'var(--error-bg)', color: ok ? colors.success : colors.error, ...style }}>
      {ok ? 'OK' : grpcCodeName(call.code)}
    </span>
  );
}
