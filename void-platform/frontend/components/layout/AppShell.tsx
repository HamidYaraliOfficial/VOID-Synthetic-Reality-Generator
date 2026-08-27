'use client';

// AppShell: the native-feeling window chrome tying every panel together —
// top Command Bar, left icon rail + Universe Explorer, center content
// (Canvas / Entity Designer / Behavior Editor / Dashboard / Scheduler),
// right Inspector (resizable, collapsible), and a bottom dock hosting the
// Timeline + Console (resizable, collapsible, tabbed).

import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  Boxes, Gauge, LayoutGrid, Moon, PanelBottomClose, PanelBottomOpen,
  PanelRightClose, PanelRightOpen, Search, Sun, Workflow, Clock, Palette, Globe, LogIn,
} from 'lucide-react';
import { useI18n } from '../../lib/i18n/I18nProvider';
import { Locale } from '../../lib/i18n/translations';
import { useAppStore, PanelId } from '../../lib/store';
import { useTheme, ThemeName } from '../theme/ThemeProvider';
import { api, ApiError, loadStoredToken, setAuthToken } from '../../lib/api';
import VoidMark from './VoidMark';
import UniverseExplorer from './UniverseExplorer';
import Inspector from './Inspector';
import TimelinePanel from './TimelinePanel';
import ConsolePanel from './ConsolePanel';
import SimulationCanvas from './SimulationCanvas';
import EntityDesigner from '../entity/EntityDesigner';
import BehaviorEditor from '../behavior/BehaviorEditor';
import Dashboard from '../dashboard/Dashboard';
import SchedulerPanel from '../scheduler/SchedulerPanel';
import CommandPalette from '../command/CommandPalette';

const NAV: { id: PanelId; icon: React.ReactNode; key: string }[] = [
  { id: 'canvas', icon: <LayoutGrid size={18} />, key: 'nav.canvas' },
  { id: 'entities', icon: <Boxes size={18} />, key: 'nav.entities' },
  { id: 'behaviors', icon: <Workflow size={18} />, key: 'nav.behaviors' },
  { id: 'dashboard', icon: <Gauge size={18} />, key: 'nav.dashboard' },
  { id: 'scheduler', icon: <Clock size={18} />, key: 'nav.scheduler' },
];

const THEMES: ThemeName[] = ['light', 'dark', 'win11', 'red', 'blue'];
const LOCALES: Locale[] = ['en', 'fa', 'zh'];

