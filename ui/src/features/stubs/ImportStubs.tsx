import { useState, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { Upload, CheckCircle2, XCircle, FileJson, Loader2 } from 'lucide-react';
import { api, type ApiError } from '../../lib/api';
import { colors } from '../../lib/theme';
import { matcherTypes, compactPreview, outputKind } from '../../lib/stub';
import { useToast } from '../../components/shared/Toast';
import type { Stub } from '../../lib/types';

type Parsed = { stubs: Partial<Stub>[]; fileErrors: string[] };

// Parse .json/.yaml/.yml files into a flat stub array. YAML parser is
// lazy-imported so js-yaml stays out of the entry chunk. Exported for tests.
export async function parseFiles(files: File[]): Promise<Parsed> {
  const stubs: Partial<Stub>[] = [];
  const fileErrors: string[] = [];
  for (const f of files) {
    try {
      const text = await f.text();
      let doc: unknown;
      if (/\.ya?ml$/i.test(f.name)) {
        const yaml = await import('js-yaml');
        doc = yaml.load(text);
      } else {
        doc = JSON.parse(text);
      }
      const arr = Array.isArray(doc) ? doc : [doc];
      for (const s of arr) {
        if (s && typeof s === 'object') stubs.push(s as Partial<Stub>);
        else fileErrors.push(`${f.name}: entry is not an object`);
      }
    } catch (e) {
      fileErrors.push(`${f.name}: ${(e as Error).message}`);
    }
  }
  return { stubs, fileErrors };
}

export function ImportStubs({ onDone }: { onDone: () => void }) {
  const qc = useQueryClient();
  const toast = useToast();
  const [dragOver, setDragOver] = useState(false);
  const [parsed, setParsed] = useState<Partial<Stub>[] | null>(null);
  const [fileErrors, setFileErrors] = useState<string[]>([]);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const handleFiles = useCallback(async (files: File[]) => {
    setValidationError(null);
    const { stubs, fileErrors: errs } = await parseFiles(files);
    setFileErrors(errs);
    if (stubs.length === 0) { setParsed(null); return; }
    // Strip ids so import always creates new stubs, then dry-run validate.
    const clean = stubs.map(({ id: _id, ...rest }) => rest);
    setBusy(true);
    try {
      await api.post('/stubs/validate', clean);
      setParsed(clean);
    } catch (e) {
      setParsed(clean);
      setValidationError((e as ApiError).message);
    }
    setBusy(false);
  }, []);

  const doImport = async () => {
    if (!parsed?.length) return;
    setBusy(true);
    try {
      await api.post('/stubs', parsed);
      qc.invalidateQueries({ queryKey: ['stubs'] });
      toast.show(`Imported ${parsed.length} stub${parsed.length > 1 ? 's' : ''}`);
      onDone();
    } catch (e) {
      setValidationError((e as ApiError).message);
    }
    setBusy(false);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* Drop zone */}
      <label
        onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => { e.preventDefault(); setDragOver(false); handleFiles(Array.from(e.dataTransfer.files)); }}
        style={{
          display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8,
          padding: '28px 16px', borderRadius: 'var(--radius-lg)', cursor: 'pointer',
          border: `2px dashed ${dragOver ? 'var(--accent)' : 'var(--border-strong)'}`,
          background: dragOver ? 'var(--accent-bg)' : 'var(--bg-secondary)',
          color: 'var(--text-muted)', fontSize: 13, textAlign: 'center', transition: 'border-color 0.15s, background 0.15s',
        }}>
        <Upload size={24} />
        <span><strong style={{ color: 'var(--text)' }}>Drop stub files</strong> or click to browse</span>
        <span style={{ fontSize: 11.5 }}>.json / .yaml / .yml — single stub or array; IDs are regenerated</span>
        <input type="file" multiple accept=".json,.yaml,.yml,application/json" style={{ display: 'none' }}
          onChange={(e) => e.target.files && handleFiles(Array.from(e.target.files))} />
      </label>

      {fileErrors.length > 0 && (
        <div style={{ padding: '8px 12px', borderRadius: 'var(--radius)', background: 'var(--error-bg)', color: colors.error, fontSize: 12 }}>
          {fileErrors.map((e, i) => <div key={i}>{e}</div>)}
        </div>
      )}
      {validationError && (
        <div style={{ padding: '8px 12px', borderRadius: 'var(--radius)', background: 'var(--warning-bg)', color: colors.warning, fontSize: 12, display: 'flex', gap: 6 }}>
          <XCircle size={14} style={{ flexShrink: 0, marginTop: 1 }} /> <span>Validation: {validationError}</span>
        </div>
      )}

      {busy && <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--text-muted)', fontSize: 12 }}><Loader2 size={13} className="animate-spin" /> Working…</div>}

      {/* Preview */}
      {parsed && parsed.length > 0 && (
        <div className="card">
          <div className="card-header" style={{ display: 'flex', alignItems: 'center' }}>
            <span style={{ flex: 1 }}>Preview — {parsed.length} stub{parsed.length > 1 ? 's' : ''}</span>
            {!validationError && <span style={{ color: colors.success, display: 'inline-flex', alignItems: 'center', gap: 4, textTransform: 'none', letterSpacing: 0 }}><CheckCircle2 size={12} /> valid</span>}
          </div>
          <div style={{ maxHeight: 300, overflow: 'auto' }}>
            {parsed.map((s, i) => {
              const stub = s as Stub;
              const out = outputKind(stub);
              return (
                <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '7px 12px', borderBottom: '1px solid var(--border)', fontSize: 12 }}>
                  <FileJson size={13} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
                  <span style={{ fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{stub.service || '?'}/{stub.method || '?'}</span>
                  {matcherTypes(stub).map((t) => <span key={t} className="badge" style={{ background: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}>{t}</span>)}
                  <span className="badge" style={{ background: `${out.color}1e`, color: out.color }}>{out.label}</span>
                  <span style={{ flex: 1, fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--text-muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', textAlign: 'right' }}>{compactPreview(stub.input, 36)}</span>
                </div>
              );
            })}
          </div>
          <div style={{ padding: 10, display: 'flex', gap: 8 }}>
            <button onClick={doImport} disabled={busy || !!validationError} className="btn btn-primary"><Upload size={13} /> Import {parsed.length}</button>
            <button onClick={() => { setParsed(null); setFileErrors([]); setValidationError(null); }} className="btn">Clear</button>
          </div>
        </div>
      )}
    </div>
  );
}
