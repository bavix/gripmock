/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: true,
  },
  base: '/',
  test: {
    // Node by default (fast, pure-function tests). Component tests opt into
    // jsdom per-file via `// @vitest-environment jsdom`.
    environment: 'node',
    include: ['src/**/*.test.{ts,tsx}'],
    setupFiles: ['./src/test/setup.ts'],
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('/node_modules/')) return;

          if (id.includes('/node_modules/monaco-editor/')) return 'monaco-editor';

          if (
            id.includes('/node_modules/react-dom/') ||
            id.includes('/node_modules/react/') ||
            id.includes('/node_modules/scheduler/')
          ) return 'vendor-react';

          if (
            id.includes('/node_modules/react-router/') ||
            id.includes('/node_modules/react-router-dom/')
          ) return 'vendor-router';

          if (id.includes('/node_modules/@tanstack/')) return 'vendor-query';

          if (id.includes('/node_modules/lucide-react/')) return 'vendor-lucide';

          if (
            id.includes('/node_modules/recharts/') ||
            id.includes('/node_modules/d3-')
          ) return 'vendor-charts';

          if (id.includes('/node_modules/js-yaml/')) return 'vendor-yaml';

          if (id.includes('/node_modules/zustand/')) return 'vendor-state';

          return 'vendor-misc';
        },
        chunkFileNames: 'assets/chunk-[name]-[hash].js',
        entryFileNames: 'assets/app-[name]-[hash].js',
      },
    },
  },
});
