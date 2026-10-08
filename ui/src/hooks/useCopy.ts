import { useState, useCallback, useEffect, useRef } from 'react';
import { copyText } from '../lib/clipboard';

// Copy-to-clipboard with a transient "copied" flag for a checkmark affordance.
export function useCopy(resetMs = 1200): {
  copied: boolean;
  failed: boolean;
  copy: (text: string) => Promise<boolean>;
} {
  const [state, setState] = useState<'idle' | 'copied' | 'failed'>('idle');
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  useEffect(() => () => clearTimeout(timer.current), []);

  const copy = useCallback(async (text: string) => {
    const ok = await copyText(text);
    setState(ok ? 'copied' : 'failed');
    clearTimeout(timer.current);
    timer.current = setTimeout(() => setState('idle'), resetMs);
    return ok;
  }, [resetMs]);

  return { copied: state === 'copied', failed: state === 'failed', copy };
}
