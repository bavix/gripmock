function quoteStr(s: string): string {
  if (/^[a-zA-Z0-9_./@:\-+]+$/.test(s) && !/^(true|false|null|yes|no|\d+)$/.test(s)) return s;
  return `"${s.replaceAll('\\', String.raw`\\`).replaceAll('"', '\\"')}"`;
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
  if (typeof val === 'string') return quoteStr(val);
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
      return `\n${pad}${k}:${multi && vStr.startsWith('\n') ? vStr : inlineVal}`;
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
