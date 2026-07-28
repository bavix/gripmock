import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

export function useSessions() {
  return useQuery({
    queryKey: ['sessions'],
    queryFn: () => api.get<{ sessions: string[] }>('/sessions'),
    // Sessions appear as live calls arrive (X-Gripmock-Session header), so poll.
    refetchInterval: 20_000,
  });
}
