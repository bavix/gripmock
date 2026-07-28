import { useState, useCallback } from 'react';

// Copy-to-clipboard with a transient "copied" flag for a checkmark affordance.
export function useCopy(resetMs = 1200): { copied: boolean; copy: (text: string) => void } {
  const [copied, setCopied] = useState(false);
  const copy = useCallback((text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), resetMs);
  }, [resetMs]);
  return { copied, copy };
}
