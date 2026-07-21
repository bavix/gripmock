import { useState, useMemo, useRef, useEffect, useCallback } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Search, LayoutGrid, List, X, Copy, Check, Edit3, Play, Filter, Download, Trash2, Loader2, Upload } from 'lucide-react';
import { SlideOver } from '../components/shared/SlideOver';
import { ImportStubs } from '../features/stubs/ImportStubs';
import { api } from '../lib/api';
import { useCreateStub, useInfiniteStubs, useStubsPage, type StubListFilters } from '../hooks/useStubs';
import { useCopy } from '../hooks/useCopy';
import { useServices } from '../hooks/useServices';
import { useSmartSearch } from '../hooks/useSearch';
import { btn, colors } from '../lib/theme';
import type { Stub } from '../lib/types';
import { compactPreview, prettyJson, matcherTypes, MATCHER_COLORS, outputKind, stubRequestExample, streamKind, hasContent, requestMessages, responseMessages, matcherEntries } from '../lib/stub';
import { stashClone } from '../lib/clone';
import { DataTable } from '../components/table/DataTable';
import { useToast } from '../components/shared/Toast';
import type { ColumnDef } from '@tanstack/react-table';

interface Props { filter?: string; }

const MATCHER_TYPES = ['equals', 'contains', 'matches', 'glob', 'anyOf', 'any'] as const;
const CARDS_PAGE = 60;
const TABLE_PAGE = 50;
const iconBtn: React.CSSProperties = { display: 'inline-flex', alignItems: 'center', justifyContent: 'center', width: 28, height: 28, borderRadius: 6, border: 'none', background: 'transparent', cursor: 'pointer' };
const smallBtn: React.CSSProperties = { display: 'inline-flex', alignItems: 'center', gap: 3, padding: '3px 8px', fontSize: 11, borderRadius: 4, cursor: 'pointer', border: '1px solid var(--border)', background: 'transparent', color: 'var(--text-secondary)' };

function LoadMoreSentinel({ onVisible, active }: { onVisible: () => void; active: boolean }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!active || !ref.current) return;
    const io = new IntersectionObserver(
      (entries) => { if (entries.some((e) => e.isIntersecting)) onVisible(); },
      { rootMargin: '600px' },
    );
    io.observe(ref.current);
    return () => io.disconnect();
  }, [onVisible, active]);
  return <div ref={ref} style={{ height: 1 }} />;
}

