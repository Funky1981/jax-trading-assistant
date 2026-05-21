import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import type { UXMode } from '@/utils/beginner-helpers';
import { getBeginnerMode, setBeginnerMode } from '@/utils/beginner-helpers';

interface BeginnerUXContextType {
  mode: UXMode;
  setMode: (mode: UXMode) => void;
  toggleMode: () => void;
}

const defaultBeginnerUXContext: BeginnerUXContextType = {
  mode: 'simple',
  setMode: () => {
    // no-op fallback to avoid runtime crashes if used outside provider
  },
  toggleMode: () => {
    // no-op fallback to avoid runtime crashes if used outside provider
  },
};

const BeginnerUXContext = createContext<BeginnerUXContextType>(defaultBeginnerUXContext);

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

/**
 * Hook to access the beginner UX mode
 */
export function useBeginnerMode(): BeginnerUXContextType {
  return useContext(BeginnerUXContext);
}
