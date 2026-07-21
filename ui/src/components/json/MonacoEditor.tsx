import { useRef, useCallback, useEffect, useState } from 'react';
import Editor, { type OnMount } from '@monaco-editor/react';
import { useStore } from '../../lib/store';

let configured = false;
async function configureWorkers() {
  if (configured) return;
  configured = true;
  try {
    const ew = (await import('monaco-editor/esm/vs/editor/editor.worker.js?worker')).default;
    const jw = (await import('monaco-editor/esm/vs/language/json/json.worker.js?worker')).default;
    (self as any).MonacoEnvironment = {
      getWorker(_: string, label: string) {
        return label === 'json' ? new jw() : new ew();
      },
    };
  } catch (e) { console.error('Monaco workers init failed:', e); }
}

interface MonacoEditorProps {
  value?: string;
  onChange?: (v: string) => void;
  readOnly?: boolean;
  height?: string | number;
  label?: string;
}

export function MonacoEditor({ value = '', onChange, readOnly = false, height = 200, label }: Readonly<MonacoEditorProps>) {
  const [ready, setReady] = useState(false);
  const editorRef = useRef<Parameters<OnMount>[0] | null>(null);
  const theme = useStore((s) => s.theme);
  const monacoTheme = theme === 'dark' ? 'vs-dark' : 'vs';

  useEffect(() => { configureWorkers().then(() => setReady(true)); }, []);

  const handleMount: OnMount = useCallback((editor) => {
    editorRef.current = editor;
    editor.updateOptions({ theme: monacoTheme });
  }, [monacoTheme]);

  useEffect(() => {
    editorRef.current?.updateOptions({ theme: monacoTheme });
  }, [monacoTheme]);

  if (!ready) {
    return (
      <div style={{ height, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-muted)', fontSize: 12, borderRadius: 6, background: 'var(--bg-secondary)' }}>
        {label ? `${label}: ` : ''}Loading editor...
      </div>
    );
  }

  return (
    <Editor
      value={value}
      onChange={(v) => onChange?.(v ?? '')}
      language="json"
      height={height}
      theme={monacoTheme}
      options={{
        readOnly, minimap: { enabled: false }, fontSize: 13, fontFamily: 'ui-monospace, monospace',
        wordWrap: 'on', tabSize: 2, bracketPairColorization: { enabled: true },
        lineNumbers: 'on', scrollBeyondLastLine: false, automaticLayout: true, padding: { top: 6, bottom: 6 },
      }}
      onMount={handleMount}
      loading={<div style={{ padding: 12, color: 'var(--text-muted)', fontSize: 12 }}>Loading...</div>}
    />
  );
}
