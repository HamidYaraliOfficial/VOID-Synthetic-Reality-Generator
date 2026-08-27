'use client';

// Bottom "Console" feed: tails the Universe's server-side log lines (spawn
// events, scenario start/stop, chaos faults, ...) by polling GET
// /api/universes/{id}/console every second while a Universe is active.

import React, { useEffect, useRef } from 'react';
import { useI18n } from '../../lib/i18n/I18nProvider';
import { useAppStore } from '../../lib/store';
import { api } from '../../lib/api';

export default function ConsolePanel() {
  const { t } = useI18n();
  const universe = useAppStore((s) => s.universe);
  const lines = useAppStore((s) => s.consoleLines);
  const setLines = useAppStore((s) => s.setConsoleLines);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!universe) return;
    let cancelled = false;
    const poll = async () => {
      try {
        const tail = await api.console(universe.id);
        if (!cancelled) setLines(tail);
      } catch {
        /* connection hiccup — keep last known lines */
      }
    };
    poll();
    const id = setInterval(poll, 1500);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [universe, setLines]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [lines]);

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', background: 'var(--console-bg)' }}>
      <div className="void-scrollable void-mono" style={{ flex: 1, padding: '10px 14px', fontSize: 12.5, color: 'var(--console-fg)', lineHeight: 1.6 }}>
        {lines.length === 0 && <div style={{ opacity: 0.5 }}>{t('console.title')}: —</div>}
        {lines.map((l, i) => (
          <div key={i}>{l}</div>
        ))}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}
