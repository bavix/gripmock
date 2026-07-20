import { useState, useMemo } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { MethodSelect } from '../components/shared/MethodSelect';
import { useVerify } from '../hooks/useVerify';
import { useScopedHistory } from '../hooks/useHistory';
import { useDashboard } from '../hooks/useDashboard';
import { AlertTriangle } from 'lucide-react';
import type { ApiError } from '../lib/api';
import { useStore } from '../lib/store';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Fingerprint, Globe, RotateCcw, ExternalLink } from 'lucide-react';
import { colors } from '../lib/theme';

const PRESETS = [1, 5, 10, 50, 100];

export function VerifyPage() {
  const navigate = useNavigate();
  const session = useStore((s) => s.session);
  const verify = useVerify();
  const { data: dash } = useDashboard();
  const [params] = useSearchParams();
  const [service, setService] = useState(params.get('service') || '');
  const [method, setMethod] = useState(params.get('method') || '');
  const [count, setCount] = useState(1);
  const [result, setResult] = useState<{ ok: boolean; msg: string; expected?: number; actual?: number } | null>(null);

  // Evidence: the endpoint's calls, fetched with the server-side scope filter
  // (only after a verify run) so the list matches the server's count basis.
  const { data: scoped } = useScopedHistory(service, method, !!result);
  const actualCalls = useMemo(() => {
    if (!result || !scoped) return [];
    return [...scoped].sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
  }, [result, scoped]);

  const handleVerify = async () => {
    setResult(null);
    try {
      await verify.mutateAsync({ service, method, expectedCount: count });
      setResult({ ok: true, msg: 'Verification passed', expected: count, actual: count });
    } catch (err) {
      const e = err as ApiError;
      const body = e.body || {};
      // Prefer structured {expected, actual} from the 400 body; fall back to parsing the message.
      const m = e.message.match(/called (\d+) times, got (\d+)/i) || e.message.match(/expected (\d+).*?(\d+)/i);
      const expected = typeof body.expected === 'number' ? body.expected : (m ? Number(m[1]) : count);
      const actual = typeof body.actual === 'number' ? body.actual : (m ? Number(m[2]) : undefined);
      setResult({ ok: false, msg: e.message, expected, actual });
    }
  };

  const presetBtn = (n: number) => ({
    padding: '4px 12px', fontSize: 11, borderRadius: 4, cursor: 'pointer', minWidth: 36, textAlign: 'center' as const,
    border: count === n ? '1px solid var(--accent)' : '1px solid var(--border)',
    background: count === n ? `${colors.accent}15` : 'transparent',
    color: count === n ? colors.accent : 'var(--text-muted)',
    fontWeight: count === n ? 600 : 400,
  });

  return (
    <div className="page-enter" style={{ display: 'flex', flexDirection: 'column', gap: 10, maxWidth: 540 }}>
      <h1 style={{ fontSize: 16, fontWeight: 600, margin: 0 }}>Verify Calls</h1>
      <p style={{ fontSize: 11, color: 'var(--text-muted)', margin: 0, lineHeight: 1.5 }}>
        Verify that a specific endpoint received the expected number of calls.
        Results are scoped to the active session.
      </p>

      {dash && !dash.historyEnabled && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px', borderRadius: 'var(--radius)', border: `1px solid ${colors.warning}40`, background: 'var(--warning-bg)', color: colors.warning, fontSize: 12.5 }}>
          <AlertTriangle size={14} /> Verify needs call history, which is disabled. Every check will report 0 actual calls. Enable with <code>HISTORY_ENABLED=true</code>.
        </div>
      )}

      <Card>
        <div className="card-header">Endpoint & Count</div>
        <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <MethodSelect service={service} method={method} onServiceChange={(s) => { setService(s); setMethod(''); setResult(null); }} onMethodChange={(m) => { setMethod(m); setResult(null); }} />

          <div>
            <div className="section-title" style={{ marginBottom: 4 }}>Expected calls</div>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <input type="number" min={0} value={count} onChange={(e) => setCount(Number(e.target.value))} className="input" style={{ width: 80 }} />
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>calls</span>
              <div style={{ display: 'flex', gap: 4, marginLeft: 8 }}>
                {PRESETS.map((n) => (
                  <button key={n} onClick={() => setCount(n)} style={presetBtn(n)}>{n}</button>
                ))}
              </div>
            </div>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '6px 10px', borderRadius: 6, background: 'var(--bg)', border: '1px solid var(--border)' }}>
            {session ? <Fingerprint size={12} color={colors.accent} /> : <Globe size={12} color={colors.success} />}
            <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>
              Session: <strong style={{ color: session ? colors.accent : colors.success }}>{session ? session.slice(0, 16) : 'Global (no session)'}</strong>
            </span>
            <div style={{ flex: 1 }} />
            <Button variant="primary" onClick={handleVerify} disabled={!service || !method || verify.isPending}>
              {verify.isPending ? 'Verifying...' : 'Verify'}
            </Button>
          </div>
        </div>
      </Card>

      {result && (
        <Card style={{ borderColor: result.ok ? colors.success : colors.error, background: result.ok ? `${colors.success}04` : `${colors.error}04` }}>
          <div className="card-header" style={{ color: result.ok ? colors.success : colors.error }}>
            {result.ok ? '✅ Passed' : '❌ Failed'}
          </div>
          <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: 6, fontSize: 12 }}>
            <div style={{ display: 'flex', gap: 10, color: 'var(--text-secondary)' }}>
              <span>{service}/{method}</span>
            </div>
            <div style={{ display: 'flex', gap: 20 }}>
              <div>
                <div style={{ fontSize: 11, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Expected</div>
                <div style={{ fontSize: 22, fontWeight: 700, color: result.ok ? colors.success : 'var(--text)' }}>{result.expected}</div>
              </div>
              {result.actual !== undefined && (
                <div>
                  <div style={{ fontSize: 11, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Actual</div>
                  <div style={{ fontSize: 22, fontWeight: 700, color: result.ok ? colors.success : colors.error }}>{result.actual}</div>
                </div>
              )}
              <div style={{ display: 'flex', alignItems: 'flex-end', paddingBottom: 2 }}>
                <div style={{
                  fontSize: 11, padding: '2px 8px', borderRadius: 4,
                  background: result.ok ? `${colors.success}18` : `${colors.error}18`,
                  color: result.ok ? colors.success : colors.error, fontWeight: 600,
                }}>
                  {result.ok ? 'Match' : result.actual !== undefined && result.expected !== undefined && result.actual > result.expected ? 'Exceeded' : 'Mismatch'}
                </div>
              </div>
            </div>
            {result.msg !== 'Verification passed' && (
              <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>{result.msg}</div>
            )}
            {!result.ok && result.actual !== undefined && (
              <div style={{ display: 'flex', gap: 6, marginTop: 4 }}>
                <Button onClick={() => { setCount(result.actual!); setResult(null); }}>
                  <RotateCcw size={11} /> Use actual ({result.actual})
                </Button>
                <Button variant="ghost" onClick={() => { setResult(null); }}>Dismiss</Button>
              </div>
            )}

            {/* Evidence: the actual calls behind the counter */}
            <div style={{ borderTop: '1px solid var(--border)', marginTop: 6, paddingTop: 8 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 6 }}>
                <span className="section-title">Actual calls ({actualCalls.length})</span>
                <div style={{ flex: 1 }} />
                {actualCalls.length > 0 && (
                  <button className="btn btn-ghost btn-sm" onClick={() => navigate(`/history?q=${encodeURIComponent(method)}`)}>
                    <ExternalLink size={11} /> History
                  </button>
                )}
              </div>
              {actualCalls.length === 0
                ? <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>No matching calls recorded in this scope.</div>
                : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 3, maxHeight: 160, overflow: 'auto' }}>
                    {actualCalls.slice(0, 20).map((c, i) => {
                      const ok = !c.code || c.code === 0;
                      return (
                        <div key={i} onClick={() => c.stubId && navigate(`/stubs/${c.stubId}`)}
                          style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 11.5, padding: '3px 6px', borderRadius: 4, cursor: c.stubId ? 'pointer' : 'default', borderLeft: `2px solid ${ok ? colors.success : colors.error}`, background: 'var(--bg)' }}>
                          <span style={{ color: 'var(--text-muted)', fontFamily: 'var(--mono)' }}>{new Date(c.timestamp).toLocaleTimeString()}</span>
                          <span style={{ color: ok ? colors.success : colors.error, fontWeight: 600 }}>{ok ? 'OK' : `err ${c.code}`}</span>
                          {c.stubId ? <code style={{ color: 'var(--accent-text)' }}>{c.stubId.slice(0, 8)}</code> : <span style={{ color: 'var(--text-muted)' }}>no match</span>}
                        </div>
                      );
                    })}
                  </div>
                )}
            </div>
          </div>
        </Card>
      )}
    </div>
  );
}
