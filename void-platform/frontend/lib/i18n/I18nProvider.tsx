'use client';

// Provides the active Locale + a `t()` translator across the whole app, and
// keeps <html lang / dir> in sync so Persian genuinely renders right-to-left
// (mirrored layout, right-aligned text) while English/Chinese stay
// left-to-right — this is a real `dir` attribute switch, not a CSS hack.

import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { Locale, dictionaries, localeMeta } from './translations';

interface I18nContextValue {
  locale: Locale;
  dir: 'ltr' | 'rtl';
  setLocale: (l: Locale) => void;
  t: (key: string) => string;
}

const I18nContext = createContext<I18nContextValue | null>(null);

const STORAGE_KEY = 'void.locale';

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>('en');

  useEffect(() => {
    const stored = typeof window !== 'undefined' ? (window.localStorage?.getItem(STORAGE_KEY) as Locale | null) : null;
    if (stored && dictionaries[stored]) {
      setLocaleState(stored);
    }
  }, []);

  const setLocale = useCallback((l: Locale) => {
    setLocaleState(l);
    try {
      window.localStorage?.setItem(STORAGE_KEY, l);
    } catch {
      /* storage unavailable — locale still applies for this session */
    }
  }, []);

  const dir = localeMeta[locale].dir;

  useEffect(() => {
    document.documentElement.lang = locale;
    document.documentElement.dir = dir;
  }, [locale, dir]);

  const t = useCallback((key: string) => dictionaries[locale][key] ?? dictionaries.en[key] ?? key, [locale]);

  const value = useMemo(() => ({ locale, dir, setLocale, t }), [locale, dir, setLocale, t]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error('useI18n must be used within I18nProvider');
  return ctx;
}
