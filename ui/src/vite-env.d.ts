/// <reference types="vite/client" />

interface Window {
  __GRIPMOCK_CONFIG__?: { apiBase?: string; basePath?: string };
}

// The esm subpaths ship no .d.ts; editor.main re-exports the same API as the root entry.
declare module 'monaco-editor/esm/vs/editor/edcore.main.js' {
  export * from 'monaco-editor';
}
declare module 'monaco-editor/esm/vs/language/json/monaco.contribution.js';
