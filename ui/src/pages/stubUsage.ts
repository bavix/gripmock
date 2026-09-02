import { byTimestampDesc } from '../lib/format';
import type { CallRecord, Stub } from '../lib/types';

export type Usage = { total: number; first: Date; last: Date } | null;

export function computeUsage(
  history: CallRecord[] | undefined,
  stub: Pick<Stub, 'id'> | undefined,
): Usage {
  if (!history || !stub) return null;

  const calls = history.filter((h) => h.stubId === stub.id).sort(byTimestampDesc);
  if (calls.length === 0) return null;

  return {
    total: calls.length,
    first: new Date(calls[calls.length - 1].timestamp),
    last: new Date(calls[0].timestamp),
  };
}
