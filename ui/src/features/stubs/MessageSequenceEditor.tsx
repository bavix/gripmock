import { MonacoEditor } from '../../components/json/MonacoEditor';
import { Plus, X, ArrowUp, ArrowDown } from 'lucide-react';

export interface SequenceItem {
  type: string; // equals | contains | matches | glob
  value: string; // JSON text
  ignoreArrayOrder: boolean;
  _k?: string; // stable React key (UI-only; ignored when the stub is built)
}

const KINDS = ['equals', 'contains', 'matches', 'glob'];

let seqKeySeed = 0;
const newSequenceItem = (): SequenceItem => ({ type: 'equals', value: '{\n  \n}', ignoreArrayOrder: false, _k: `seq-${seqKeySeed++}` });

/**
 * Editor for a stub's `inputs[]`.
 *
 * The same storage field has two semantics depending on the method type:
 * - client/bidi streaming → ORDERED request messages, matched in sequence;
 * - unary/server-streaming → alternative matchers, any may match (OR).
 * The `streaming` flag only changes labels/ordering affordances — never the data.
 */
export function MessageSequenceEditor({ items, onChange, streaming }: Readonly<{
  items: SequenceItem[];
  onChange: (items: SequenceItem[]) => void;
  streaming: boolean;
}>) {
  const set = (i: number, patch: Partial<SequenceItem>) => {
    const n = [...items]; n[i] = { ...n[i], ...patch }; onChange(n);
  };
  const move = (i: number, dir: -1 | 1) => {
    const j = i + dir;
    if (j < 0 || j >= items.length) return;
    const n = [...items]; [n[i], n[j]] = [n[j], n[i]]; onChange(n);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>
        {streaming
          ? 'Ordered request messages — the client streams these and the stub matches them in order.'
          : 'Alternative matchers — the stub matches if ANY of these matches the request.'}
      </div>
      {items.length === 0 && <div style={{ fontSize: 11, color: 'var(--text-muted)', fontStyle: 'italic' }}>None.</div>}
      {items.map((item, i) => (
        <div key={item._k ?? `${item.type}:${item.value}`} style={{ border: '1px solid var(--border)', borderRadius: 'var(--radius)', background: 'var(--bg)', overflow: 'hidden' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '4px 8px', borderBottom: '1px solid var(--border)' }}>
            {streaming && (
              <span style={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'center', minWidth: 18, height: 18, borderRadius: 5, background: 'var(--bg-tertiary)', color: 'var(--text-secondary)', fontWeight: 700, fontSize: 10.5 }}>{i + 1}</span>
            )}
            <select value={item.type} onChange={(e) => set(i, { type: e.target.value })} className="input" style={{ width: 100, fontSize: 11, padding: '2px 6px' }} aria-label="Matcher kind">
              {KINDS.map((k) => <option key={k} value={k}>{k}</option>)}
            </select>
            <label style={{ fontSize: 10.5, display: 'flex', alignItems: 'center', gap: 3, color: 'var(--text-muted)', cursor: 'pointer' }}>
              <input type="checkbox" checked={item.ignoreArrayOrder} onChange={() => set(i, { ignoreArrayOrder: !item.ignoreArrayOrder })} /> ignore array order
            </label>
            <div style={{ flex: 1 }} />
            {streaming && (
              <>
                <button type="button" onClick={() => move(i, -1)} disabled={i === 0} className="icon-btn" style={{ width: 22, height: 22 }} title="Move up" aria-label="Move message up"><ArrowUp size={12} /></button>
                <button type="button" onClick={() => move(i, 1)} disabled={i === items.length - 1} className="icon-btn" style={{ width: 22, height: 22 }} title="Move down" aria-label="Move message down"><ArrowDown size={12} /></button>
              </>
            )}
            <button type="button" onClick={() => onChange(items.filter((_, j) => j !== i))} className="icon-btn" style={{ width: 22, height: 22 }} title="Remove" aria-label="Remove message"><X size={12} /></button>
          </div>
          <MonacoEditor value={item.value} onChange={(v) => set(i, { value: v })} height={80} />
        </div>
      ))}
      <button type="button" onClick={() => onChange([...items, newSequenceItem()])} className="btn btn-ghost btn-sm" style={{ alignSelf: 'flex-start' }}>
        <Plus size={11} /> {streaming ? 'Add message' : 'Add alternative'}
      </button>
    </div>
  );
}
