// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MessageSequenceEditor, type SequenceItem } from './MessageSequenceEditor';

// Monaco is heavy and DOM-hostile in jsdom; stub it with a textarea.
vi.mock('../../components/json/MonacoEditor', () => ({
  MonacoEditor: ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
    <textarea data-testid="monaco" value={value} onChange={(e) => onChange(e.target.value)} />
  ),
}));

afterEach(cleanup);

const items: SequenceItem[] = [
  { type: 'equals', value: '{"a":1}', ignoreArrayOrder: false },
  { type: 'contains', value: '{"b":2}', ignoreArrayOrder: false },
];

describe('MessageSequenceEditor', () => {
  it('streaming: move down swaps ordered messages', async () => {
    const onChange = vi.fn();
    render(<MessageSequenceEditor items={items} onChange={onChange} streaming />);
    await userEvent.click(screen.getAllByLabelText('Move message down')[0]);
    expect(onChange).toHaveBeenCalledWith([items[1], items[0]]);
  });

  it('streaming: first item cannot move up', () => {
    render(<MessageSequenceEditor items={items} onChange={vi.fn()} streaming />);
    expect(screen.getAllByLabelText('Move message up')[0]).toBeDisabled();
  });

  it('non-streaming: no reorder controls, shows alternative wording', () => {
    render(<MessageSequenceEditor items={items} onChange={vi.fn()} streaming={false} />);
    expect(screen.queryByLabelText('Move message down')).toBeNull();
    expect(screen.getByText(/Add alternative/)).toBeInTheDocument();
  });

  it('remove drops the item', async () => {
    const onChange = vi.fn();
    render(<MessageSequenceEditor items={items} onChange={onChange} streaming />);
    await userEvent.click(screen.getAllByLabelText('Remove message')[0]);
    expect(onChange).toHaveBeenCalledWith([items[1]]);
  });

  it('changing matcher kind updates that item only', async () => {
    const onChange = vi.fn();
    render(<MessageSequenceEditor items={items} onChange={onChange} streaming />);
    await userEvent.selectOptions(screen.getAllByLabelText('Matcher kind')[0], 'glob');
    expect(onChange).toHaveBeenCalledWith([{ ...items[0], type: 'glob' }, items[1]]);
  });
});
