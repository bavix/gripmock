// @vitest-environment jsdom
import { describe, it, expect } from 'vitest';
import { parseFiles } from './ImportStubs';

const file = (name: string, body: string) => new File([body], name, { type: 'text/plain' });

describe('parseFiles', () => {
  it('parses a JSON array of stubs', async () => {
    const { stubs, fileErrors } = await parseFiles([file('s.json', '[{"service":"A","method":"m"},{"service":"B","method":"n"}]')]);
    expect(fileErrors).toEqual([]);
    expect(stubs).toHaveLength(2);
    expect(stubs[0].service).toBe('A');
  });

  it('wraps a single JSON object into an array', async () => {
    const { stubs } = await parseFiles([file('s.json', '{"service":"A","method":"m"}')]);
    expect(stubs).toHaveLength(1);
  });

  it('parses YAML', async () => {
    const { stubs, fileErrors } = await parseFiles([file('s.yaml', 'service: A\nmethod: m\n')]);
    expect(fileErrors).toEqual([]);
    expect(stubs[0]).toMatchObject({ service: 'A', method: 'm' });
  });

  it('reports a file error for malformed JSON without dropping other files', async () => {
    const { stubs, fileErrors } = await parseFiles([
      file('bad.json', '{not json'),
      file('good.json', '{"service":"C","method":"m"}'),
    ]);
    expect(fileErrors).toHaveLength(1);
    expect(fileErrors[0]).toContain('bad.json');
    expect(stubs).toHaveLength(1);
    expect(stubs[0].service).toBe('C');
  });

  it('flags non-object array entries', async () => {
    const { fileErrors } = await parseFiles([file('s.json', '[1, 2]')]);
    expect(fileErrors.length).toBe(2);
  });
});
