'use client';

// Applies one of VOID's 5 themes as data-theme on <html>, matching the
// Windows 11 Fluent token sets defined in styles/themes.css.

import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';

export type ThemeName = 'light' | 'dark' | 'win11' | 'red' | 'blue';

interface ThemeContextValue {
  theme: ThemeName;
  setTheme: (t: ThemeName) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);
const STORAGE_KEY = 'void.theme';

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setThemeState] = useState<ThemeName>('dark');

  useEffect(() => {
    const stored = window.localStorage?.getItem(STORAGE_KEY) as ThemeName | null;
    if (stored) {
      setThemeState(stored);
    } else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
      setThemeState('light');
    }
  }, []);

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  const setTheme = useCallback((t: ThemeName) => {
    setThemeState(t);
    try {
      window.localStorage?.setItem(STORAGE_KEY, t);
    } catch {
      /* ignore */
    }
  }, []);

  return <ThemeContext.Provider value={{ theme, setTheme }}>{children}</ThemeContext.Provider>;
}

export function useTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider');
  return ctx;
}
