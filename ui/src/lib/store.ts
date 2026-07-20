import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface AppState {
  theme: 'light' | 'dark';
  setTheme: (t: 'light' | 'dark') => void;
  session: string | null;
  setSession: (s: string | null) => void;
  recentSessions: string[];
  trackSession: (s: string) => void;
  pinnedStubs: { id: string; service: string; method: string }[];
  togglePin: (s: { id: string; service: string; method: string }) => void;
}

export const useStore = create<AppState>()(
  persist(
    (set) => ({
      theme: 'dark',
      setTheme: (theme) => {
        document.documentElement.setAttribute('data-theme', theme);
        set({ theme });
      },
      session: null,
      setSession: (session) => {
        if (session) window.dispatchEvent(new CustomEvent('gripmock:session-changed', { detail: session }));
        set({ session });
      },
      recentSessions: [],
      trackSession: (s) => set((state) => ({ recentSessions: [s, ...state.recentSessions.filter((x) => x !== s)].slice(0, 8) })),
      pinnedStubs: [],
      togglePin: (stub) => set((state) => {
        const exists = state.pinnedStubs.find((p) => p.id === stub.id);
        return {
          pinnedStubs: exists
            ? state.pinnedStubs.filter((p) => p.id !== stub.id)
            : [...state.pinnedStubs, stub].slice(0, 10),
        };
      }),
    }),
    {
      name: 'gripmock-ui-store',
      partialize: (state) => ({ theme: state.theme, session: state.session, recentSessions: state.recentSessions, pinnedStubs: state.pinnedStubs }),
      onRehydrateStorage: () => (state) => { if (state?.theme) document.documentElement.setAttribute('data-theme', state.theme); },
    },
  ),
);
