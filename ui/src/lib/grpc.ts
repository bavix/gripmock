import type { CallRecord } from './types';

// Canonical gRPC status codes — single source for the code picker and status labels.
export const GRPC_CODES = [
  { value: 0, label: 'OK' }, { value: 1, label: 'Canceled' }, { value: 2, label: 'Unknown' },
  { value: 3, label: 'InvalidArgument' }, { value: 4, label: 'DeadlineExceeded' }, { value: 5, label: 'NotFound' },
  { value: 6, label: 'AlreadyExists' }, { value: 7, label: 'PermissionDenied' }, { value: 8, label: 'ResourceExhausted' },
  { value: 9, label: 'FailedPrecondition' }, { value: 10, label: 'Aborted' }, { value: 11, label: 'OutOfRange' },
  { value: 12, label: 'Unimplemented' }, { value: 13, label: 'Internal' }, { value: 14, label: 'Unavailable' },
  { value: 15, label: 'DataLoss' }, { value: 16, label: 'Unauthenticated' },
] as const;

const GRPC_CODE_NAME = new Map<number, string>(GRPC_CODES.map((c) => [c.value, c.label]));

// Canonical name for a gRPC code; empty for null/undefined, numeric string for unknown codes.
export function grpcCodeName(code?: number): string {
  if (code == null) return '';
  return GRPC_CODE_NAME.get(code) ?? String(code);
}

const SHELL_SAFE = /^[A-Za-z0-9_@%+=:,./-]+$/;

export function shellQuote(value: string): string {
  if (value === '') return "''";
  if (SHELL_SAFE.test(value)) return value;
  return `'${value.replaceAll("'", "'\\''")}'`;
}

const DEFAULT_GRPC_ADDR = 'localhost:4770';

export function dialableAddr(addr?: string): string {
  const a = addr?.trim();
  if (!a) return DEFAULT_GRPC_ADDR;
  return a.replace(/^(0\.0\.0\.0|\[::\]|):/, 'localhost:');
}

export function grpcurlCommand(
  record: Pick<CallRecord, 'service' | 'method' | 'requests'>,
  addr?: string,
): string {
  const msgs = record.requests?.length ? record.requests : [{}];
  const data = msgs.map((m) => JSON.stringify(m)).join('\n');
  return `grpcurl -plaintext -d ${shellQuote(data)} ${shellQuote(dialableAddr(addr))} ${shellQuote(`${record.service}/${record.method}`)}`;
}
