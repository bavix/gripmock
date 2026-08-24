import type { InspectCandidate, InspectReport, InspectStage } from '../lib/types';

export const STAGE_LABEL: Record<string, string> = {
  id: 'Stub ID lookup',
  service_method: 'Service / method',
  fallback_method: 'Method fallback',
  session: 'Session scope',
  times: 'Times limit',
  headers: 'Header matcher',
  input: 'Input matcher',
  selected: 'Selection',
};

export const stageLabel = (name: string): string => STAGE_LABEL[name] ?? name;

export interface ChainSummary {
  start: number;
  end: number;
  changes: { label: string; delta: number }[];
  text: string;
}

// The collapsed form of the chain: how many stubs entered, how many survived, and
// which stages moved the number. Stages that change nothing are noise here — they
// stay in the expanded step list. The method fallback *adds* candidates, so a
// summary that only counted removals would read as "3 → 5 · −3".
export function chainSummary(stages: InspectStage[]): ChainSummary {
  if (stages.length === 0) return { start: 0, end: 0, changes: [], text: 'no stages recorded' };

  const start = stages[0].before;
  const end = stages[stages.length - 1].after;
  const changes = stages
    .map((s) => ({ label: stageLabel(s.name), delta: s.after - s.before }))
    .filter((c) => c.delta !== 0);

  const shown = changes.slice(0, 2).map((c) => `${c.label} ${c.delta > 0 ? '+' : '−'}${Math.abs(c.delta)}`);
  if (changes.length > shown.length) shown.push(`+${changes.length - shown.length} more`);

  const tail = shown.length > 0 ? ` · ${shown.join(' · ')}` : '';

  return { start, end, changes, text: `${start} → ${end}${tail}` };
}

// One sentence for the top of the report: what happened, in the order a reader
// asks about it — who won, out of how many, and on what grounds.
export function explainOutcome(report: InspectReport, candidates: InspectCandidate[]): string {
  const total = candidates.length;
  const winner = candidates.find((c) => c.matched);

  if (!winner) {
    if (total === 0) return 'No stub is registered for this service and method.';

    const excluded = candidates.filter((c) => (c.excludedBy?.length ?? 0) > 0).length;
    const reason = excluded === total
      ? 'every candidate was excluded'
      : `${excluded} of ${total} candidates were excluded and none of the rest qualified`;

    return `No stub matched: ${reason}.`;
  }

  const rivals = candidates.filter((c) => c.id !== winner.id && (c.excludedBy?.length ?? 0) === 0);
  const via = report.fallbackToMethod ? ' after the method fallback' : '';

  if (rivals.length === 0) {
    return `${winner.id.slice(0, 8)} was the only candidate left${via}.`;
  }

  const runnerUp = rivals.reduce((best, c) => (c.priority > best.priority ? c : best), rivals[0]);
  const ground = winner.priority !== runnerUp.priority
    ? `higher priority (${winner.priority} vs ${runnerUp.priority})`
    : winner.specificity !== runnerUp.specificity
      ? `a more specific matcher (${winner.specificity} vs ${runnerUp.specificity} fields)`
      : `a higher score (${winner.score.toFixed(3)} vs ${runnerUp.score.toFixed(3)})`;

  return `${winner.id.slice(0, 8)} won over ${rivals.length} other qualified stub${rivals.length > 1 ? 's' : ''}${via} on ${ground}.`;
}
