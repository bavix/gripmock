import { useState } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { MethodSelect } from '../components/shared/MethodSelect';
import { MonacoEditor } from '../components/json/MonacoEditor';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Send, CheckCircle2, XCircle, ArrowLeft, RotateCcw, Hash, Sparkles, Bug, Plus } from 'lucide-react';
import { colors } from '../lib/theme';
import { api } from '../lib/api';
import { useServiceMethod } from '../hooks/useServices';
import { generateSample } from '../features/stubs/generateSample';
import { stashClone } from '../lib/clone';

export function StubTestPage() {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [service, setService] = useState(params.get('service') || '');
  const [method, setMethod] = useState(params.get('method') || '');
  const [stubId, setStubId] = useState(params.get('id') || '');
  const [payload, setPayload] = useState(params.get('payload') || '{\n  "name": "test"\n}');
  const [headers, setHeaders] = useState(params.get('headers') || '{}');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{
    matched: boolean; stubId?: string; error?: string; code?: number;
    headers?: Record<string, string>; data?: unknown;
  } | null>(null);
  const { data: methodSchema } = useServiceMethod(service || null, method || null);

  // Seed a create-stub form from the tested request (as an equals matcher).
  const createFromRequest = () => {
    let input: unknown = {};
    try { input = JSON.parse(payload); } catch {}
    const stub = { service, method, input: { equals: input }, output: { data: {} } };
    stashClone(stub);
    navigate('/stubs/create?clone=1');
  };

  const handleTest = async () => {
    setResult(null);
    setLoading(true);
    let data: Record<string, unknown> | undefined;
    let hdrs: Record<string, string> | undefined;
    try { data = JSON.parse(payload); } catch {}
    try { hdrs = JSON.parse(headers); } catch {}

    const body: Record<string, unknown> = { service, method, data };
    if (stubId) body.id = stubId;
    if (hdrs && Object.keys(hdrs).length > 0) body.headers = hdrs;

    try {
      const res = await api.post<{ id?: string; error?: string; code?: number; headers?: Record<string, string>; data?: unknown }>('/stubs/search', body);
      setResult({ matched: true, stubId: res.id, error: res.error, code: res.code, headers: res.headers, data: res.data ?? res });
    } catch (err) {
      // Keep the full server message — it includes the "Closest match" diagnostic.
      setResult({ matched: false, error: (err as Error).message });
    }
    setLoading(false);
  };

  const handleReset = () => {
    setStubId('');
    setPayload('{\n  "name": "test"\n}');
    setHeaders('{}');
    setResult(null);
  };

  return (
    <div className="page-enter" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <Button variant="ghost" onClick={() => navigate(-1)}><ArrowLeft size={14} /> Back</Button>
        <div style={{ flex: 1 }} />
      </div>

      <h1 style={{ fontSize: 16, fontWeight: 600, margin: 0 }}>Test Stub Matching</h1>
      <p style={{ fontSize: 11, color: 'var(--text-muted)', margin: 0 }}>
        Sends <code>POST /stubs/search</code> to find which stub matches your request.
      </p>

      {service && method && (
        <div className="chip" style={{ background: `${colors.accent}12`, color: colors.accent, fontSize: 11, alignSelf: 'flex-start' }}>
          Testing: {service}/{method}
        </div>
      )}

      <Card>
        <div className="card-header">Request</div>
        <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <MethodSelect service={service} method={method}
            onServiceChange={(s) => { setService(s); setMethod(''); setResult(null); }}
            onMethodChange={(m) => { setMethod(m); setResult(null); }} />

          <div>
            <div className="section-title" style={{ marginBottom: 4 }}>Stub ID (optional)</div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <Hash size={12} style={{ color: 'var(--text-muted)' }} />
              <input value={stubId} onChange={(e) => setStubId(e.target.value)}
                placeholder="Filter by stub UUID"
                className="input" style={{ width: 280, fontFamily: 'monospace', fontSize: 11 }} />
            </div>
          </div>

          <div>
            <div className="section-title" style={{ marginBottom: 4 }}>Request Payload</div>
            <MonacoEditor value={payload} onChange={setPayload} height={140} />
          </div>

          <div>
            <div className="section-title" style={{ marginBottom: 4 }}>Headers (optional)</div>
            <MonacoEditor value={headers} onChange={setHeaders} height={80} />
          </div>

          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            <Button variant="primary" onClick={handleTest} disabled={!service || !method || loading}>
              <Send size={12} /> {loading ? 'Searching...' : 'Search'}
            </Button>
            {methodSchema?.requestSchema && (
              <Button onClick={() => setPayload(JSON.stringify(generateSample(methodSchema.requestSchema), null, 2))}>
                <Sparkles size={12} /> Generate payload
              </Button>
            )}
            {service && method && (
              <Button onClick={() => navigate(`/inspect?service=${encodeURIComponent(service)}&method=${encodeURIComponent(method)}&payload=${encodeURIComponent(payload)}`)}>
                <Bug size={12} /> Inspect
              </Button>
            )}
            <Button onClick={handleReset} disabled={loading}><RotateCcw size={12} /> Reset</Button>
          </div>
        </div>
      </Card>

      {loading && (
        <Card><div className="card-body" style={{ textAlign: 'center', color: 'var(--text-muted)', fontSize: 12 }}>Searching...</div></Card>
      )}

      {result && !loading && (() => {
        const returnsError = result.matched && result.code !== undefined && result.code > 0;
        const headColor = returnsError ? colors.warning : result.matched ? colors.success : colors.warning;
        return (
        <Card style={{ borderColor: headColor }}>
          <div className="card-header">Result</div>
          <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, fontWeight: 600, color: headColor }}>
              {result.matched ? <CheckCircle2 size={16} /> : <XCircle size={16} />}
              {returnsError ? 'Matched — stub returns an error' : result.matched ? 'Match found' : 'No match'}
              {result.stubId && <code style={{ fontSize: 11, background: `${headColor}18`, padding: '1px 8px', borderRadius: 4 }}>{result.stubId.slice(0, 12)}</code>}
            </div>

            {result.code !== undefined && result.code > 0 && (
              <div style={{ fontSize: 11, padding: '4px 8px', borderRadius: 4, background: `${colors.error}10`, color: colors.error, display: 'inline-flex', alignItems: 'center', gap: 4, alignSelf: 'flex-start' }}>
                Code: {result.code} · {result.error || 'No error message'}
              </div>
            )}

            {!result.matched && result.error && (
              <div>
                <div className="section-title" style={{ marginBottom: 2 }}>Why it didn't match</div>
                <pre style={jsonBlock}>{result.error}</pre>
                <div style={{ display: 'flex', gap: 6, marginTop: 6 }}>
                  <Button onClick={createFromRequest}><Plus size={12} /> Create stub from this request</Button>
                  <Button onClick={() => navigate(`/inspect?service=${encodeURIComponent(service)}&method=${encodeURIComponent(method)}&payload=${encodeURIComponent(payload)}`)}><Bug size={12} /> Inspect ranking</Button>
                </div>
              </div>
            )}

            {result.headers && Object.keys(result.headers).length > 0 && (
              <div>
                <div className="section-title" style={{ marginBottom: 2 }}>Response Headers</div>
                <pre style={jsonBlock}>{JSON.stringify(result.headers, null, 2)}</pre>
              </div>
            )}

            {!!result.data && (
              <div>
                <div className="section-title" style={{ marginBottom: 2 }}>Response Data</div>
                <pre style={jsonBlock}>{JSON.stringify(result.data, null, 2)}</pre>
              </div>
            )}

            {!result.data && result.matched && (
              <pre style={jsonBlock}>{JSON.stringify({ id: result.stubId, headers: result.headers }, null, 2)}</pre>
            )}

            {result.matched && result.stubId && (
              <Button onClick={() => navigate(`/stubs/${result.stubId}`)}>
                View stub →
              </Button>
            )}
          </div>
        </Card>
        );
      })()}
    </div>
  );
}

const jsonBlock = {
  fontSize: 11, fontFamily: 'ui-monospace, monospace' as const, padding: 10, borderRadius: 6,
  border: '1px solid var(--border)', overflow: 'auto' as const, maxHeight: 300, margin: 0,
  background: 'var(--bg)', lineHeight: 1.5, whiteSpace: 'pre-wrap' as const, wordBreak: 'break-word' as const,
};
