import { createContext, useContext } from 'react';
import type { UXMode } from '@/utils/beginner-helpers';

export interface BeginnerUXContextType {
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

export const BeginnerUXContext = createContext<BeginnerUXContextType>(defaultBeginnerUXContext);

export function useBeginnerMode(): BeginnerUXContextType {
  return useContext(BeginnerUXContext);
}
