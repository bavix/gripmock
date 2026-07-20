import { useState, type ReactNode } from 'react';
import { TopBar } from './TopBar';
import { Sidebar } from './Sidebar';
import { StatusBar } from './StatusBar';

export function AppShell({ children }: { children: ReactNode }) {
  const [collapsed, setCollapsed] = useState(true);

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <a href="#main-content" className="skip-link">Skip to content</a>
      <TopBar onToggleSidebar={() => setCollapsed((v) => !v)} />
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        <aside style={{
          width: collapsed ? 48 : 200,
          flexShrink: 0,
          background: 'var(--bg-secondary)',
          borderRight: '1px solid var(--border)',
          overflow: 'hidden',
          transition: 'width 0.15s ease',
        }}>
          <Sidebar collapsed={collapsed} />
        </aside>
        <main id="main-content" style={{ flex: 1, overflow: 'auto', background: 'var(--bg-primary)' }}>
          <div style={{ padding: 16 }}>{children}</div>
        </main>
      </div>
      <StatusBar />
    </div>
  );
}
