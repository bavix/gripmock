import { LayoutDashboard, ListOrdered, History, FileSearch, FileUp, Users, Layers, CheckCheck, ShieldCheck, SearchCode } from 'lucide-react';

export interface AppRoute {
  to: string;
  icon: typeof LayoutDashboard;
  label: string;
}

// One list for the sidebar and the command palette: keeping two of them let the
// palette silently miss Descriptors, Session and the used/unused stub views.
export const ROUTES: AppRoute[] = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/services', icon: Layers, label: 'Services' },
  { to: '/stubs', icon: ListOrdered, label: 'Stubs' },
  { to: '/stubs/used', icon: CheckCheck, label: 'Used' },
  { to: '/stubs/unused', icon: FileSearch, label: 'Unused' },
  { to: '/history', icon: History, label: 'History' },
  { to: '/inspect', icon: SearchCode, label: 'Inspect' },
  { to: '/verify', icon: ShieldCheck, label: 'Verify' },
  { to: '/descriptors', icon: FileUp, label: 'Descriptors' },
  { to: '/session', icon: Users, label: 'Session' },
];
