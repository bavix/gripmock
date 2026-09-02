import { describe, it, expect } from 'vitest';
import { grpcCodeName, shellQuote, dialableAddr, grpcurlCommand } from './grpc';

describe('grpcCodeName', () => {
  it('names known codes and passes unknown ones through', () => {
    expect(grpcCodeName(0)).toBe('OK');
    expect(grpcCodeName(5)).toBe('NotFound');
    expect(grpcCodeName(99)).toBe('99');
    expect(grpcCodeName(undefined)).toBe('');
  });
});

describe('shellQuote', () => {
  it('leaves plain arguments unquoted', () => {
    expect(shellQuote('localhost:4770')).toBe('localhost:4770');
    expect(shellQuote('simple.Gripmock/SayHello')).toBe('simple.Gripmock/SayHello');
  });

  it('quotes empty strings', () => {
    expect(shellQuote('')).toBe("''");
  });

  it('quotes anything with shell metacharacters', () => {
    expect(shellQuote('{"name":"bob"}')).toBe(`'{"name":"bob"}'`);
    expect(shellQuote('a b')).toBe("'a b'");
    expect(shellQuote('$(id)')).toBe("'$(id)'");
  });

  it('closes, escapes and reopens embedded single quotes', () => {
    expect(shellQuote("don't")).toBe(`'don'\\''t'`);
  });
});

describe('dialableAddr', () => {
  it('rewrites wildcard binds to a host a client can dial', () => {
    expect(dialableAddr('0.0.0.0:4770')).toBe('localhost:4770');
    expect(dialableAddr('[::]:4770')).toBe('localhost:4770');
    expect(dialableAddr(':4770')).toBe('localhost:4770');
  });

  it('keeps a real host and falls back when the server reports none', () => {
    expect(dialableAddr('mock.internal:9000')).toBe('mock.internal:9000');
    expect(dialableAddr(undefined)).toBe('localhost:4770');
    expect(dialableAddr('  ')).toBe('localhost:4770');
  });
});

describe('grpcurlCommand', () => {
  const call = (requests?: Record<string, unknown>[]) => ({
    service: 'simple.Gripmock',
    method: 'SayHello',
    requests,
  });

  it('builds a runnable command for a single request', () => {
    expect(grpcurlCommand(call([{ name: 'bob' }]), '0.0.0.0:4770')).toBe(
      `grpcurl -plaintext -d '{"name":"bob"}' localhost:4770 simple.Gripmock/SayHello`,
    );
  });

  it('uses the address the dashboard reports instead of a hard-coded port', () => {
    expect(grpcurlCommand(call([{ name: 'bob' }]), 'mock.internal:9999')).toContain(' mock.internal:9999 ');
    expect(grpcurlCommand(call([{ name: 'bob' }]))).toContain(' localhost:4770 ');
  });

  it('newline-separates a client-streaming request sequence', () => {
    expect(grpcurlCommand(call([{ n: 1 }, { n: 2 }]))).toContain(`'{"n":1}\n{"n":2}'`);
  });

  it('falls back to an empty message when nothing was recorded', () => {
    expect(grpcurlCommand(call())).toContain("-d '{}'");
    expect(grpcurlCommand(call([]))).toContain("-d '{}'");
  });

  it('cannot be escaped by a quote in the recorded payload', () => {
    const payload = { name: "'; rm -rf ~; echo '" };
    const cmd = grpcurlCommand(call([payload]), 'localhost:4770');

    expect(shellWords(cmd)).toEqual([
      'grpcurl', '-plaintext', '-d', JSON.stringify(payload),
      'localhost:4770', 'simple.Gripmock/SayHello',
    ]);
  });

  it('keeps a payload with a backslash or newline in one argument', () => {
    const payload = { path: 'C:\\tmp', note: 'line1\nline2' };
    const words = shellWords(grpcurlCommand(call([payload])));

    expect(words).toHaveLength(6);
    expect(words[3]).toBe(JSON.stringify(payload));
  });
});

function shellWords(command: string): string[] {
  const words: string[] = [];
  let word = '';
  let started = false;
  let quoted = false;

  for (let i = 0; i < command.length; i++) {
    const ch = command[i];

    if (quoted) {
      if (ch === "'") quoted = false;
      else word += ch;
      continue;
    }

    if (ch === "'") { quoted = true; started = true; continue; }
    if (ch === '\\') { word += command[++i] ?? ''; started = true; continue; }
    if (ch === ' ' || ch === '\n') {
      if (started) { words.push(word); word = ''; started = false; }
      continue;
    }

    word += ch;
    started = true;
  }

  if (started) words.push(word);

  return words;
}
