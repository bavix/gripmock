import { useState, useCallback, Fragment, type ReactNode, type CSSProperties } from 'react';
import {
  useReactTable, getCoreRowModel, getSortedRowModel, getPaginationRowModel,
  getFilteredRowModel, flexRender, type ColumnDef, type SortingState,
  type VisibilityState,
} from '@tanstack/react-table';
import { ChevronDown, ChevronUp, ChevronsUpDown, ChevronRight } from 'lucide-react';

type Density = 'compact' | 'normal' | 'comfortable';

const DENSITY_STYLES: Record<Density, CSSProperties> = {
  compact: { height: 32, fontSize: 12, padding: '0 8px' },
  normal: { height: 40, fontSize: 13, padding: '0 10px' },
  comfortable: { height: 48, fontSize: 13, padding: '0 12px' },
};

interface DataTableProps<T> {
  data: T[];
  columns: ColumnDef<T>[];
  loading?: boolean;
  emptyMessage?: string;
  getRowId?: (row: T) => string;
  renderExpanded?: (row: T) => ReactNode;
  density?: Density;
  onDensityChange?: (d: Density) => void;
  onRowClick?: (row: T) => void;
  /** Server-side pagination: total row count + controlled page state. */
  manualPagination?: boolean;
  rowCount?: number;
  pageIndex?: number;
  pageSize?: number;
  onPageChange?: (pageIndex: number) => void;
}

