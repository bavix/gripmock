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

export function generateSample(schema: ProtoMessageSchema | null | undefined): Record<string, unknown> {
  if (!schema?.fields?.length) return {};
  const result: Record<string, unknown> = {};
  const usedOneofs = new Set<string>();

  for (const field of schema.fields) {
    if (field.oneof) {
      if (usedOneofs.has(field.oneof)) continue;
      usedOneofs.add(field.oneof);
    }
    const value = generateField(field);
    if (value !== undefined) result[field.jsonName || field.name] = value;
  }
  return result;
}

function generateField(field: ProtoFieldSchema): unknown {
  if (field.cardinality === 'repeated') {
    if (field.map) return generateMap(field);
    return [generateSingle(field), generateSingle(field)];
  }
  return generateSingle(field);
}

function generateMap(field: ProtoFieldSchema): Record<string, unknown> {
  const key = field.mapKeyKind === 'int64' || field.mapKeyKind === 'uint64' ? '1' : 'key1';
  return { [key]: generateByKind(field.mapValueKind || 'string', field.mapValueTypeName, null) };
}

function generateSingle(field: ProtoFieldSchema): unknown {
  return generateByKind(field.kind, field.typeName, field);
}

function generateByKind(kind: string, typeName?: string, field?: ProtoFieldSchema | null): unknown {
  if (typeName) {
    if (wktSamples[typeName]) return wktSamples[typeName]();
    if (field?.enumValues?.length) return field.enumValues.find((v) => v !== 'UNSPECIFIED' && v !== 'UNKNOWN') || field.enumValues[0];
    if (!typeName.startsWith('google.')) {
      if (field?.message) return generateSample(field.message);
      return { [`${typeName.split('.').pop() || 'value'}`]: '...' };
    }
  }
  if (field?.enumValues?.length) return field.enumValues.find((v) => v !== 'UNSPECIFIED' && v !== 'UNKNOWN') || field.enumValues[0];
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
