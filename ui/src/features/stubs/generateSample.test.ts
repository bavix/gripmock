import { describe, it, expect } from 'vitest';
import { generateSample } from './generateSample';
import type { ProtoMessageSchema, ProtoFieldSchema } from '../../lib/types';

function field(over: Partial<ProtoFieldSchema>): ProtoFieldSchema {
  return { name: 'f', jsonName: 'f', number: 1, kind: 'string', cardinality: 'optional', ...over };
}

function maxDepth(v: unknown): number {
  if (Array.isArray(v)) return v.length ? 1 + Math.max(...v.map(maxDepth)) : 1;
  if (v && typeof v === 'object') {
    const vals = Object.values(v);
    return vals.length ? 1 + Math.max(...vals.map(maxDepth)) : 1;
  }
  return 0;
}

describe('generateSample', () => {
  it('does not stack-overflow on a self-referential message (regression)', () => {
    // message Node { Node child = 1; }
    const node: ProtoMessageSchema = { typeName: 'pkg.Node', fields: [] };
    node.fields = [field({ name: 'child', jsonName: 'child', kind: 'message', typeName: 'pkg.Node', message: node })];

    let out!: Record<string, unknown>;
    expect(() => { out = generateSample(node); }).not.toThrow();
    // Depth is capped (MAX_DEPTH = 8), not infinite.
    expect(maxDepth(out)).toBeLessThanOrEqual(10);
  });

  it('handles mutually-recursive messages without overflow', () => {
    const a: ProtoMessageSchema = { typeName: 'pkg.A', fields: [] };
    const b: ProtoMessageSchema = { typeName: 'pkg.B', fields: [] };
    a.fields = [field({ name: 'b', jsonName: 'b', kind: 'message', typeName: 'pkg.B', message: b })];
    b.fields = [field({ name: 'a', jsonName: 'a', kind: 'message', typeName: 'pkg.A', message: a })];

    expect(() => generateSample(a)).not.toThrow();
  });

  it('still generates plain scalar fields', () => {
    const schema: ProtoMessageSchema = {
      typeName: 'pkg.M',
      fields: [field({ name: 'name', jsonName: 'name', kind: 'string' }), field({ name: 'age', jsonName: 'age', kind: 'int32' })],
    };
    expect(generateSample(schema)).toEqual({ name: 'sample', age: 42 });
  });
});
