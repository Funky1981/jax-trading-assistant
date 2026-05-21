import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { BeginnerUXProvider, useBeginnerMode } from './BeginnerUXContext';

const setBeginnerModeSpy = vi.fn();
const getBeginnerModeSpy = vi.fn(() => 'simple');

vi.mock('@/utils/beginner-helpers', () => ({
  getBeginnerMode: () => getBeginnerModeSpy(),
  setBeginnerMode: (mode: 'simple' | 'detailed' | 'technical') => setBeginnerModeSpy(mode),
}));

function Probe() {
  const { mode, setMode, toggleMode } = useBeginnerMode();

  return (
    <div>
      <p data-testid="mode">{mode}</p>
      <button onClick={() => setMode('technical')}>set technical</button>
      <button onClick={toggleMode}>toggle</button>
    </div>
  );
}

describe('BeginnerUXContext', () => {
  beforeEach(() => {
    setBeginnerModeSpy.mockReset();
    getBeginnerModeSpy.mockReset();
    getBeginnerModeSpy.mockReturnValue('simple');
  });

  it('returns safe defaults when used outside provider', () => {
    render(<Probe />);

    expect(screen.getByTestId('mode')).toHaveTextContent('simple');

    fireEvent.click(screen.getByText('set technical'));
    fireEvent.click(screen.getByText('toggle'));

    expect(setBeginnerModeSpy).not.toHaveBeenCalled();
  });

  it('loads persisted mode and updates storage through provider', () => {
    getBeginnerModeSpy.mockReturnValue('detailed');

    render(
      <BeginnerUXProvider>
        <Probe />
      </BeginnerUXProvider>
    );

    expect(screen.getByTestId('mode')).toHaveTextContent('detailed');

    fireEvent.click(screen.getByText('set technical'));

    expect(screen.getByTestId('mode')).toHaveTextContent('technical');
    expect(setBeginnerModeSpy).toHaveBeenCalledWith('technical');
  });
});
