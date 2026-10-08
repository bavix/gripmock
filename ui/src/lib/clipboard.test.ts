// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { copyText } from './clipboard';

function setClipboard(clipboard: unknown) {
  Object.defineProperty(navigator, 'clipboard', { value: clipboard, configurable: true });
}

function setExecCommand(impl: ((command: string) => boolean) | undefined) {
  Object.defineProperty(document, 'execCommand', { value: impl, configurable: true });
}

beforeEach(() => {
  setClipboard(undefined);
  setExecCommand(undefined);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('copyText', () => {
  it('uses the async clipboard API when the origin is secure', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    setClipboard({ writeText });

    await expect(copyText('hello')).resolves.toBe(true);
    expect(writeText).toHaveBeenCalledWith('hello');
  });

  it('falls back to execCommand when navigator.clipboard is missing', async () => {
    const execCommand = vi.fn().mockReturnValue(true);
    setExecCommand(execCommand);

    await expect(copyText('hello')).resolves.toBe(true);
    expect(execCommand).toHaveBeenCalledWith('copy');
  });

  it('falls back when the clipboard write is rejected', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('NotAllowedError'));
    const execCommand = vi.fn().mockReturnValue(true);
    setClipboard({ writeText });
    setExecCommand(execCommand);

    await expect(copyText('hello')).resolves.toBe(true);
    expect(writeText).toHaveBeenCalled();
    expect(execCommand).toHaveBeenCalledWith('copy');
  });

  it('reports failure instead of a phantom success when nothing can copy', async () => {
    await expect(copyText('hello')).resolves.toBe(false);
  });

  it('reports failure when the fallback throws', async () => {
    setExecCommand(() => { throw new Error('unsupported'); });

    await expect(copyText('hello')).resolves.toBe(false);
  });

  it('leaves no scratch textarea behind', async () => {
    setExecCommand(vi.fn().mockReturnValue(true));

    await copyText('hello');

    expect(document.querySelectorAll('textarea')).toHaveLength(0);
  });

  it('hands the fallback the exact text to copy, then restores focus', async () => {
    const input = document.createElement('input');
    document.body.appendChild(input);
    input.focus();

    let seen = '';
    setExecCommand(() => {
      const area = document.querySelector('textarea');
      seen = area?.value ?? '';
      expect(document.activeElement).toBe(area);
      return true;
    });

    await copyText('multi\nline value');

    expect(seen).toBe('multi\nline value');
    expect(document.activeElement).toBe(input);

    input.remove();
  });
});
