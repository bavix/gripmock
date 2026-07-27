import type { ProtoMessageSchema, ProtoFieldSchema } from '../../lib/types';

const wktSamples: Record<string, () => unknown> = {
  'google.protobuf.Timestamp': () => '2024-06-01T12:00:00Z',
  'google.protobuf.Duration': () => '5s',
  'google.protobuf.Struct': () => ({ key: 'value' }),
  'google.protobuf.Any': () => ({ '@type': 'type.googleapis.com/example.Message', value: '...' }),
  'google.protobuf.Empty': () => ({}),
  'google.protobuf.StringValue': () => 'string',
  'google.protobuf.Int32Value': () => 42,
  'google.protobuf.UInt32Value': () => 42,
  'google.protobuf.Int64Value': () => '42',
  'google.protobuf.UInt64Value': () => '42',
  'google.protobuf.FloatValue': () => 3.14,
  'google.protobuf.DoubleValue': () => 3.14,
  'google.protobuf.BoolValue': () => true,
  'google.protobuf.BytesValue': () => 'Ynl0ZXM=',
  'google.protobuf.FieldMask': () => ({ paths: ['field1', 'field2'] }),
  'google.type.Money': () => ({ currencyCode: 'USD', units: 100, nanos: 500000000 }),
  'google.type.Date': () => ({ year: 2024, month: 6, day: 1 }),
  'google.type.TimeOfDay': () => ({ hours: 12, minutes: 0, seconds: 0, nanos: 0 }),
  'google.type.DateTime': () => ({ year: 2024, month: 6, day: 1, hours: 12, minutes: 0 }),
  'google.type.Color': () => ({ red: 0.1, green: 0.2, blue: 0.3, alpha: 1.0 }),
  'google.type.LatLng': () => ({ latitude: 48.8584, longitude: 2.2945 }),
  'google.type.PostalAddress': () => ({
    regionCode: 'US',
    postalCode: '94043',
    administrativeArea: 'CA',
    locality: 'Mountain View',
    addressLines: ['1600 Amphitheatre Pkwy'],
  }),
  'google.type.Interval': () => ({ startTime: '2024-06-01T00:00:00Z', endTime: '2024-06-02T00:00:00Z' }),
  'google.rpc.ErrorInfo': () => ({ reason: 'ERROR_REASON', domain: 'example.local', metadata: { key: 'value' } }),
  'google.rpc.BadRequest': () => ({ fieldViolations: [{ field: 'fieldName', description: 'must not be empty' }] }),
};

// Depth cap so a self-referential message schema (a field of its own type, e.g.
// a tree/linked-list proto) can't recurse forever and blow the stack.
const MAX_DEPTH = 8;

// Prefer a meaningful enum value for samples over the zero/placeholder member,
// which is conventionally named *_UNSPECIFIED / *_UNKNOWN (proto style guide).
function firstMeaningfulEnum(values: string[]): string {
  return values.find((v) => !/(?:^|_)(UNSPECIFIED|UNKNOWN)$/.test(v)) ?? values[0];
}

export function generateSample(schema: ProtoMessageSchema | null | undefined, depth = 0): Record<string, unknown> {
  if (!schema?.fields?.length || depth >= MAX_DEPTH) return {};
  const result: Record<string, unknown> = {};
  const usedOneofs = new Set<string>();

  for (const field of schema.fields) {
    if (field.oneof) {
      if (usedOneofs.has(field.oneof)) continue;
      usedOneofs.add(field.oneof);
    }
    const value = generateField(field, depth);
    if (value !== undefined) result[field.jsonName || field.name] = value;
  }
  return result;
}

function generateField(field: ProtoFieldSchema, depth: number): unknown {
  if (field.cardinality === 'repeated') {
    if (field.map) return generateMap(field, depth);
    return [generateSingle(field, depth), generateSingle(field, depth)];
  }
  return generateSingle(field, depth);
}

function generateMap(field: ProtoFieldSchema, depth: number): Record<string, unknown> {
  const key = field.mapKeyKind === 'int64' || field.mapKeyKind === 'uint64' ? '1' : 'key1';
  return { [key]: generateByKind(field.mapValueKind || 'string', field.mapValueTypeName, null, depth) };
}

function generateSingle(field: ProtoFieldSchema, depth: number): unknown {
  return generateByKind(field.kind, field.typeName, field, depth);
}

function generateByKind(kind: string, typeName: string | undefined, field: ProtoFieldSchema | null | undefined, depth: number): unknown {
  if (typeName) {
    if (wktSamples[typeName]) return wktSamples[typeName]();
    if (field?.enumValues?.length) return firstMeaningfulEnum(field.enumValues);
    if (!typeName.startsWith('google.')) {
      if (field?.message) return generateSample(field.message, depth + 1);
      return { [`${typeName.split('.').pop() || 'value'}`]: '...' };
    }
  }
  if (field?.enumValues?.length) return firstMeaningfulEnum(field.enumValues);
  switch (kind) {
    case 'string': return 'sample';
    case 'int32': case 'int64': case 'sint32': case 'sint64':
    case 'sfixed32': case 'sfixed64': case 'uint32': case 'uint64':
    case 'fixed32': case 'fixed64': return 42;
    case 'float': case 'double': return 3.14;
    case 'bool': return true;
    case 'bytes': return 'Ynl0ZXM=';
    case 'message': return {};
    default: return null;
  }
}
