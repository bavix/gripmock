// Offset for the next infinite-query page, or undefined to stop.
//
// Anchors on the FIRST page's total (authoritative) rather than the latest
// page's, so a later page that omits X-Total-Count (api falls back to its own
// length) does not spuriously satisfy loaded >= total and stop early. Stops on
// an empty page as a guard against an offset loop if the total shrank mid-scroll
// (concurrent deletes) while `loaded` never catches up.
export function nextPageOffset(
  lastPageLen: number,
  loaded: number,
  firstPageTotal: number,
): number | undefined {
  if (lastPageLen === 0) return undefined;
  return loaded < firstPageTotal ? loaded : undefined;
}
