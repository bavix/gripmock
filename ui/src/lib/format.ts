// Prefix a bare numeric version with "v" (leave named versions like "development" as-is).
export const versionLabel = (version: string): string => `${/^\d/.test(version) ? 'v' : ''}${version}`;

// Comparator: newest timestamp first.
export const byTimestampDesc = (a: { timestamp: string }, b: { timestamp: string }): number =>
  new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime();