export function DataTable<T = any>({
  data, columns, loading, emptyMessage = 'No data', getRowId, renderExpanded,
  density: extDensity, onDensityChange, onRowClick,
  manualPagination, rowCount, pageIndex = 0, pageSize = 50, onPageChange,
}: DataTableProps<T>) {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [internalVisibility, setInternalVisibility] = useState<VisibilityState>({});
  const [density, setDensity] = useState<Density>(extDensity || 'normal');
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const expandCol: ColumnDef<T> | undefined = renderExpanded ? {
    id: '_exp',
    header: '',
    cell: ({ row }) => (
      <button style={{ border: 'none', background: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: 0, display: 'flex' }} tabIndex={-1}>
        {expandedId === row.id ? <ChevronDown size={13} /> : <ChevronRight size={13} />}
      </button>
    ),
    size: 28,
  } : undefined;

  const allCols = expandCol ? [expandCol, ...columns] : columns;

  // Stable per-row id: use the provided getRowId, else the row index so rows
  // without an `id` field (e.g. history call records) never collide.
  const table = useReactTable({
    data, columns: allCols,
    getRowId: getRowId ? ((row) => getRowId(row)) : ((_row, index) => String(index)),
    state: {
      sorting, columnVisibility: internalVisibility,
      ...(manualPagination ? { pagination: { pageIndex, pageSize } } : {}),
    },
    onSortingChange: setSorting,
    onColumnVisibilityChange: setInternalVisibility,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    ...(manualPagination
      ? {
          manualPagination: true,
          rowCount: rowCount ?? 0,
          onPaginationChange: (updater) => {
            const next = typeof updater === 'function' ? updater({ pageIndex, pageSize }) : updater;
            onPageChange?.(next.pageIndex);
          },
        }
      : { initialState: { pagination: { pageSize: 50 } } }),
  });

  const handleDensity = useCallback(() => {
    const m: Record<string, Density> = { compact: 'normal', normal: 'comfortable', comfortable: 'compact' };
    const d = m[density]; setDensity(d); onDensityChange?.(d);
  }, [density, onDensityChange]);

  const rows = table.getRowModel().rows;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
          <thead>
            {table.getHeaderGroups().map((hg) => (
              <tr key={hg.id}>
                {hg.headers.map((h) => (
                  <th key={h.id} onClick={h.column.getToggleSortingHandler()}
                    aria-sort={h.column.getCanSort()
                      ? (h.column.getIsSorted() === 'asc' ? 'ascending' : h.column.getIsSorted() === 'desc' ? 'descending' : 'none')
                      : undefined}
                    style={{
                      textAlign: 'left', padding: '8px 10px', fontWeight: 600, fontSize: 11,
                      textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--text-muted)',
                      borderBottom: '1px solid var(--border)', cursor: h.column.getCanSort() ? 'pointer' : 'default',
                      whiteSpace: 'nowrap', position: 'sticky', top: 0, background: 'var(--bg-secondary)', zIndex: 1, width: h.getSize(),
                    }}>
                    <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                      {flexRender(h.column.columnDef.header, h.getContext())}
                      {h.column.getCanSort() && (
                        h.column.getIsSorted() === 'asc' ? <ChevronUp size={12} /> :
                        h.column.getIsSorted() === 'desc' ? <ChevronDown size={12} /> :
                        <ChevronsUpDown size={12} style={{ opacity: 0.3 }} />
                      )}
                    </span>
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={allCols.length} style={{ padding: 40, textAlign: 'center', color: 'var(--text-muted)' }}>Loading...</td></tr>
            ) : rows.length === 0 ? (
              <tr><td colSpan={allCols.length} style={{ padding: 40, textAlign: 'center', color: 'var(--text-muted)' }}>{emptyMessage}</td></tr>
            ) : rows.map((row) => {
              const isExpanded = expandedId === row.id;
              const canExpand = !!renderExpanded;
              const handleRowClick = () => {
                if (canExpand) setExpandedId(isExpanded ? null : row.id);
                onRowClick?.(row.original);
              };
              return (
                <Fragment key={row.id}>
                  <tr className={canExpand || onRowClick ? 'hover-row' : undefined}
                    style={{ borderBottom: isExpanded ? 'none' : '1px solid var(--border)', cursor: (canExpand || onRowClick) ? 'pointer' : undefined, background: isExpanded ? 'var(--bg-secondary)' : undefined }}
                    onClick={handleRowClick}>
                    {row.getVisibleCells().map((cell) => (
                      <td key={cell.id} onClick={cell.column.id === '_sel' ? (e) => e.stopPropagation() : undefined}
                        style={{ ...DENSITY_STYLES[density], whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    ))}
                  </tr>
                  {isExpanded && renderExpanded && (
                    <tr style={{ background: 'var(--bg-secondary)' }}>
                      <td colSpan={allCols.length} style={{ padding: 0, borderBottom: '1px solid var(--border)' }}>
                        <div style={{ padding: '10px 10px 10px 38px' }}>
                          {renderExpanded(row.original)}
                        </div>
                      </td>
                    </tr>
                  )}
                </Fragment>
              );
            })}
          </tbody>
        </table>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '4px 0' }}>
        <button onClick={handleDensity} style={{ fontSize: 11, padding: '4px 8px', borderRadius: 4, border: '1px solid var(--border)', background: 'var(--bg-secondary)', color: 'var(--text-secondary)', cursor: 'pointer' }}>
          {density}
        </button>
        <div style={{ display: 'flex', gap: 8, fontSize: 12, color: 'var(--text-muted)' }}>
          <button onClick={() => table.setPageIndex(0)} disabled={!table.getCanPreviousPage()} style={pageBtn}>««</button>
          <button onClick={() => table.previousPage()} disabled={!table.getCanPreviousPage()} style={pageBtn}>«</button>
          <span>Page {table.getState().pagination.pageIndex + 1} of {table.getPageCount()}</span>
          <button onClick={() => table.nextPage()} disabled={!table.getCanNextPage()} style={pageBtn}>»</button>
          <button onClick={() => table.setPageIndex(table.getPageCount() - 1)} disabled={!table.getCanNextPage()} style={pageBtn}>»»</button>
        </div>
      </div>
    </div>
  );
}

const pageBtn: CSSProperties = {
  padding: '2px 8px', fontSize: 12, borderRadius: 4,
  border: '1px solid var(--border)', background: 'var(--bg-secondary)',
  color: 'var(--text-secondary)', cursor: 'pointer',
};
