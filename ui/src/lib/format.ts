// Prefix a bare numeric version with "v" (leave named versions like "development" as-is).
export const versionLabel = (version: string): string => `${/^\d/.test(version) ? 'v' : ''}${version}`;

// Comparator: newest timestamp first.
export const byTimestampDesc = (a: { timestamp: string }, b: { timestamp: string }): number =>
  new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime();

export const callTime = (timestamp: string): string =>
  new Date(timestamp).toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    fractionalSecondDigits: 3,
  });
