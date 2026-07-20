import { Fragment, type ReactNode } from 'react';

const YAML_COLORS: Record<string, string> = {
  key: '#3b82f6',
  str: '#22c55e',
  num: '#f59e0b',
  bool: '#a855f7',
  null: '#94a3b8',
  punct: '#64748b',
};

// Renders YAML as colored React nodes. No HTML string / dangerouslySetInnerHTML,
// so user content can never become markup (React escapes text automatically).
export function highlightYaml(yaml: string): ReactNode {
  const lines = yaml.split('\n');
  return lines.map((line, i) => (
    <Fragment key={i}>
      {i > 0 && '\n'}
      {renderLine(line)}
    </Fragment>
  ));
}

function renderLine(line: string): ReactNode {
  const indent = line.match(/^\s*/)?.[0] ?? '';
  const rest = line.slice(indent.length);

  if (!rest.trim()) return line;

  const kvMatch = rest.match(/^([\w./@:-]+?)(:)(\s*)(.*)$/);
  if (kvMatch) {
    const [, key, , space, value] = kvMatch;
    return (
      <>
        {indent}
        <span style={{ color: YAML_COLORS.key }}>{key}</span>
        <span style={{ color: YAML_COLORS.punct }}>:</span>
        {space}
        {colorYamlValue(value)}
      </>
    );
  }

  const arrMatch = rest.match(/^(- )(.*)$/);
  if (arrMatch) {
    const [, , value] = arrMatch;
    return (
      <>
        {indent}
        <span style={{ color: YAML_COLORS.punct }}>-</span>
        {' '}
        {colorYamlValue(value)}
      </>
    );
  }

  return line;
}

function colorYamlValue(value: string): ReactNode {
  if (!value) return '';

  const strMatch = value.match(/^"((?:[^"\\]|\\.)*)"$/);
  if (strMatch) {
    return <span style={{ color: YAML_COLORS.str }}>{`"${strMatch[1]}"`}</span>;
  }

  if (/^-?\d+(\.\d+)?$/.test(value)) {
    return <span style={{ color: YAML_COLORS.num }}>{value}</span>;
  }

  if (/^(true|false)$/.test(value)) {
    return <span style={{ color: YAML_COLORS.bool }}>{value}</span>;
  }

  if (value === 'null') {
    return <span style={{ color: YAML_COLORS.null }}>null</span>;
  }

  if (value === '[]' || value === '{}') {
    return <span style={{ color: YAML_COLORS.punct }}>{value}</span>;
  }

  return value;
}
