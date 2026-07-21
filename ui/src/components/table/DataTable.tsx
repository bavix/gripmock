import { useState, useCallback, Fragment, type ReactNode, type CSSProperties } from 'react';
import {
  useReactTable, getCoreRowModel, getSortedRowModel, getPaginationRowModel,
  getFilteredRowModel, flexRender, type ColumnDef, type SortingState,
  type VisibilityState, type Row, type Table, type SortDirection,
} from '@tanstack/react-table';
import { ChevronDown, ChevronUp, ChevronsUpDown, ChevronRight } from 'lucide-react';

type Density = 'compact' | 'normal' | 'comfortable';
type Sorted = false | SortDirection;

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

function ariaSortValue(canSort: boolean, sorted: Sorted): 'ascending' | 'descending' | 'none' | undefined {
  if (!canSort) return undefined;
  if (sorted === 'asc') return 'ascending';
  if (sorted === 'desc') return 'descending';
  return 'none';
}

function SortIcon({ sorted }: Readonly<{ sorted: Sorted }>) {
  if (sorted === 'asc') return <ChevronUp size={12} />;
  if (sorted === 'desc') return <ChevronDown size={12} />;
  return <ChevronsUpDown size={12} style={{ opacity: 0.3 }} />;
}

function ExpandChevron({ expanded }: Readonly<{ expanded: boolean }>) {
  return (
    <button type="button" style={{ border: 'none', background: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: 0, display: 'flex' }} tabIndex={-1}>
      {expanded ? <ChevronDown size={13} /> : <ChevronRight size={13} />}
    </button>
  );
}

function TableHead<T>({ table }: Readonly<{ table: Table<T> }>) {
  return (
    <thead>
      {table.getHeaderGroups().map((hg) => (
        <tr key={hg.id}>
          {hg.headers.map((h) => {
            const canSort = h.column.getCanSort();
            const sorted = h.column.getIsSorted();
            return (
              <th key={h.id} onClick={h.column.getToggleSortingHandler()}
                aria-sort={ariaSortValue(canSort, sorted)}
                style={{
                  textAlign: 'left', padding: '8px 10px', fontWeight: 600, fontSize: 11,
                  textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--text-muted)',
                  borderBottom: '1px solid var(--border)', cursor: canSort ? 'pointer' : 'default',
                  whiteSpace: 'nowrap', position: 'sticky', top: 0, background: 'var(--bg-secondary)', zIndex: 1, width: h.getSize(),
                }}>
                <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                  {flexRender(h.column.columnDef.header, h.getContext())}
                  {canSort && <SortIcon sorted={sorted} />}
                </span>
              </th>
            );
          })}
        </tr>
      ))}
    </thead>
  );
}

interface RowProps<T> {
  row: Row<T>;
  density: Density;
  canExpand: boolean;
  expandedId: string | null;
  setExpandedId: (id: string | null) => void;
  onRowClick?: (row: T) => void;
  renderExpanded?: (row: T) => ReactNode;
  colCount: number;
}

function TableRow<T>({ row, density, canExpand, expandedId, setExpandedId, onRowClick, renderExpanded, colCount }: Readonly<RowProps<T>>) {
  const isExpanded = expandedId === row.id;
  const clickable = canExpand || !!onRowClick;
  const handleRowClick = () => {
    if (canExpand) setExpandedId(isExpanded ? null : row.id);
    onRowClick?.(row.original);
  };
  return (
    <Fragment>
      <tr className={clickable ? 'hover-row' : undefined}
        style={{ borderBottom: isExpanded ? 'none' : '1px solid var(--border)', cursor: clickable ? 'pointer' : undefined, background: isExpanded ? 'var(--bg-secondary)' : undefined }}
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
          <td colSpan={colCount} style={{ padding: 0, borderBottom: '1px solid var(--border)' }}>
            <div style={{ padding: '10px 10px 10px 38px' }}>
              {renderExpanded(row.original)}
            </div>
          </td>
        </tr>
      )}
    </Fragment>
  );
}

interface BodyProps<T> {
  loading?: boolean;
  rows: Row<T>[];
  colCount: number;
  emptyMessage: string;
  density: Density;
  canExpand: boolean;
  expandedId: string | null;
  setExpandedId: (id: string | null) => void;
  onRowClick?: (row: T) => void;
  renderExpanded?: (row: T) => ReactNode;
}

function messageRow(colCount: number, message: ReactNode) {
  return (
    <tbody>
      <tr><td colSpan={colCount} style={{ padding: 40, textAlign: 'center', color: 'var(--text-muted)' }}>{message}</td></tr>
    </tbody>
  );
}

function DataTableBody<T>({ loading, rows, colCount, emptyMessage, density, canExpand, expandedId, setExpandedId, onRowClick, renderExpanded }: Readonly<BodyProps<T>>) {
  if (loading) return messageRow(colCount, 'Loading...');
  if (rows.length === 0) return messageRow(colCount, emptyMessage);
  return (
    <tbody>
      {rows.map((row) => (
        <TableRow key={row.id} row={row} density={density} canExpand={canExpand}
          expandedId={expandedId} setExpandedId={setExpandedId} onRowClick={onRowClick}
          renderExpanded={renderExpanded} colCount={colCount} />
      ))}
    </tbody>
  );
}

export function DataTable<T = any>({
  data, columns, loading, emptyMessage = 'No data', getRowId, renderExpanded,
  density: extDensity, onDensityChange, onRowClick,
  manualPagination, rowCount, pageIndex = 0, pageSize = 50, onPageChange,
}: Readonly<DataTableProps<T>>) {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [internalVisibility, setInternalVisibility] = useState<VisibilityState>({});
  const [density, setDensity] = useState<Density>(extDensity || 'normal');
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const expandCol: ColumnDef<T> | undefined = renderExpanded ? {
    id: '_exp',
    header: '',
    cell: ({ row }) => <ExpandChevron expanded={expandedId === row.id} />,
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
          <TableHead table={table} />
          <DataTableBody loading={loading} rows={rows} colCount={allCols.length}
            emptyMessage={emptyMessage} density={density} canExpand={!!renderExpanded}
            expandedId={expandedId} setExpandedId={setExpandedId} onRowClick={onRowClick}
            renderExpanded={renderExpanded} />
        </table>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '4px 0' }}>
        <button type="button" onClick={handleDensity} style={{ fontSize: 11, padding: '4px 8px', borderRadius: 4, border: '1px solid var(--border)', background: 'var(--bg-secondary)', color: 'var(--text-secondary)', cursor: 'pointer' }}>
          {density}
        </button>
        <div style={{ display: 'flex', gap: 8, fontSize: 12, color: 'var(--text-muted)' }}>
          <button type="button" onClick={() => table.setPageIndex(0)} disabled={!table.getCanPreviousPage()} style={pageBtn}>««</button>
          <button type="button" onClick={() => table.previousPage()} disabled={!table.getCanPreviousPage()} style={pageBtn}>«</button>
          <span>Page {table.getState().pagination.pageIndex + 1} of {table.getPageCount()}</span>
          <button type="button" onClick={() => table.nextPage()} disabled={!table.getCanNextPage()} style={pageBtn}>»</button>
          <button type="button" onClick={() => table.setPageIndex(table.getPageCount() - 1)} disabled={!table.getCanNextPage()} style={pageBtn}>»»</button>
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
