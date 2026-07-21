import { useEffect } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useStore } from './lib/store';
import { AppShell } from './components/layout/AppShell';
import { CommandPalette } from './components/command/CommandPalette';
import { ErrorBoundary } from './components/shared/ErrorBoundary';
import { Dashboard } from './pages/Dashboard';
import { StubsList } from './pages/StubsList';
import { StubShow } from './pages/StubShow';
import { StubCreate } from './pages/StubCreate';
import { StubEdit } from './pages/StubEdit';
import { ServicesList } from './pages/ServicesList';
import { HistoryList } from './pages/HistoryList';
import { DescriptorsList } from './pages/DescriptorsList';
import { SessionPage } from './pages/SessionPage';
import { VerifyPage } from './pages/VerifyPage';
import { InspectPage } from './pages/InspectPage';
import { StubTestPage } from './pages/StubTestPage';

export function App() {
  const theme = useStore((s) => s.theme);
  const setTheme = useStore((s) => s.setTheme);
  useEffect(() => { document.documentElement.dataset.theme = theme; }, [theme]);
  // Optional ?theme=light|dark override for shareable/embedded links.
  useEffect(() => {
    const t = new URLSearchParams(window.location.search).get('theme');
    if (t === 'light' || t === 'dark') setTheme(t);
  }, [setTheme]);

  return (<>
    <CommandPalette />
    <AppShell>
      <ErrorBoundary>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/stubs" element={<StubsList />} />
        <Route path="/stubs/used" element={<StubsList filter="used" />} />
        <Route path="/stubs/unused" element={<StubsList filter="unused" />} />
        <Route path="/stubs/create" element={<StubCreate />} />
        <Route path="/stubs/test" element={<StubTestPage />} />
        <Route path="/stubs/:id" element={<StubShow />} />
        <Route path="/stubs/:id/edit" element={<StubEdit />} />
        <Route path="/services" element={<ServicesList />} />
        <Route path="/history" element={<HistoryList />} />
        <Route path="/descriptors" element={<DescriptorsList />} />
        <Route path="/session" element={<SessionPage />} />
        <Route path="/verify" element={<VerifyPage />} />
        <Route path="/inspect" element={<InspectPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
      </ErrorBoundary>
    </AppShell>
  </>);
}
