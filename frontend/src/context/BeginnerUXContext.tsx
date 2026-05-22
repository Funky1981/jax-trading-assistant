import { useState, useEffect, ReactNode } from 'react';
import type { UXMode } from '@/utils/beginner-helpers';
import { getBeginnerMode, setBeginnerMode } from '@/utils/beginner-helpers';
import { BeginnerUXContext } from './BeginnerUXContextValue';

export function BeginnerUXProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<UXMode>('simple');

  // Load from localStorage on mount
  useEffect(() => {
    const saved = getBeginnerMode();
    setModeState(saved);
  }, []);

  const setMode = (newMode: UXMode) => {
    setModeState(newMode);
    setBeginnerMode(newMode);
  };

  const toggleMode = () => {
    const modes: UXMode[] = ['simple', 'detailed', 'technical'];
    const currentIdx = modes.indexOf(mode);
    const nextIdx = (currentIdx + 1) % modes.length;
    setMode(modes[nextIdx]);
  };

  return (
    <BeginnerUXContext.Provider value={{ mode, setMode, toggleMode }}>
      {children}
    </BeginnerUXContext.Provider>
  );
}
