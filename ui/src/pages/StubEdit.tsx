import { useParams, useNavigate } from 'react-router-dom';
import { useStub } from '../hooks/useStubs';
import { StubForm } from '../features/stubs/StubForm';

export function StubEdit() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: stub, isLoading } = useStub(id!);

  if (isLoading) return <div className="page-enter" style={{ padding: 24, color: 'var(--text-muted)' }}>Loading stub...</div>;
  if (!stub) return <div className="page-enter" style={{ padding: 24, color: 'var(--error)' }}>Stub not found.</div>;

  return (
    <div className="page-enter">
      <StubForm
        initial={{ ...stub, id: stub.id } as any}
        onSaved={() => navigate(`/stubs/${id}`)}
      />
    </div>
  );
}
