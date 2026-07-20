import { useState, useCallback, createContext, useContext, type ReactNode } from 'react';

interface ToastItem {
  id: number;
  message: string;
  action?: { label: string; onClick: () => void };
}

interface ToastCtx {
  show: (msg: string, action?: { label: string; onClick: () => void }) => void;
}

const Ctx = createContext<ToastCtx>({ show: () => {} });

export function useToast() { return useContext(Ctx); }

let nextId = 0;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const show = useCallback((message: string, action?: { label: string; onClick: () => void }) => {
    const id = nextId++;
    setToasts((prev) => [...prev, { id, message, action }]);
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 5000);
  }, []);

  return (
    <Ctx.Provider value={{ show }}>
      {children}
      <div role="status" aria-live="polite" aria-atomic="true" style={{ position: 'fixed', bottom: 48, left: '50%', transform: 'translateX(-50%)', zIndex: 300, display: 'flex', flexDirection: 'column', gap: 6, pointerEvents: 'none' }}>
        {toasts.map((t) => (
          <div key={t.id} style={{
            pointerEvents: 'auto', padding: '8px 14px', borderRadius: 6,
            background: 'var(--bg-primary)', border: '1px solid var(--border)',
            boxShadow: '0 4px 16px rgba(0,0,0,0.2)',
            display: 'flex', alignItems: 'center', gap: 10,
            fontSize: 12, color: 'var(--text-primary)',
            animation: 'slideUp 0.2s ease',
          }}>
            <span>{t.message}</span>
            {t.action && (
              <button onClick={() => { t.action?.onClick?.(); setToasts((prev) => prev.filter((x) => x.id !== t.id)); }}
                style={{ padding: '3px 8px', fontSize: 11, borderRadius: 4, border: '1px solid var(--accent)', background: 'transparent', color: colors.accent, cursor: 'pointer', whiteSpace: 'nowrap' }}>
                {t.action.label}
              </button>
            )}
          </div>
        ))}
      </div>
      <style>{`
        @keyframes slideUp { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
      `}</style>
    </Ctx.Provider>
  );
}

const colors = { accent: 'var(--accent)' } as const;
