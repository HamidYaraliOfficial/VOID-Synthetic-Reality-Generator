'use client';

// Global Command Palette (Ctrl+K / Cmd+K): fuzzy-searchable list of actions
// — navigate to a panel, control the running scenario, switch theme/language.

import React, { useEffect, useMemo, useState } from 'react';
import { Search } from 'lucide-react';
import { useI18n } from '../../lib/i18n/I18nProvider';
import { useAppStore, PanelId } from '../../lib/store';
import { useTheme, ThemeName } from '../theme/ThemeProvider';
import { api } from '../../lib/api';
import type { Locale } from '../../lib/i18n/translations';

interface Command {
  id: string;
  label: string;
  hint?: string;
  run: () => void;
}

export default function CommandPalette() {
  const { t, setLocale } = useI18n();
  const open = useAppStore((s) => s.commandPaletteOpen);
  const setOpen = useAppStore((s) => s.setCommandPaletteOpen);
  const setActivePanel = useAppStore((s) => s.setActivePanel);
  const universe = useAppStore((s) => s.universe);
  const { setTheme } = useTheme();
  const [query, setQuery] = useState('');

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setOpen(!open);
      }
      if (e.key === 'Escape') setOpen(false);
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [open, setOpen]);

  const panelCommands: Command[] = (['canvas', 'entities', 'behaviors', 'dashboard', 'scheduler'] as PanelId[]).map((p) => ({
    id: `panel-${p}`,
    label: `${t('nav.' + (p === 'entities' ? 'entities' : p === 'behaviors' ? 'behaviors' : p))}`,
    hint: 'Go to',
    run: () => setActivePanel(p),
  }));

  const runCommands: Command[] = universe
    ? [
        { id: 'run-pause', label: t('action.pause'), run: () => api.pause(universe.id).catch(() => {}) },
        { id: 'run-resume', label: t('action.resume'), run: () => api.resume(universe.id).catch(() => {}) },
        { id: 'run-stop', label: t('action.stop'), run: () => api.stop(universe.id).catch(() => {}) },
        { id: 'run-snapshot', label: t('action.snapshot'), run: () => api.snapshot(universe.id, `manual-${Date.now()}`).catch(() => {}) },
      ]
    : [];

  const themeCommands: Command[] = (['light', 'dark', 'win11', 'red', 'blue'] as ThemeName[]).map((th) => ({
    id: `theme-${th}`,
    label: `${t('nav.settings')}: ${t('theme.' + th)}`,
    run: () => setTheme(th),
  }));

  const langCommands: Command[] = (['en', 'fa', 'zh'] as Locale[]).map((l) => ({
    id: `lang-${l}`,
    label: `${t('nav.settings')}: ${l.toUpperCase()}`,
    run: () => setLocale(l),
  }));

  const all = useMemo(() => [...panelCommands, ...runCommands, ...themeCommands, ...langCommands], [universe, t]); // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = query.trim()
    ? all.filter((c) => c.label.toLowerCase().includes(query.toLowerCase()))
    : all;

  if (!open) return null;

  return (
    <div
      onClick={() => setOpen(false)}
      style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,.45)', zIndex: 100, display: 'flex', justifyContent: 'center', paddingTop: '12vh' }}
    >
      <div onClick={(e) => e.stopPropagation()} className="void-panel" style={{ width: 560, maxWidth: '92vw', maxHeight: '60vh', display: 'flex', flexDirection: 'column', boxShadow: 'var(--shadow-elevation-3)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '12px 14px', borderBottom: '1px solid var(--border)' }}>
          <Search size={16} color="var(--fg-tertiary)" />
          <input
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('command.placeholder')}
            style={{ flex: 1, border: 'none', outline: 'none', background: 'transparent', color: 'var(--fg-primary)', fontSize: 14 }}
          />
          <span style={{ fontSize: 11, color: 'var(--fg-tertiary)' }}>{t('command.hint')}</span>
        </div>
        <div className="void-scrollable" style={{ flex: 1, padding: 6 }}>
          {filtered.map((c) => (
            <button
              key={c.id}
              onClick={() => { c.run(); setOpen(false); setQuery(''); }}
              style={{
                display: 'flex', width: '100%', justifyContent: 'space-between', alignItems: 'center',
                padding: '9px 10px', background: 'transparent', border: 'none', borderRadius: 6,
                color: 'var(--fg-primary)', fontSize: 13, cursor: 'pointer', textAlign: 'left',
              }}
              onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--bg-hover)')}
              onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
            >
              <span>{c.label}</span>
              {c.hint && <span style={{ fontSize: 11, color: 'var(--fg-tertiary)' }}>{c.hint}</span>}
            </button>
          ))}
          {filtered.length === 0 && <div style={{ padding: 14, fontSize: 13, color: 'var(--fg-tertiary)' }}>No matches.</div>}
        </div>
      </div>
    </div>
  );
}
