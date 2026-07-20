// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useFocusTrap } from './useFocusTrap';

afterEach(cleanup);

function Modal({ onClose }: { onClose: () => void }) {
  const ref = useFocusTrap<HTMLDivElement>(true, onClose);
  return (
    <div ref={ref} role="dialog" tabIndex={-1}>
      <button>first</button>
      <button>second</button>
    </div>
  );
}

describe('useFocusTrap', () => {
  it('focuses the first focusable element on activate', () => {
    render(<Modal onClose={vi.fn()} />);
    expect(screen.getByText('first')).toHaveFocus();
  });

  it('Escape invokes onClose', async () => {
    const onClose = vi.fn();
    render(<Modal onClose={onClose} />);
    await userEvent.keyboard('{Escape}');
    expect(onClose).toHaveBeenCalled();
  });

  it('Tab past the last element wraps to the first', async () => {
    render(<Modal onClose={vi.fn()} />);
    screen.getByText('second').focus();
    await userEvent.tab();
    expect(screen.getByText('first')).toHaveFocus();
  });

  it('Shift+Tab from the first element wraps to the last', async () => {
    render(<Modal onClose={vi.fn()} />);
    screen.getByText('first').focus();
    await userEvent.tab({ shift: true });
    expect(screen.getByText('second')).toHaveFocus();
  });
});
