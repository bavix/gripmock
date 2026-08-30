import { describe, it, expect } from 'vitest';
import { chainSummary, explainOutcome } from './inspectChain';
import type { InspectCandidate, InspectStage } from '../lib/types';

const stage = (name: string, before: number, after: number): InspectStage =>
  ({ name, before, after, removed: Math.max(before - after, 0) });

const candidate = (over: Partial<InspectCandidate>): InspectCandidate => ({
  id: 'aaaaaaaa-0000-0000-0000-000000000000', service: 's.S', method: 'M',
  priority: 0, times: 0, used: 0, specificity: 0, score: 0,
  visibleBySession: true, withinTimes: true, headersMatched: true, inputMatched: true,
  matched: false, ...over,
});

describe('chainSummary', () => {
  it('collapses to start, end and the stages that actually cut', () => {
    const s = chainSummary([stage('service_method', 10, 4), stage('session', 4, 4), stage('input', 4, 1)]);
    expect(s.start).toBe(10);
    expect(s.end).toBe(1);
    expect(s.text).toBe('10 → 1 · Service / method −6 · Input matcher −3');
  });

  it('names no stage when nothing was removed', () => {
    expect(chainSummary([stage('service_method', 2, 2)]).text).toBe('2 → 2');
  });

  it('shows a stage that adds candidates as a gain', () => {
    const s = chainSummary([stage('service_method', 3, 0), { name: 'fallback_method', before: 0, after: 5, removed: 0 }]);
    expect(s.text).toBe('3 → 5 · Service / method −3 · Method fallback +5');
  });

  it('folds a long list of cutting stages', () => {
    const s = chainSummary([stage('service_method', 9, 6), stage('session', 6, 4), stage('times', 4, 3), stage('input', 3, 1)]);
    expect(s.text).toContain('+2 more');
  });

  it('survives an empty report', () => {
    expect(chainSummary([]).text).toBe('no stages recorded');
  });
});

describe('explainOutcome', () => {
  const report = { service: 's.S', method: 'M' };

  it('names the ground the winner won on', () => {
    const winner = candidate({ id: 'winner00-0000', matched: true, priority: 10 });
    const rival = candidate({ id: 'rival000-0000', priority: 1 });
    expect(explainOutcome(report, [winner, rival])).toBe(
      'winner00 won over 1 other qualified stub on higher priority (10 vs 1).',
    );
  });

  it('falls back to specificity, then score', () => {
    const bySpec = explainOutcome(report, [
      candidate({ id: 'winner00-0000', matched: true, priority: 1, specificity: 3 }),
      candidate({ id: 'rival000-0000', priority: 1, specificity: 1 }),
    ]);
    expect(bySpec).toContain('more specific matcher (3 vs 1 fields)');

    const byScore = explainOutcome(report, [
      candidate({ id: 'winner00-0000', matched: true, priority: 1, specificity: 1, score: 2 }),
      candidate({ id: 'rival000-0000', priority: 1, specificity: 1, score: 1 }),
    ]);
    expect(byScore).toContain('higher score (2.000 vs 1.000)');
  });

  it('mentions the method fallback and the lone-candidate case', () => {
    const only = explainOutcome({ ...report, fallbackToMethod: true }, [candidate({ id: 'winner00-0000', matched: true })]);
    expect(only).toBe('winner00 was the only candidate left after the method fallback.');
  });

  it('explains a miss by how many candidates were excluded', () => {
    expect(explainOutcome(report, [])).toBe('No stub is registered for this service and method.');
    expect(explainOutcome(report, [candidate({ excludedBy: ['input'] })])).toBe(
      'No stub matched: every candidate was excluded.',
    );
    expect(explainOutcome(report, [candidate({ excludedBy: ['input'] }), candidate({ id: 'b' })])).toBe(
      'No stub matched: 1 of 2 candidates were excluded and none of the rest qualified.',
    );
  });
});
