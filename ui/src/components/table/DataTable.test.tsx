// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { Column } from './tableFeatures';
import { DataTable } from './DataTable';

afterEach(cleanup);

interface Row { id: string; name: string }
const columns: Column<Row>[] = [{ id: 'name', header: 'Name', accessorKey: 'name' }];
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

describe('DataTable expansion', () => {
  const renderExpanded = (r: Row) => <div>expanded-{r.name}</div>;

  it('collapses the expanded row when its id leaves the data', async () => {
    const data = [{ id: 'a', name: 'row-a' }, { id: 'b', name: 'row-b' }];
    const { rerender } = render(
      <DataTable data={data} columns={columns} getRowId={(r) => r.id} renderExpanded={renderExpanded} />,
    );

    await userEvent.click(screen.getByText('row-a'));
    expect(screen.getByText('expanded-row-a')).toBeTruthy();

    // Data reloads without row 'a' — the expansion must not stick to the row
    // that now occupies a's slot (the index-keyed bug), it must collapse.
    rerender(
      <DataTable
        data={[{ id: 'b', name: 'row-b' }, { id: 'c', name: 'row-c' }]}
        columns={columns} getRowId={(r) => r.id} renderExpanded={renderExpanded} />,
    );

    expect(screen.queryByText('expanded-row-a')).toBeNull();
    expect(screen.queryByText('expanded-row-b')).toBeNull();
    expect(screen.queryByText('expanded-row-c')).toBeNull();
  });

  it('keeps the expansion on the same record after data reorders', async () => {
    const renderData = (d: Row[]) => (
      <DataTable data={d} columns={columns} getRowId={(r) => r.id} renderExpanded={renderExpanded} />
    );
    const { rerender } = render(renderData([{ id: 'a', name: 'row-a' }, { id: 'b', name: 'row-b' }]));

    await userEvent.click(screen.getByText('row-b'));
    expect(screen.getByText('expanded-row-b')).toBeTruthy();

    // 'b' shifts position (new record prepended); expansion follows the record.
    rerender(renderData([{ id: 'c', name: 'row-c' }, { id: 'a', name: 'row-a' }, { id: 'b', name: 'row-b' }]));
    expect(screen.getByText('expanded-row-b')).toBeTruthy();
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