export function StubsList({ filter }: Props) {
  const navigate = useNavigate();
  const toast = useToast();
  const [params] = useSearchParams();
  const [viewMode, setViewMode] = useState<'cards' | 'table'>('cards');
  const [searchText, setSearchText] = useState(params.get('q') || '');
  const [svcFilter, setSvcFilter] = useState(params.get('service') || '');
  const [methodFilter, setMethodFilter] = useState(params.get('method') || '');
  const [srcFilter, setSrcFilter] = useState(params.get('source') || '');
  const [activeMatchers, setActiveMatchers] = useState<Set<string>>(new Set());
  const [showFilters, setShowFilters] = useState(false);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [smartResults, setSmartResults] = useState<{ results: Stub[]; query: string } | null>(null);
  const [smartSearching, setSmartSearching] = useState(false);
  const qc = useQueryClient();
  const createMut = useCreateStub();
  const { data: svcList } = useServices();

  // service/method -> methodType, for streaming badges on stubs. Keyed by both
  // the FQN and the bare service name, since a stub's `service` may omit the package.
  const methodType = useMemo(() => {
    const m: Record<string, string> = {};
    for (const s of svcList ?? []) for (const mm of s.methods ?? []) {
      m[`${s.id}/${mm.name}`] = mm.methodType;
      if (s.name) m[`${s.name}/${mm.name}`] = mm.methodType;
    }
    return m;
  }, [svcList]);
  const mtOf = (s: Stub) => methodType[`${s.service}/${s.method}`];

  // Server-side query text: debounced mirror of the search box.
  const [serverQ, setServerQ] = useState(searchText.trim());
  const [page, setPage] = useState(0);
  const filters: StubListFilters = useMemo(
    () => ({ service: svcFilter, method: methodFilter, source: srcFilter, q: serverQ }),
    [svcFilter, methodFilter, srcFilter, serverQ],
  );
  useEffect(() => { setPage(0); }, [filters]);
  // Selection is scoped to the loaded page (bulk-delete filters the current page
  // only) — drop it when the page changes so "N selected" never over-promises.
  useEffect(() => { setSelected(new Set()); }, [page]);

  // Main view is server-paginated (thousands-scale). Used/unused endpoints
  // have no pagination — they keep the legacy full fetch.
  const infinite = useInfiniteStubs(filters, CARDS_PAGE);
  const tablePage = useStubsPage(filters, TABLE_PAGE, page * TABLE_PAGE);
  const legacy = useQuery({
    queryKey: ['stubs', filter],
    queryFn: () => api.get<Stub[]>(`/stubs/${filter}`),
    retry: 1, staleTime: 5_000,
    enabled: !!filter,
  });

  const isMain = !filter;
  const all: Stub[] | undefined = isMain
    ? (viewMode === 'cards'
        ? infinite.data?.pages.flatMap((p) => p.data)
        : tablePage.data?.data)
    : legacy.data;
  const total = isMain
    ? (viewMode === 'cards'
        ? infinite.data?.pages[0]?.total ?? 0
        : tablePage.data?.total ?? 0)
    : legacy.data?.length ?? 0;
  const isLoading = isMain
    ? (viewMode === 'cards' ? infinite.isLoading : tablePage.isLoading)
    : legacy.isLoading;

  const deleteMut = useMutation({
    mutationFn: async (stub: Stub) => { await api.post('/stubs/batchDelete', [stub.id]); return stub; },
    onSuccess: (d) => {
      qc.invalidateQueries({ queryKey: ['stubs'] });
      toast.show(`Deleted ${d.service}/${d.method}`, {
        label: 'Undo',
        onClick: async () => {
          try { await createMut.mutateAsync({ service: d.service, method: d.method, priority: d.priority, input: d.input, output: d.output }); qc.invalidateQueries({ queryKey: ['stubs'] }); } catch {}
        },
      });
    },
  });

  // Bulk delete: ONE batchDelete call for all selected ids, with a single Undo
  // that re-creates the full stub objects.
  const bulkDelete = async (items: Stub[]) => {
    if (items.length === 0) return;
    const ids = items.map((s) => s.id);
    await api.post('/stubs/batchDelete', ids);
    qc.invalidateQueries({ queryKey: ['stubs'] });
    setSelected(new Set());
    toast.show(`Deleted ${items.length} stub${items.length > 1 ? 's' : ''}`, {
      label: 'Undo',
      onClick: async () => {
        try {
          await api.post('/stubs', items.map(({ id: _id, ...rest }) => rest));
          qc.invalidateQueries({ queryKey: ['stubs'] });
        } catch {}
      },
    });
  };

  // Filter catalog comes from the registered services (not from loaded stubs,
  // which are just one page of a potentially huge set).
  const services = useMemo(() => (svcList ?? []).map((s) => s.id).sort((a, b) => a.localeCompare(b)), [svcList]);
  const methodsByService = useMemo(() => {
    const m: Record<string, Set<string>> = {};
    for (const s of svcList ?? []) for (const mm of s.methods ?? []) (m[s.id] ??= new Set()).add(mm.name);
    return m;
  }, [svcList]);
  const hasFilters = !!(svcFilter || methodFilter || srcFilter || activeMatchers.size > 0 || searchText);

  // Residual client-side filtering over the LOADED items: matcher-kind chips
  // everywhere; text/service/method too on legacy (unpaginated) views.
  const filtered = useMemo(() => {
    if (!all) return [];
    return all.filter((s) => {
      if (activeMatchers.size > 0) {
        const types = matcherTypes(s);
        if (![...activeMatchers].some((t) => types.includes(t))) return false;
      }
      if (isMain) return true;
      if (svcFilter && s.service !== svcFilter) return false;
      if (methodFilter && s.method !== methodFilter) return false;
      if (srcFilter && s.source !== srcFilter) return false;
      if (!searchText) return true;
      const q = searchText.toLowerCase();
      return [s.service, s.method, s.id, compactPreview(s.input, 9999), compactPreview(s.output, 9999)].some((f) => f.toLowerCase().includes(q));
    });
  }, [all, isMain, searchText, svcFilter, methodFilter, srcFilter, activeMatchers]);

  const toggleMatcher = (t: string) => setActiveMatchers((p) => {
    const n = new Set(p); if (n.has(t)) n.delete(t); else n.add(t); return n;
  });

  const smartSearch = useSmartSearch();
  const searchTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const handleSearchChange = (value: string) => {
    setSearchText(value);
    setSmartResults(null);
    clearTimeout(searchTimer.current);

    const trimmed = value.trim();
    const isUUID = /^[0-9a-f]{8,}$/i.test(trimmed.replace(/-/g, ''));
    const hasJSON = trimmed.includes('{') && trimmed.includes('}');
    const hasEndpoint = trimmed.includes('/') || trimmed.includes('.');
    const smart = trimmed.length >= 3 && (isUUID || hasJSON || (hasEndpoint && trimmed.length > 5));

    // One debounce drives both the server-side q filter and the smart search.
    searchTimer.current = setTimeout(async () => {
      setServerQ(trimmed.length >= 2 ? trimmed : '');
      if (!smart) return;
      setSmartSearching(true);
      try {
        const result = await smartSearch.search(trimmed);
        if (result.results.length > 0) setSmartResults({ results: result.results as any, query: trimmed });
      } catch {}
      setSmartSearching(false);
    }, 300);
  };

  const clearAllFilters = () => { setSvcFilter(''); setMethodFilter(''); setSrcFilter(''); setActiveMatchers(new Set()); setSearchText(''); setServerQ(''); setShowFilters(false); setSmartResults(null); };

  const loadMore = useCallback(() => {
    if (infinite.hasNextPage && !infinite.isFetchingNextPage) infinite.fetchNextPage();
  }, [infinite]);

  const shownCards = smartResults?.results ?? filtered;

  const shared = (
    <>
      <CardsHeader total={total} shownCount={filtered.length} filters={filters} filter={filter} viewMode={viewMode} setViewMode={setViewMode} navigate={navigate} onDeleteAllUnused={() => bulkDelete(filtered)} unusedCount={filter === 'unused' ? filtered.length : 0} />
      <SearchRow searchText={searchText} setSearchText={setSearchText} showFilters={showFilters} setShowFilters={setShowFilters} hasFilters={hasFilters} onSmartSearch={handleSearchChange} smartSearching={smartSearching} />
      <MatcherChips activeMatchers={activeMatchers} toggleMatcher={toggleMatcher} setActiveMatchers={setActiveMatchers} />
      {showFilters && <FilterRow svcFilter={svcFilter} setSvcFilter={setSvcFilter} methodFilter={methodFilter} setMethodFilter={setMethodFilter} srcFilter={srcFilter} setSrcFilter={setSrcFilter} services={services} methodsByService={methodsByService} />}
      <FilterBadges svcFilter={svcFilter} methodFilter={methodFilter} activeMatchers={activeMatchers} clearAll={clearAllFilters} toggleMatcher={toggleMatcher} onRemoveSvc={() => setSvcFilter('')} onRemoveMethod={() => setMethodFilter('')} />
      {smartResults && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '6px 10px', borderRadius: 6, border: '1px solid var(--accent)', background: `${colors.accent}08`, fontSize: 11, color: 'var(--text-secondary)' }}>
          <span>Search: <strong>{smartResults.results.length}</strong> stubs match <code style={{ background: 'var(--bg-tertiary)', padding: '1px 6px', borderRadius: 3 }}>{smartResults.query}</code></span>
          <div style={{ flex: 1 }} />
          <button onClick={() => { setSmartResults(null); setSearchText(''); setServerQ(''); }} className="btn btn-ghost" style={{ fontSize: 11, padding: '2px 8px', color: colors.accent }}>Clear</button>
        </div>
      )}
    </>
  );

  if (viewMode === 'cards') {
    return (
      <div className="page-enter" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {shared}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))', gap: 6 }}>
          {shownCards.map((s) => <StubCard key={s.id} stub={s} onDelete={(st) => deleteMut.mutate(st)} expandedId={expandedId} setExpandedId={setExpandedId} navigate={navigate} methodType={mtOf(s)} />)}
        </div>
        {isMain && !smartResults && <LoadMoreSentinel onVisible={loadMore} active={!!infinite.hasNextPage} />}
        {infinite.isFetchingNextPage && (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6, padding: 12, color: 'var(--text-muted)', fontSize: 12 }}>
            <Loader2 size={13} className="animate-spin" /> Loading more…
          </div>
        )}
        {!isLoading && shownCards.length === 0 && <div className="empty" style={{ padding: 32 }}>No stubs found</div>}
      </div>
    );
  }

  const columns = makeColumns(selected, setSelected);
  return (
    <div className="page-enter" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {shared}
      {selected.size > 0 && (
        <div style={{ padding: '5px 10px', borderRadius: 6, background: `${colors.accent}10`, border: '1px solid var(--accent)', fontSize: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
          <span>{selected.size} selected</span>
          <button onClick={() => setSelected(new Set())} style={{ fontSize: 11, padding: '2px 8px', borderRadius: 4, border: '1px solid var(--border)', background: 'var(--bg-primary)', cursor: 'pointer' }}>Clear</button>
          <button onClick={() => bulkDelete(all?.filter((s) => selected.has(s.id)) ?? [])}
            style={{ fontSize: 11, padding: '2px 8px', borderRadius: 4, border: '1px solid var(--error)', background: `${colors.error}10`, color: colors.error, cursor: 'pointer' }}>Delete {selected.size}</button>
        </div>
      )}
      <DataTable data={filtered} columns={columns} loading={isLoading} emptyMessage={isLoading ? 'Loading...' : 'No stubs'}
        manualPagination={isMain} rowCount={isMain ? total : undefined} pageIndex={page} pageSize={TABLE_PAGE} onPageChange={setPage}
        renderExpanded={(stub: Stub) => <ExpandedContent stub={stub} navigate={navigate} onDelete={(s) => deleteMut.mutate(s)} methodType={mtOf(stub)} />} />
    </div>
  );
}

function makeColumns(selected: Set<string>, setSelected: (fn: (prev: Set<string>) => Set<string>) => void): ColumnDef<Stub>[] {
  return [
    { id: '_sel', header: '', cell: ({ row }) => (
      <input type="checkbox" checked={selected.has(row.original.id)} onChange={() => setSelected((p) => { const n = new Set(p); if (n.has(row.original.id)) n.delete(row.original.id); else n.add(row.original.id); return n; })} />
    ), size: 32 },
    { id: 'id', header: 'ID', accessorKey: 'id', cell: (info) => (
      <span style={{ fontFamily: 'monospace', fontSize: 11, color: 'var(--text-muted)', cursor: 'pointer' }} title={info.getValue() as string}
        onClick={() => navigator.clipboard.writeText(info.getValue() as string)}>{(info.getValue() as string)?.slice(0, 8)}</span>
    )},
    { id: 'type', header: 'Type', cell: (info) => (
      <span style={{ display: 'inline-flex', gap: 3, flexWrap: 'wrap' }}>
        {matcherTypes(info.row.original).map((t) => (
          <span key={t} style={{ fontSize: 11, padding: '0 5px', borderRadius: 3, fontWeight: 600, background: `${MATCHER_COLORS[t] || '#64748b'}18`, color: MATCHER_COLORS[t] || '#64748b' }}>{t}</span>
        ))}
      </span>
    )},
    { id: 'priority', header: 'Prio', accessorKey: 'priority' },
    { id: 'input', header: 'Request', cell: (info) => {
      const req = requestMessages(info.row.original);
      const first = req[0] ? (matcherEntries(req[0])[0]?.value ?? req[0]) : undefined;
      return <span style={{ fontSize: 11, color: 'var(--text-secondary)', fontFamily: 'monospace' }} title={first ? (prettyJson(first) || undefined) : undefined}>
        {req.length > 1 && <span style={{ color: 'var(--accent-text)' }}>{req.length}× </span>}{req.length ? compactPreview(first) : '—'}
      </span>;
    }},
    { id: 'output', header: 'Response', cell: (info) => {
      const res = responseMessages(info.row.original);
      const o = info.row.original.output;
      if (o?.error || (o?.code ?? 0) > 0) return <span style={{ fontSize: 11, color: colors.error, fontFamily: 'monospace' }}>error {o?.code}</span>;
      return <span style={{ fontSize: 11, color: 'var(--text-secondary)', fontFamily: 'monospace' }} title={res[0] ? (prettyJson(res[0]) || undefined) : undefined}>
        {res.length > 1 && <span style={{ color: '#06b6d4' }}>{res.length}× </span>}{res.length ? compactPreview(res[0]) : '—'}
      </span>;
    }},
  ];
}

function downloadStubs(stubs: Stub[], name: string) {
  const clean = stubs.map(({ id: _id, ...rest }) => rest);
  const blob = new Blob([JSON.stringify(clean, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url; a.download = `${name}.json`; a.click();
  URL.revokeObjectURL(url);
}

// Export the FULL filtered set from the server (the page shows only a slice).
async function exportAll(filters: StubListFilters, filter: string | undefined) {
  const stubs = filter
    ? await api.get<Stub[]>(`/stubs/${filter}`)
    : await api.get<Stub[]>('/stubs', {
        service: filters.service || undefined, method: filters.method || undefined,
        source: filters.source || undefined, q: filters.q || undefined,
        limit: '100000',
      });
  downloadStubs(stubs ?? [], filter ? `stubs-${filter}` : 'stubs');
}

function CardsHeader({ total, shownCount, filters, filter, viewMode, setViewMode, navigate, onDeleteAllUnused, unusedCount }: {
  total: number; shownCount: number; filters: StubListFilters; filter?: string;
  viewMode: string; setViewMode: (v: 'cards'|'table') => void; navigate: (p: string) => void;
  onDeleteAllUnused: () => void; unusedCount: number;
}) {
  const [importOpen, setImportOpen] = useState(false);
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
      <h1>
        {filter === 'used' ? 'Used Stubs' : filter === 'unused' ? 'Unused Stubs' : 'Stubs'}
        <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400, marginLeft: 6 }}>
          {shownCount < total ? `(${shownCount} of ${total})` : `(${total})`}
        </span>
      </h1>
      <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
        {filter === 'unused' && unusedCount > 0 && (
          <button onClick={onDeleteAllUnused} className="btn btn-sm" style={{ color: colors.error }} title="Delete all unused stubs"><Trash2 size={13} /> Delete all</button>
        )}
        {!filter && <button onClick={() => setImportOpen(true)} className="icon-btn" title="Import stubs from JSON/YAML"><Upload size={15} /></button>}
        <button onClick={() => exportAll(filters, filter)} className="icon-btn" title="Export all matching stubs as JSON" disabled={total === 0}><Download size={15} /></button>
        <button onClick={() => setViewMode('cards')} className={`icon-btn ${viewMode === 'cards' ? 'active' : ''}`} title="Cards"><LayoutGrid size={15} /></button>
        <button onClick={() => setViewMode('table')} className={`icon-btn ${viewMode === 'table' ? 'active' : ''}`} title="Table"><List size={15} /></button>
        {!filter && <button onClick={() => navigate('/stubs/create')} style={btn('primary', 'sm')}><Plus size={13} /> New</button>}
      </div>
      <SlideOver open={importOpen} onClose={() => setImportOpen(false)} title="Import stubs" width="560px">
        <ImportStubs onDone={() => setImportOpen(false)} />
      </SlideOver>
    </div>
  );
}

function SearchRow({ searchText, setSearchText, showFilters, setShowFilters, hasFilters, onSmartSearch, smartSearching }: {
  searchText: string; setSearchText: (v: string) => void;
  showFilters: boolean; setShowFilters: (v: boolean) => void; hasFilters: boolean;
  onSmartSearch?: (v: string) => void; smartSearching?: boolean;
}) {
  return (
    <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
      <div style={{ position: 'relative', flex: 1 }}>
        <Search size={13} style={{ position: 'absolute', left: 8, top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
        <input value={searchText} onChange={(e) => (onSmartSearch ? onSmartSearch(e.target.value) : setSearchText(e.target.value))}
          placeholder="Search by service, method, ID, or payload content..."
          style={{ width: '100%', padding: '7px 8px 7px 30px', fontSize: 12, borderRadius: 6, border: '1px solid var(--border)', background: 'var(--bg-primary)', color: 'var(--text-primary)', outline: 'none' }} />
        {smartSearching && <span style={{ position: 'absolute', right: 32, top: '50%', transform: 'translateY(-50%)', fontSize: 11, color: 'var(--text-muted)' }}>searching...</span>}
      </div>
      <button onClick={() => setShowFilters(!showFilters)}
        style={{ ...iconBtn, color: showFilters || hasFilters ? colors.accent : 'var(--text-muted)', background: hasFilters ? `${colors.accent}10` : 'transparent' }}>
        <Filter size={14} />
      </button>
    </div>
  );
}

function MatcherChips({ activeMatchers, toggleMatcher, setActiveMatchers }: { activeMatchers: Set<string>; toggleMatcher: (t: string) => void; setActiveMatchers: React.Dispatch<React.SetStateAction<Set<string>>> }) {
  return (
    <div style={{ display: 'flex', gap: 3, flexWrap: 'wrap' }}>
      {MATCHER_TYPES.map((t) => {
        const active = activeMatchers.has(t);
        return (
          <button key={t} onClick={() => toggleMatcher(t)}
            style={{ padding: '2px 8px', fontSize: 11, borderRadius: 4, cursor: 'pointer', border: 'none', outline: 'none',
              background: active ? MATCHER_COLORS[t] : `${MATCHER_COLORS[t]}18`, color: active ? '#fff' : MATCHER_COLORS[t], fontWeight: active ? 600 : 400 }}>
            {t}
          </button>
        );
      })}
      {activeMatchers.size > 0 && (
        <button onClick={() => setActiveMatchers(new Set<string>())} style={{ fontSize: 11, padding: '2px 6px', borderRadius: 4, border: 'none', background: 'transparent', color: colors.error, cursor: 'pointer' }}>Clear</button>
      )}
    </div>
  );
}

const SOURCES = ['', 'file', 'rest', 'mcp', 'proxy'];

function FilterRow({ svcFilter, setSvcFilter, methodFilter, setMethodFilter, srcFilter, setSrcFilter, services, methodsByService }: {
  svcFilter: string; setSvcFilter: (v: string) => void;
  methodFilter: string; setMethodFilter: (v: string) => void;
  srcFilter: string; setSrcFilter: (v: string) => void;
  services: string[]; methodsByService: Record<string, Set<string>>;
}) {
  // Methods scoped to the selected service; else the union across all services.
  const byName = (a: string, b: string) => a.localeCompare(b);
  const methods = svcFilter
    ? [...(methodsByService[svcFilter] ?? [])].sort(byName)
    : [...new Set(Object.values(methodsByService).flatMap((s) => [...s]))].sort(byName);
  const field: React.CSSProperties = { display: 'flex', flexDirection: 'column', gap: 3, minWidth: 180 };
  return (
    <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', padding: 12, borderRadius: 'var(--radius-lg)', border: '1px solid var(--border)', background: 'var(--bg-secondary)' }}>
      <div style={field}>
        <span className="field-label">Service</span>
        <select value={svcFilter} onChange={(e) => { setSvcFilter(e.target.value); setMethodFilter(''); }} className="input" style={{ fontSize: 12.5 }}>
          <option value="">All services</option>
          {services.map((s) => <option key={s} value={s}>{s}</option>)}
        </select>
      </div>
      <div style={field}>
        <span className="field-label">Method{svcFilter ? '' : ' (all services)'}</span>
        <select value={methodFilter} onChange={(e) => setMethodFilter(e.target.value)} className="input" style={{ fontSize: 12.5 }}>
          <option value="">All methods</option>
          {methods.map((m) => <option key={m} value={m}>{m}</option>)}
        </select>
      </div>
      <div style={field}>
        <span className="field-label">Source</span>
        <div className="tabs" style={{ alignSelf: 'flex-start' }}>
          {SOURCES.map((s) => (
            <button key={s} className={`tab ${srcFilter === s ? 'active' : ''}`} onClick={() => setSrcFilter(s)}>{s || 'all'}</button>
          ))}
        </div>
      </div>
    </div>
  );
}

function FilterBadges({ svcFilter, methodFilter, activeMatchers, clearAll, toggleMatcher, onRemoveSvc, onRemoveMethod }: {
  svcFilter: string; methodFilter: string; activeMatchers: Set<string>;
  clearAll: () => void; toggleMatcher: (t: string) => void;
  onRemoveSvc: () => void; onRemoveMethod: () => void;
}) {
  const chips: { label: string; onRemove: () => void }[] = [];
  if (svcFilter) chips.push({ label: `service: ${svcFilter}`, onRemove: onRemoveSvc });
  if (methodFilter) chips.push({ label: `method: ${methodFilter}`, onRemove: onRemoveMethod });
  [...activeMatchers].forEach((t) => chips.push({ label: t, onRemove: () => toggleMatcher(t) }));
  if (chips.length === 0) return null;
  return (
    <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', alignItems: 'center' }}>
      {chips.map((c, i) => (
        <span key={i} style={{ display: 'inline-flex', alignItems: 'center', gap: 3, fontSize: 11, padding: '2px 6px', borderRadius: 4, background: `${colors.accent}12`, color: colors.accent, fontWeight: 500 }}>
          {c.label} <X size={10} style={{ cursor: 'pointer' }} onClick={c.onRemove} />
        </span>
      ))}
      <button onClick={clearAll} style={{ fontSize: 11, padding: '2px 6px', borderRadius: 4, border: 'none', background: 'transparent', color: colors.error, cursor: 'pointer' }}>Clear all</button>
    </div>
  );
}

function StreamTag({ methodType }: { methodType?: string }) {
  const k = streamKind(methodType);
  return <span className="badge" style={{ background: `${k.color}1e`, color: k.color }} title={k.full}>{k.label}</span>;
}

function testLink(stub: Stub): string {
  const ex = stubRequestExample(stub);
  return `/stubs/test?service=${encodeURIComponent(stub.service)}&method=${encodeURIComponent(stub.method)}&id=${encodeURIComponent(stub.id)}&payload=${encodeURIComponent(ex.payload)}&headers=${encodeURIComponent(ex.headers)}`;
}

function StubCard({ stub, onDelete, expandedId, setExpandedId, navigate, methodType }: { stub: Stub; onDelete: (s: Stub) => void; expandedId: string | null; setExpandedId: (id: string | null) => void; navigate: (p: string) => void; methodType?: string }) {
  const expanded = expandedId === stub.id;
  const types = matcherTypes(stub);
  const color = MATCHER_COLORS[types[0] || 'any'] || '#64748b';
  const out = outputKind(stub);
  const hasHeaders = hasContent(stub.headers);
  const reqMsgs = requestMessages(stub);
  const resMsgs = responseMessages(stub);
  const firstMatcher = reqMsgs[0] ? (matcherEntries(reqMsgs[0])[0]?.value ?? reqMsgs[0]) : undefined;
  const inPreview = reqMsgs.length === 0 ? 'any request' : compactPreview(firstMatcher, 42);
  const inTitle = reqMsgs.length ? (prettyJson(firstMatcher) || undefined) : undefined;
  return (
    <div style={{ borderRadius: 8, border: `1px solid ${expanded ? color : 'var(--border)'}`, borderLeft: `3px solid ${color}`, background: expanded ? `${color}05` : 'var(--bg-secondary)', overflow: 'hidden' }}>
      <div onClick={() => setExpandedId(expanded ? null : stub.id)} style={{ padding: '8px 10px', cursor: 'pointer' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <code style={{ fontSize: 11, color: 'var(--text-muted)' }} title={stub.id}>{stub.id.slice(0, 8)}</code>
          <span style={{ fontSize: 11, padding: '1px 5px', borderRadius: 3, background: '#64748b18', color: '#64748b', fontWeight: 500 }}>{stub.source || '—'}</span>
        </div>
        <div style={{ fontSize: 12, fontWeight: 600, marginTop: 2, display: 'flex', alignItems: 'center', gap: 5 }}>
          <StreamTag methodType={methodType} />
          <span>{stub.service}/{stub.method}</span>
        </div>
        <div style={{ display: 'flex', gap: 8, fontSize: 11, color: 'var(--text-muted)', marginTop: 4, flexWrap: 'wrap' }}>
          <span>Prio <strong style={{ color: 'var(--text-secondary)' }}>{stub.priority}</strong></span>
          <span>Times <strong style={{ color: 'var(--text-secondary)' }}>{stub.options?.times ?? '∞'}</strong></span>
          {reqMsgs.length > 1 && <span style={{ color: 'var(--accent-text)' }}>{reqMsgs.length} req msgs</span>}
          {resMsgs.length > 1 && <span style={{ color: '#06b6d4' }}>{resMsgs.length} resp msgs</span>}
        </div>
        <div style={{ fontSize: 11, color: 'var(--text-secondary)', fontFamily: 'monospace', marginTop: 3, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={inTitle}>
          <span style={{ color: 'var(--text-muted)' }}>{reqMsgs.length > 1 ? 'req[0] ' : 'in '}</span>{inPreview}
        </div>
        <div style={{ display: 'flex', gap: 4, marginTop: 5, alignItems: 'center', flexWrap: 'wrap' }}>
          <span style={{ fontSize: 11, padding: '1px 6px', borderRadius: 4, fontWeight: 500, background: `${out.color}18`, color: out.color }}>{out.label}</span>
          {types.map((t) => <span key={t} style={{ fontSize: 11, padding: '1px 5px', borderRadius: 3, fontWeight: 600, background: `${MATCHER_COLORS[t] || '#64748b'}18`, color: MATCHER_COLORS[t] || '#64748b' }}>{t}</span>)}
          {hasHeaders && <span style={{ fontSize: 11, padding: '1px 5px', borderRadius: 3, fontWeight: 600, background: '#06b6d418', color: '#06b6d4' }}>headers</span>}
        </div>
      </div>
      {expanded && <div style={{ borderTop: '1px solid var(--border)', padding: 10 }}><ExpandedContent stub={stub} navigate={navigate} onDelete={onDelete} methodType={methodType} /></div>}
    </div>
  );
}

// Wraps rather than truncates so the whole id stays readable.
function CopyableId({ id }: { id: string }) {
  const { copied, copy } = useCopy();
  return (
    <code onClick={() => copy(id)} title="Click to copy"
      style={{ display: 'inline-flex', alignItems: 'center', gap: 6, cursor: 'pointer', fontFamily: 'var(--mono)', fontSize: 11.5, color: 'var(--text-primary)', background: 'var(--bg-tertiary)', padding: '3px 8px', borderRadius: 4, userSelect: 'all', wordBreak: 'break-all', lineHeight: 1.4 }}>
      {id}
      {copied
        ? <Check size={12} style={{ color: colors.success, flexShrink: 0 }} />
        : <Copy size={11} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />}
    </code>
  );
}

function ExpandedContent({ stub, navigate, onDelete, methodType }: { stub: Stub; navigate: (p: string) => void; onDelete: (s: Stub) => void; methodType?: string }) {
  const headersJson = prettyJson(stub.headers);
  const k = streamKind(methodType);
  const reqMsgs = requestMessages(stub);
  const resMsgs = responseMessages(stub);
  const reqStream = reqMsgs.length > 1;
  const resStream = (stub.output?.stream?.length ?? 0) > 0;
  const isErr = !!stub.output?.error || (stub.output?.code ?? 0) > 0;
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8, fontSize: 12 }}>
      <div style={{ display: 'flex', gap: 8, alignItems: 'center', color: 'var(--text-muted)' }}>
        <span style={{ flexShrink: 0 }}>ID:</span> <CopyableId id={stub.id} />
      </div>
      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', color: 'var(--text-muted)', alignItems: 'center' }}>
        <span>Type: <strong style={{ color: k.color }}>{k.full}</strong></span>
        <span>Priority: <strong style={{ color: 'var(--text-primary)' }}>{stub.priority}</strong></span>
        <span>Times: <strong style={{ color: 'var(--text-primary)' }}>{stub.options?.times ?? '∞'}</strong></span>
      </div>
      {headersJson && <><div className="section-title" style={{ marginBottom: 2 }}>Headers matcher</div><pre className="json-block">{headersJson}</pre></>}
      <div className="section-title" style={{ marginBottom: 2 }}>{reqStream ? `Request stream · ${reqMsgs.length} msgs` : 'Request matcher'}</div>
      {reqMsgs.length === 0
        ? <div style={{ fontSize: 11, color: 'var(--text-muted)', fontStyle: 'italic' }}>any request</div>
        : reqMsgs.map((m, i) => <pre key={i} className="json-block">{reqStream ? `# ${i + 1}\n` : ''}{prettyJson(m) || '{}'}</pre>)}
      <div className="section-title" style={{ marginBottom: 2 }}>{isErr ? 'Error response' : resStream ? `Response stream · ${resMsgs.length} msgs` : 'Response'}</div>
      {isErr
        ? <pre className="json-block" style={{ color: colors.error }}>{`code ${stub.output?.code ?? 0}${stub.output?.error ? '\n' + stub.output.error : ''}`}</pre>
        : resMsgs.length === 0
          ? <div style={{ fontSize: 11, color: 'var(--text-muted)', fontStyle: 'italic' }}>empty</div>
          : resMsgs.map((r, i) => <pre key={i} className="json-block">{resStream ? `# ${i + 1}\n` : ''}{prettyJson(r) || JSON.stringify(r, null, 2)}</pre>)}
      <div style={{ display: 'flex', gap: 6 }}>
        <button onClick={() => navigate(`/stubs/${stub.id}`)} style={smallBtn}><Edit3 size={11} /> View</button>
        <button onClick={() => { stashClone(stub); navigate('/stubs/create?clone=1'); }} style={smallBtn}><Copy size={11} /> Clone</button>
        <button onClick={() => navigate(testLink(stub))} style={smallBtn}><Play size={11} /> Test</button>
        {stub.source !== 'file' && <button onClick={() => onDelete(stub)} style={{ ...smallBtn, color: colors.error, marginLeft: 'auto' }}>Delete</button>}
      </div>
    </div>
  );
}
