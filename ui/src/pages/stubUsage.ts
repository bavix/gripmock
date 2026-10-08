import type { CallRecord, Stub } from '../lib/types';

export type Usage = { total: number; first: Date; last: Date } | null;

export function computeUsage(
  history: CallRecord[] | undefined,
  stub: Pick<Stub, 'id'> | undefined,
): Usage {
  if (!history || !stub) return null;

  let total = 0;
  let first = 0;
  let last = 0;

  for (const call of history) {
    if (call.stubId !== stub.id) continue;
    const at = new Date(call.timestamp).getTime();
    if (total === 0 || at < first) first = at;
    if (total === 0 || at > last) last = at;
    total++;
  }
  if (total === 0) return null;

  return { total, first: new Date(first), last: new Date(last) };
}
