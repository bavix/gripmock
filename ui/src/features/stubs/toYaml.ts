// A bare string is safe to emit unquoted only if it has no YAML-special
// characters AND would not be re-parsed as a non-string scalar (number,
// boolean, null) on load. Otherwise it must be quoted, or a round-trip through
// the server silently changes its type (e.g. "-5" → -5, "on" → true).
const YAML_NUMERIC = /^[-+]?(\d+\.?\d*|\.\d+)([eE][-+]?\d+)?$/;
const YAML_HEX_OCT_BIN = /^[-+]?0(x[0-9a-f]+|o[0-7]+|b[01]+)$/i;
const YAML_BOOL_NULL = /^(y|n|yes|no|on|off|true|false|null|~)$/i;
const YAML_INF_NAN = /^[-+]?\.(inf|nan)$/i;

function looksLikeScalar(s: string): boolean {
  return YAML_NUMERIC.test(s) || YAML_HEX_OCT_BIN.test(s) || YAML_BOOL_NULL.test(s) || YAML_INF_NAN.test(s);
}

// A plain scalar may not START with a reserved YAML indicator (`@`, `:` here),
// though those chars are fine mid-string — so the first char uses a narrower set.
const PLAIN = /^[a-zA-Z0-9_./\-+][a-zA-Z0-9_./@:\-+]*$/;

function dquote(s: string): string {
  return `"${s.replaceAll('\\', String.raw`\\`).replaceAll('"', '\\"')}"`;
}

function quoteStr(s: string): string {
  if (PLAIN.test(s) && !looksLikeScalar(s)) return s;
  return dquote(s);
}

// Keys only need quoting for structurally-unsafe characters (spaces, YAML
// specials). Unlike values they are not subject to scalar coercion here, so a
// numeric- or boolean-looking key stays unquoted (matches user expectation).
function quoteKey(s: string): string {
  if (PLAIN.test(s)) return s;
  return dquote(s);
}

// A multi-line string must not be emitted as a double-quoted scalar: the raw
// newlines land at column 0 and break the document. A block scalar keeps it
// readable and pastable into a stub file.
function blockScalar(s: string, pad: string): string {
  const keepsNewline = s.endsWith('\n');
  const body = (keepsNewline ? s.slice(0, -1) : s)
    .split('\n')
    .map((line) => (line ? pad + line : ''))
    .join('\n');

  return `${keepsNewline ? '|' : '|-'}\n${body}`;
}

function isEmpty(v: unknown): boolean {
  if (v === null || v === undefined) return true;
  if (typeof v === 'string' && v === '') return true;
  if (typeof v === 'number' && v === 0) return false;
  if (Array.isArray(v) && v.length === 0) return true;
  if (typeof v === 'object' && !Array.isArray(v) && Object.keys(v as Record<string, unknown>).length === 0) return true;
  return false;
}

function clean(val: unknown): unknown {
  if (val === null || val === undefined) return undefined;
  if (Array.isArray(val)) {
    const cleaned = val.map(clean).filter((v) => v !== undefined);
    return cleaned.length > 0 ? cleaned : undefined;
  }
  if (typeof val === 'object') {
    const obj = val as Record<string, unknown>;
    const cleaned: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(obj)) {
      const cv = clean(v);
      if (cv !== undefined && !isEmpty(cv)) cleaned[k] = cv;
    }
    return Object.keys(cleaned).length > 0 ? cleaned : undefined;
  }
  return val;
}

function toYamlVal(val: unknown, indent: number): string {
  const pad = '  '.repeat(indent);
  if (val === null || val === undefined) return 'null';
  if (typeof val === 'string') return val.includes('\n') ? blockScalar(val, pad) : quoteStr(val);
  if (typeof val === 'number' || typeof val === 'boolean') return String(val);
  if (Array.isArray(val)) {
    if (val.length === 0) return '[]';
    return val.map((v) => {
      const isObj = typeof v === 'object' && v !== null && !Array.isArray(v);
      const child = toYamlVal(v, indent + 1);
      const objChild = `\n${pad}  ${child.trim()}`;
      return `\n${pad}- ${isObj ? objChild : child.trimStart()}`;
    }).join('');
  }
  if (typeof val === 'object') {
    const entries = Object.entries(val as Record<string, unknown>).filter(([, v]) => v !== undefined);
    if (entries.length === 0) return '{}';
    return entries.map(([k, v]) => {
      const vStr = toYamlVal(v, indent + 1);
      const multi = typeof v === 'object' && v !== null;
      const inlineVal = ` ${vStr}`;
      // Keys need the same quoting as scalar values — a key with a space, colon
      // or YAML-special char emitted raw produces invalid YAML.
      return `\n${pad}${quoteKey(k)}:${multi && vStr.startsWith('\n') ? vStr : inlineVal}`;
    }).join('');
  }
  return typeof val === 'object' ? JSON.stringify(val) : String(val);
}

export function toYaml(obj: unknown): string {
  const cleaned = clean(obj);
  if (!cleaned) return '';
  const r = toYamlVal(cleaned, 0);
  return r.startsWith('\n') ? r.slice(1) : r;
}