function ConnectBar() {
  const { t } = useI18n();
  const setAuthed = useAppStore((s) => s.setAuthed);
  const [username, setUsername] = useState('operator');
  const [role, setRole] = useState('engineer');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const connect = async () => {
    setBusy(true);
    setError(null);
    try {
      const { token } = await api.login(username, role);
      setAuthToken(token);
      setAuthed(true);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Could not reach the VOID API server.');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
      <input className="void-input" style={{ width: 110 }} value={username} onChange={(e) => setUsername(e.target.value)} />
      <select className="void-select" value={role} onChange={(e) => setRole(e.target.value)}>
        <option value="admin">admin</option>
        <option value="engineer">engineer</option>
        <option value="viewer">viewer</option>
      </select>
      <button className="void-btn void-btn-primary" onClick={connect} disabled={busy}>
        <LogIn size={14} /> {t('action.connect')}
      </button>
      {error && <span style={{ fontSize: 11, color: 'var(--danger)' }}>{error}</span>}
    </div>
  );
}

export default function AppShell() {
  const { t, locale, setLocale } = useI18n();
  const { theme, setTheme } = useTheme();
  const activePanel = useAppStore((s) => s.activePanel);
  const setActivePanel = useAppStore((s) => s.setActivePanel);
  const universe = useAppStore((s) => s.universe);
  const setUniverse = useAppStore((s) => s.setUniverse);
  const authed = useAppStore((s) => s.authed);
  const setAuthed = useAppStore((s) => s.setAuthed);
  const setCommandPaletteOpen = useAppStore((s) => s.setCommandPaletteOpen);

  const [rightOpen, setRightOpen] = useState(true);
  const [bottomOpen, setBottomOpen] = useState(true);
  const [rightWidth, setRightWidth] = useState(300);
  const [bottomHeight, setBottomHeight] = useState(240);
  const [bottomTab, setBottomTab] = useState<'timeline' | 'console'>('timeline');
  const [themeMenuOpen, setThemeMenuOpen] = useState(false);
  const [langMenuOpen, setLangMenuOpen] = useState(false);

  const dragState = useRef<{ mode: 'right' | 'bottom' | null }>({ mode: null });

  useEffect(() => {
    const stored = loadStoredToken();
    if (stored) setAuthed(true);
  }, [setAuthed]);

  // --- resize handles: real drag-to-resize for the Inspector width and the
  // bottom dock height, implementing the panel "Resize" requirement.
  const onDragMove = useCallback((e: MouseEvent) => {
    if (dragState.current.mode === 'right') {
      setRightWidth(Math.min(520, Math.max(220, window.innerWidth - e.clientX)));
    } else if (dragState.current.mode === 'bottom') {
      setBottomHeight(Math.min(520, Math.max(120, window.innerHeight - e.clientY)));
    }
  }, []);
  const onDragEnd = useCallback(() => {
    dragState.current.mode = null;
    window.removeEventListener('mousemove', onDragMove);
    window.removeEventListener('mouseup', onDragEnd);
  }, [onDragMove]);
  const startDrag = (mode: 'right' | 'bottom') => {
    dragState.current.mode = mode;
    window.addEventListener('mousemove', onDragMove);
    window.addEventListener('mouseup', onDragEnd);
  };

  const runControl = async (action: 'run' | 'pause' | 'resume' | 'stop') => {
    if (!universe) return;
    try {
      if (action === 'run') await api.runScenario(universe.id, { id: 'quick-run', name: 'quick-run', seed: universe.seed, timeScale: 60, timeline: [] });
      if (action === 'pause') await api.pause(universe.id);
      if (action === 'resume') await api.resume(universe.id);
      if (action === 'stop') await api.stop(universe.id);
      const refreshed = await api.getUniverse(universe.id);
      setUniverse(refreshed);
    } catch {
      /* surfaced via console tail */
    }
  };

  const statusClass = universe?.status ?? 'idle';

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* Top command bar */}
      <div style={{
        height: 52, display: 'flex', alignItems: 'center', gap: 10, padding: '0 14px',
        borderBottom: '1px solid var(--border)', background: 'var(--bg-panel)',
        backdropFilter: 'blur(20px)', flexShrink: 0,
      }}>
        <VoidMark size={26} />
        <div style={{ fontWeight: 800, fontSize: 14, letterSpacing: '.02em' }}>VOID</div>
        <div style={{ fontSize: 11, color: 'var(--fg-tertiary)', display: 'none' }} className="void-app-subtitle">{t('app.subtitle')}</div>

        <button className="void-btn" onClick={() => setCommandPaletteOpen(true)} style={{ marginInlineStart: 12 }}>
          <Search size={13} /> {t('command.placeholder')} <span className="void-mono" style={{ opacity: .6 }}>Ctrl K</span>
        </button>

        {universe && (
          <span className={`void-badge ${statusClass}`} style={{ marginInlineStart: 8 }}>
            <span className="void-badge-dot" /> {t('status.' + statusClass)}
          </span>
        )}

        <div style={{ flex: 1 }} />

        {universe && (
          <div style={{ display: 'flex', gap: 6 }}>
            <button className="void-btn void-btn-primary" onClick={() => runControl('run')}>{t('action.run')}</button>
            <button className="void-btn" onClick={() => runControl('pause')}>{t('action.pause')}</button>
            <button className="void-btn" onClick={() => runControl('resume')}>{t('action.resume')}</button>
            <button className="void-btn" onClick={() => runControl('stop')}>{t('action.stop')}</button>
          </div>
        )}

        <div style={{ position: 'relative' }}>
          <button className="void-icon-btn" onClick={() => { setLangMenuOpen((v) => !v); setThemeMenuOpen(false); }} title="Language">
            <Globe size={16} />
          </button>
          {langMenuOpen && (
            <div className="void-panel" style={{ position: 'absolute', insetInlineEnd: 0, top: 40, zIndex: 20, minWidth: 140, padding: 6 }}>
              {LOCALES.map((l) => (
                <button key={l} onClick={() => { setLocale(l); setLangMenuOpen(false); }}
                  style={{ display: 'block', width: '100%', textAlign: 'left', padding: '7px 10px', background: locale === l ? 'var(--bg-pressed)' : 'transparent', border: 'none', borderRadius: 6, color: 'var(--fg-primary)', fontSize: 13, cursor: 'pointer' }}>
                  {l === 'en' ? 'English' : l === 'fa' ? 'فارسی' : '中文'}
                </button>
              ))}
            </div>
          )}
        </div>

        <div style={{ position: 'relative' }}>
          <button className="void-icon-btn" onClick={() => { setThemeMenuOpen((v) => !v); setLangMenuOpen(false); }} title="Theme">
            {theme === 'light' || theme === 'win11' ? <Sun size={16} /> : <Moon size={16} />}
          </button>
          {themeMenuOpen && (
            <div className="void-panel" style={{ position: 'absolute', insetInlineEnd: 0, top: 40, zIndex: 20, minWidth: 190, padding: 6 }}>
              {THEMES.map((th) => (
                <button key={th} onClick={() => { setTheme(th); setThemeMenuOpen(false); }}
                  style={{ display: 'flex', alignItems: 'center', gap: 8, width: '100%', textAlign: 'left', padding: '7px 10px', background: theme === th ? 'var(--bg-pressed)' : 'transparent', border: 'none', borderRadius: 6, color: 'var(--fg-primary)', fontSize: 13, cursor: 'pointer' }}>
                  <Palette size={13} /> {t('theme.' + th)}
                </button>
              ))}
            </div>
          )}
        </div>

        <button className="void-icon-btn" onClick={() => setRightOpen((v) => !v)} title="Toggle Inspector">
          {rightOpen ? <PanelRightClose size={16} /> : <PanelRightOpen size={16} />}
        </button>
        <button className="void-icon-btn" onClick={() => setBottomOpen((v) => !v)} title="Toggle Dock">
          {bottomOpen ? <PanelBottomClose size={16} /> : <PanelBottomOpen size={16} />}
        </button>

        {!authed ? <ConnectBar /> : (
          <span className="void-badge" style={{ marginInlineStart: 4 }}><span className="void-badge-dot" style={{ background: 'var(--success)' }} />{t('label.connected')}</span>
        )}
      </div>

      {/* Body: left rail + explorer, center, right inspector */}
      <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>
        {/* icon rail */}
        <div style={{ width: 52, borderInlineEnd: '1px solid var(--border)', display: 'flex', flexDirection: 'column', alignItems: 'center', paddingTop: 10, gap: 4, flexShrink: 0 }}>
          {NAV.map((n) => (
            <button key={n.id} className="void-icon-btn" data-active={activePanel === n.id} title={t(n.key)} onClick={() => setActivePanel(n.id)}>
              {n.icon}
            </button>
          ))}
        </div>

        {/* universe explorer */}
        <div style={{ width: 240, borderInlineEnd: '1px solid var(--border)', flexShrink: 0, background: 'var(--bg-panel)' }}>
          <UniverseExplorer />
        </div>

        {/* center + bottom dock */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
          <div style={{ flex: 1, minHeight: 0 }}>
            {activePanel === 'canvas' && <SimulationCanvas />}
            {activePanel === 'entities' && <EntityDesigner />}
            {activePanel === 'behaviors' && <BehaviorEditor />}
            {activePanel === 'dashboard' && <Dashboard />}
            {activePanel === 'scheduler' && <SchedulerPanel />}
          </div>

          {bottomOpen && (
            <div style={{ height: bottomHeight, borderTop: '1px solid var(--border)', flexShrink: 0, display: 'flex', flexDirection: 'column' }}>
              <div onMouseDown={() => startDrag('bottom')} style={{ height: 5, cursor: 'row-resize', background: 'transparent' }} />
              <div style={{ display: 'flex', borderBottom: '1px solid var(--border)' }}>
                {(['timeline', 'console'] as const).map((tab) => (
                  <button key={tab} onClick={() => setBottomTab(tab)}
                    style={{
                      padding: '7px 16px', fontSize: 12.5, fontWeight: 600, border: 'none', background: 'transparent',
                      color: bottomTab === tab ? 'var(--accent)' : 'var(--fg-tertiary)',
                      borderBottom: bottomTab === tab ? '2px solid var(--accent)' : '2px solid transparent', cursor: 'pointer',
                    }}>
                    {t(tab === 'timeline' ? 'timeline.title' : 'console.title')}
                  </button>
                ))}
              </div>
              <div style={{ flex: 1, minHeight: 0 }}>
                {bottomTab === 'timeline' ? <TimelinePanel /> : <ConsolePanel />}
              </div>
            </div>
          )}
        </div>

        {/* right inspector */}
        {rightOpen && (
          <div style={{ display: 'flex', flexShrink: 0 }}>
            <div onMouseDown={() => startDrag('right')} style={{ width: 5, cursor: 'col-resize' }} />
            <div style={{ width: rightWidth, borderInlineStart: '1px solid var(--border)', background: 'var(--bg-panel)' }}>
              <Inspector />
            </div>
          </div>
        )}
      </div>

      <CommandPalette />
    </div>
  );
}
