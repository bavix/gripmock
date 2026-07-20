import { useMutation } from '@tanstack/react-query';
import { api } from '../lib/api';

export function useVerify() {
  return useMutation({
    mutationFn: (req: { service: string; method: string; expectedCount: number }) => api.post('/verify', req),
  });
}
