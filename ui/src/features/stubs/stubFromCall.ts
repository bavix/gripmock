import { type CallRecord, isCallOk } from '../../lib/types';
import { hasKeys } from './buildStubOutput';

// Build a stub prefilled from a recorded call: the request(s) become the input
// matcher(s), the recorded response(s) and headers become the output. Lives
// next to buildStubBody/buildStubOutput so the stub wire-format stays owned by
// one module (StubCreate consumes this via the stashClone handoff).
export function stubFromCall(r: CallRecord): Record<string, unknown> {
  const requests = r.requests ?? (r.request ? [r.request] : []);
  const responses = r.responses ?? (r.response ? [r.response] : []);

  const output: Record<string, unknown> = {};
  if (responses.length > 1) output.stream = responses;
  else output.data = responses[0] ?? {};
  if (hasKeys(r.responseHeaders)) output.headers = r.responseHeaders;
  if (!isCallOk(r) && r.error) {
    output.error = r.error;
    output.code = r.code;
  }

  const stub: Record<string, unknown> = { service: r.service, method: r.method, output };

  // Multi-message requests (client-stream/bidi) must become the ordered
  // inputs[] sequence — a single input.equals would never match the stream.
  if (requests.length > 1) stub.inputs = requests.map((m) => ({ equals: m }));
  else stub.input = { equals: requests[0] ?? {} };

  return stub;
}
