import { useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useDescriptors, useUploadDescriptor } from '../hooks/useDescriptors';
import { api } from '../lib/api';
import { Upload, FileUp, CheckCircle2, Trash2, ArrowRight, AlertCircle } from 'lucide-react';
import { colors } from '../lib/theme';
import { useToast } from '../components/shared/Toast';

export function DescriptorsList() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const toast = useToast();
  const { data, isLoading } = useDescriptors();
  const upload = useUploadDescriptor();
  const fileRef = useRef<HTMLInputElement>(null);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [justAdded, setJustAdded] = useState<string[] | null>(null);
  const [fileName, setFileName] = useState<string>('');

  const ids = data?.serviceIDs ?? [];

  const unregister = useMutation({
    mutationFn: (id: string) => api.delete(`/services/${encodeURIComponent(id)}`),
    onSuccess: (_r, id) => {
      qc.invalidateQueries({ queryKey: ['descriptors'] });
      qc.invalidateQueries({ queryKey: ['services'] });
      toast.show(`Unregistered ${id}`);
    },
    onError: (e) => toast.show((e as Error).message),
  });

  const handleUpload = async () => {
    const file = fileRef.current?.files?.[0];
    if (!file) return;
    setUploadError(null);
    setJustAdded(null);
    try {
      const res = await upload.mutateAsync(file);
      setJustAdded(res?.serviceIDs ?? []);
      if (fileRef.current) fileRef.current.value = '';
      setFileName('');
    } catch (err) {
      setUploadError((err as Error).message);
    }
  };

  return (
    <div className="page-enter" style={{ display: 'flex', flexDirection: 'column', gap: 12, maxWidth: 720 }}>
      <h1>Descriptors</h1>
      <p style={{ fontSize: 12.5, color: 'var(--text-muted)', margin: 0, marginTop: -4, lineHeight: 1.5 }}>
        Register new gRPC services into the running mock by uploading a compiled <code>FileDescriptorSet</code> (<code>.pb</code>/<code>.bin</code>) — no restart needed.
        Startup-loaded services live under <button className="btn btn-ghost btn-sm" style={{ padding: '0 4px', height: 'auto', display: 'inline' }} onClick={() => navigate('/services')}>Services</button>.
      </p>

      <div className="card">
        <div className="card-header">Upload descriptor</div>
        <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <input ref={fileRef} type="file" accept=".bin,.pb" onChange={(e) => { setFileName(e.target.files?.[0]?.name ?? ''); setJustAdded(null); setUploadError(null); }}
              className="input" style={{ flex: 1, fontSize: 12.5 }} />
            <button onClick={handleUpload} disabled={upload.isPending || !fileName} className="btn btn-primary">
              <Upload size={13} /> {upload.isPending ? 'Uploading…' : 'Upload'}
            </button>
          </div>
          {justAdded && (
            <div style={{ fontSize: 12.5, color: colors.success, display: 'flex', alignItems: 'flex-start', gap: 6, padding: '7px 10px', borderRadius: 'var(--radius)', background: 'var(--success-bg)', border: `1px solid ${colors.success}30` }}>
              <CheckCircle2 size={14} style={{ flexShrink: 0, marginTop: 1 }} />
              <span>
                {justAdded.length > 0
                  ? <>Registered {justAdded.length} service{justAdded.length > 1 ? 's' : ''}: {justAdded.map((s, i) => <code key={s} style={{ color: 'var(--text)' }}>{i > 0 ? ', ' : ''}{s}</code>)}</>
                  : 'Descriptor uploaded.'}
              </span>
            </div>
          )}
          {uploadError && (
            <div style={{ fontSize: 12.5, color: colors.error, display: 'flex', alignItems: 'center', gap: 6, padding: '7px 10px', borderRadius: 'var(--radius)', background: 'var(--error-bg)' }}>
              <AlertCircle size={14} /> {uploadError}
            </div>
          )}
        </div>
      </div>

      <div className="card">
        <div className="card-header">Runtime-added descriptors ({ids.length})</div>
        <div>
          {isLoading && <div style={{ padding: 14, color: 'var(--text-muted)', fontSize: 13 }}>Loading…</div>}
          {!isLoading && ids.length === 0 && (
            <div className="empty" style={{ padding: 28 }}>
              <FileUp size={26} />
              <span>No descriptors added at runtime. Upload a <code>.pb</code> above to register services on the fly.</span>
            </div>
          )}
          {ids.map((sid) => (
            <div key={sid} className="hover-row" style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '9px 14px', fontSize: 12.5, borderBottom: '1px solid var(--border)' }}>
              <FileUp size={14} style={{ color: colors.accent, flexShrink: 0 }} />
              <code style={{ flex: 1 }}>{sid}</code>
              <button className="btn btn-ghost btn-sm" onClick={() => navigate(`/services?q=${encodeURIComponent(sid)}`)} title="View in Services"><ArrowRight size={12} /></button>
              <button className="btn btn-ghost btn-sm" onClick={() => unregister.mutate(sid)} disabled={unregister.isPending}
                style={{ color: colors.error }} title="Unregister this service"><Trash2 size={12} /></button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
