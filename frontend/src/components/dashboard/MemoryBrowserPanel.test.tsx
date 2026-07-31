import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { MemoryBrowserPanel } from './MemoryBrowserPanel';

const useMemoryBanks = vi.fn();
vi.mock('@/hooks/useMemory', () => ({
  useMemoryBanks: () => useMemoryBanks(),
  useMemoryRecall: () => ({ data: undefined, isLoading: false }),
  useMemorySearch: () => ({ data: undefined, isLoading: false }),
}));

describe('MemoryBrowserPanel diagnostics', () => {
  it('shows an honest unavailable state when the memory-bank diagnostic fails', () => {
    useMemoryBanks.mockReturnValue({ data: undefined, isLoading: false, isError: true });

    render(<MemoryBrowserPanel isOpen onToggle={() => undefined} />);

    expect(screen.getByRole('alert')).toHaveTextContent('Memory-bank diagnostics are unavailable.');
    expect(screen.queryByText(/memory banks/i)).not.toBeInTheDocument();
  });
});
