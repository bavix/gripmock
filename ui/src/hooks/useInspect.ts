import { useMutation } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { InspectRequest, InspectReport } from '../lib/types';

export function useInspect() {
  return useMutation({
    mutationFn: (req: InspectRequest) => api.post<InspectReport>('/stubs/inspect', req),
  });
}
