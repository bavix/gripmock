import { useRef, useCallback, useEffect, useState } from 'react';
import Editor, { loader, type OnMount } from '@monaco-editor/react';
import { useStore } from '../../lib/store';

let configured = false;
async function configureWorkers() {
  if (configured) return;
  configured = true;
  try {
    const ew = (await import('monaco-editor/editor/editor.worker.js?worker')).default;
    const jw = (await import('monaco-editor/languages/features/json/json.worker.js?worker')).default;
    (self as any).MonacoEnvironment = {
      getWorker(_: string, label: string) {
        return label === 'json' ? new jw() : new ew();
      },
    };
    // Bundle monaco locally; without this the loader fetches it from cdn.jsdelivr.net
    // at runtime and the editor breaks in offline/air-gapped environments.
    // editor.api + register.all + json register only (monaco 0.56 layout;
    // the esm/vs/* deep paths and edcore.main are gone) — the editor.main entry
    // would also pull every language definition and the css/html/ts workers
    // (~2MB extra in the binary).
    const monaco = await import('monaco-editor/editor/editor.api.js');
    await import('monaco-editor/features/register.all.js');
    await import('monaco-editor/languages/features/json/register.js');
    loader.config({ monaco });
  } catch (e) { console.error('Monaco workers init failed:', e); }
}

interface MonacoEditorProps {
  value?: string;
  onChange?: (v: string) => void;
  readOnly?: boolean;
  height?: string | number;
  label?: string;
  language?: string;
}

export function MonacoEditor({ value = '', onChange, readOnly = false, height = 200, label, language = 'json' }: Readonly<MonacoEditorProps>) {
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
      language={language}
      height={height}
      theme={monacoTheme}
      options={{
        readOnly, minimap: { enabled: false }, fontSize: 13, fontFamily: 'ui-monospace, monospace',
        wordWrap: 'on', tabSize: 2, bracketPairColorization: { enabled: true },
        // A Go template is typed as `{{ … }}`; auto-closing would leave the braces doubled.
        autoClosingBrackets: language === 'plaintext' ? 'never' : 'languageDefined',
        autoClosingQuotes: language === 'plaintext' ? 'never' : 'languageDefined',
        lineNumbers: 'on', scrollBeyondLastLine: false, automaticLayout: true, padding: { top: 6, bottom: 6 },
      }}
      onMount={handleMount}
      loading={<div style={{ padding: 12, color: 'var(--text-muted)', fontSize: 12 }}>Loading...</div>}
    />
  );
}
