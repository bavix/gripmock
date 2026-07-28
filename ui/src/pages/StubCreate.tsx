import { useSearchParams } from 'react-router-dom';
import { StubForm } from '../features/stubs/StubForm';
import { takeClone } from '../lib/clone';

export function StubCreate() {
  const [params] = useSearchParams();
  let initial: Record<string, unknown> | undefined;
  // Clone payload comes via sessionStorage (see stashClone) to avoid URL limits.
  if (params.get('clone')) initial = takeClone() ?? undefined;
  // Legacy: stub JSON in the ?source= param.
  const source = params.get('source');
  if (!initial && source) try { initial = JSON.parse(decodeURIComponent(source)); } catch {}
  // Prefill service/method when coming from a method's "New stub" action.
  if (!initial) {
    const service = params.get('service'); const method = params.get('method');
    if (service || method) initial = { service: service || '', method: method || '' };
  }

  return (
    <div className="page-enter">
      <StubForm initial={initial} />
    </div>
  );
}
