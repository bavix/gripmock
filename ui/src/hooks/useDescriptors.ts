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
    // The endpoint takes a raw FileDescriptorSet; api.post would JSON.stringify the
    // File into "{}" and the server would reject an empty descriptor.
    mutationFn: (file: Blob) => api.postBinary<{ serviceIDs: string[] }>('/descriptors', file),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['descriptors'] }),
  });
}
