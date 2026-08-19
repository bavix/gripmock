import {
  coreFeatures,
  rowSortingFeature,
  rowPaginationFeature,
  columnVisibilityFeature,
  columnSizingFeature,
  createCoreRowModel,
  createSortedRowModel,
  createPaginatedRowModel,
  type ColumnDef,
  type Row,
  type Table,
  type RowData,
} from '@tanstack/react-table';

export const tableFeatures = {
  ...coreFeatures,
  rowSortingFeature,
  rowPaginationFeature,
  columnVisibilityFeature,
  columnSizingFeature,
  coreRowModel: createCoreRowModel(),
  sortedRowModel: createSortedRowModel(),
  paginatedRowModel: createPaginatedRowModel(),
} as const;

export type TableFeatureSet = typeof tableFeatures;

export type Column<T extends RowData> = ColumnDef<TableFeatureSet, T>;
export type RowOf<T extends RowData> = Row<TableFeatureSet, T>;
export type TableOf<T extends RowData> = Table<TableFeatureSet, T>;
