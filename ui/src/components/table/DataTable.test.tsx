// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ColumnDef } from '@tanstack/react-table';
import { DataTable } from './DataTable';

afterEach(cleanup);

interface Row { id: string; name: string }
const columns: ColumnDef<Row>[] = [{ id: 'name', header: 'Name', accessorKey: 'name' }];
const rows = (n: number): Row[] => Array.from({ length: n }, (_, i) => ({ id: String(i), name: `row-${i}` }));

describe('DataTable manual pagination', () => {
  it('derives page count from rowCount, not loaded rows', () => {
    render(<DataTable data={rows(50)} columns={columns} manualPagination rowCount={120} pageIndex={0} pageSize={50} onPageChange={vi.fn()} />);
    expect(screen.getByText(/Page 1 of 3/)).toBeInTheDocument();
  });

  it('next button reports the target page index', async () => {
    const onPageChange = vi.fn();
    render(<DataTable data={rows(50)} columns={columns} manualPagination rowCount={120} pageIndex={0} pageSize={50} onPageChange={onPageChange} />);
    await userEvent.click(screen.getByText('»'));
    expect(onPageChange).toHaveBeenCalledWith(1);
  });

  it('prev is disabled on the first page', () => {
    render(<DataTable data={rows(50)} columns={columns} manualPagination rowCount={120} pageIndex={0} pageSize={50} onPageChange={vi.fn()} />);
    expect(screen.getByText('«')).toBeDisabled();
  });

  it('next is disabled on the last page', () => {
    render(<DataTable data={rows(20)} columns={columns} manualPagination rowCount={120} pageIndex={2} pageSize={50} onPageChange={vi.fn()} />);
    expect(screen.getByText('»')).toBeDisabled();
  });
});

describe('DataTable a11y', () => {
  it('sortable headers expose aria-sort and toggle it on click', async () => {
    render(<DataTable data={rows(3)} columns={columns} />);
    const th = screen.getByRole('columnheader', { name: /Name/ });
    expect(th).toHaveAttribute('aria-sort', 'none');
    await userEvent.click(th);
    expect(th).toHaveAttribute('aria-sort', 'ascending');
    await userEvent.click(th);
    expect(th).toHaveAttribute('aria-sort', 'descending');
  });
});
