// Count label for the Stubs header. When a client-only filter (matcher-kind
// chips) is active, `total` is the server-wide UNFILTERED count, so comparing
// the filtered `shownCount` against it ("60 of 500") is misleading — those 60
// are the matches within the loaded window, not out of 500. Report the honest
// "N matching" instead.
export function stubCountLabel(shownCount: number, total: number, clientFiltered: boolean): string {
  if (clientFiltered) return `(${shownCount} matching)`;
  return shownCount < total ? `(${shownCount} of ${total})` : `(${total})`;
}
