import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

export function useDescriptors() {
  return useQuery({
    queryKey: ['descriptors'],
    queryFn: () => api.get<{ serviceIDs: string[] }>('/descriptors'),
  });
}

export function useUploadDescriptor() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (file: Blob) => api.post<{ serviceIDs: string[] }>('/descriptors', file),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['descriptors'] }),
  });
}
