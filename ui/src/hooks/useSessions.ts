import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

export function useSessions() {
  return useQuery({
    queryKey: ['sessions'],
    queryFn: () => api.get<{ sessions: string[] }>('/sessions'),
    // Sessions appear as live calls arrive (X-Gripmock-Session header), so poll.
    refetchInterval: 20_000,
  });
}

// Wipes one session's fixtures and evidence. Both endpoints scope by the
// X-Gripmock-Session header, so this never touches another session or the
// global stubs — which is why the id is sent explicitly instead of relying on
// whichever session happens to be active.
export function usePurgeSession() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (session: string) => {
      const headers = { 'X-Gripmock-Session': session };
      await api.delete('/stubs', headers);
      await api.delete('/history', headers);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['stubs'] });
      qc.invalidateQueries({ queryKey: ['history'] });
      qc.invalidateQueries({ queryKey: ['sessions'] });
      qc.invalidateQueries({ queryKey: ['dashboard'] });
    },
  });
}
