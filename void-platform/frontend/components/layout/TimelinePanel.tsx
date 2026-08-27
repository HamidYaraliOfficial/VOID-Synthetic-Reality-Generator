'use client';

// Bottom "Timeline" editor: build a Scenario as an ordered list of Timeline
// Actions (spawn / emit / load / chaos / wait / snapshot / set_network),
// each with a start offset and optional duration, then Run it against the
// active Universe. This is a compact, list-based rendering of the same
// scenario.Action[] the visual Timeline Editor targets on the backend.

import React, { useState } from 'react';
import { Play, Plus, Trash2 } from 'lucide-react';
import { useI18n } from '../../lib/i18n/I18nProvider';
import { useAppStore } from '../../lib/store';
import { api, ApiError } from '../../lib/api';

type ActionKind = 'spawn' | 'attach' | 'emit' | 'load' | 'chaos' | 'wait' | 'snapshot' | 'set_network';

interface Row {
  id: string;
  kind: ActionKind;
  atSeconds: number;
  durationSeconds: number;
  paramsText: string;
}

let rowSeq = 0;
function newRow(kind: ActionKind = 'spawn'): Row {
  rowSeq += 1;
  return { id: `action-${rowSeq}`, kind, atSeconds: 0, durationSeconds: 0, paramsText: '{"schema":"User","count":100}' };
}

const KIND_OPTIONS: ActionKind[] = ['spawn', 'attach', 'emit', 'load', 'chaos', 'wait', 'snapshot', 'set_network'];

export default function TimelinePanel() {
  const { t } = useI18n();
  const universe = useAppStore((s) => s.universe);
  const setUniverse = useAppStore((s) => s.setUniverse);
  const [rows, setRows] = useState<Row[]>([newRow('spawn')]);
  const [scenarioName, setScenarioName] = useState('scenario-1');
  const [timeScale, setTimeScale] = useState('60');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const maxOffset = Math.max(1, ...rows.map((r) => r.atSeconds + r.durationSeconds));

  const updateRow = (id: string, patch: Partial<Row>) => setRows((rs) => rs.map((r) => (r.id === id ? { ...r, ...patch } : r)));
  const removeRow = (id: string) => setRows((rs) => rs.filter((r) => r.id !== id));

  const run = async () => {
    if (!universe) return;
    setBusy(true);
    setError(null);
    try {
      const timeline = rows.map((r) => {
        let params: Record<string, unknown> = {};
        try {
          params = JSON.parse(r.paramsText || '{}');
        } catch {
          /* keep empty params on invalid JSON rather than block the run */
        }
        return {
          id: r.id,
          at: Math.round(r.atSeconds * 1e9),
          kind: r.kind,
          duration: Math.round(r.durationSeconds * 1e9),
          params,
        };
      });
      await api.runScenario(universe.id, {
        id: scenarioName, name: scenarioName, seed: universe.seed,
        timeScale: Number(timeScale) || 1, timeline,
      });
      const refreshed = await api.getUniverse(universe.id);
      setUniverse(refreshed);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to run scenario');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px', borderBottom: '1px solid var(--border)' }}>
        <input className="void-input" style={{ width: 160 }} value={scenarioName} onChange={(e) => setScenarioName(e.target.value)} />
        <label className="void-label">scale</label>
        <input className="void-input" style={{ width: 64 }} value={timeScale} onChange={(e) => setTimeScale(e.target.value)} />
        <button className="void-btn" onClick={() => setRows((rs) => [...rs, newRow()])}>
          <Plus size={14} /> action
        </button>
        <div style={{ flex: 1 }} />
        <button className="void-btn void-btn-primary" onClick={run} disabled={!universe || busy}>
          <Play size={14} /> {t('action.run')}
        </button>
      </div>

      {/* mini offset ruler */}
      <div style={{ position: 'relative', height: 26, margin: '6px 12px', borderBottom: '1px solid var(--border)' }}>
        {rows.map((r) => (
          <div
            key={r.id}
            title={`${r.kind} @ ${r.atSeconds}s`}
            style={{
              position: 'absolute', top: 4, height: 14,
              left: `${(r.atSeconds / maxOffset) * 100}%`,
              width: `${Math.max(1.5, (r.durationSeconds / maxOffset) * 100)}%`,
              background: 'var(--accent)', opacity: 0.75, borderRadius: 3,
            }}
          />
        ))}
      </div>

      <div className="void-scrollable" style={{ flex: 1, padding: '4px 12px 12px' }}>
        {rows.map((r) => (
          <div key={r.id} style={{ display: 'flex', gap: 6, alignItems: 'center', marginBottom: 6 }}>
            <select className="void-select" value={r.kind} onChange={(e) => updateRow(r.id, { kind: e.target.value as ActionKind })} style={{ width: 120 }}>
              {KIND_OPTIONS.map((k) => (
                <option key={k} value={k}>{k}</option>
              ))}
            </select>
            <input className="void-input" style={{ width: 70 }} type="number" value={r.atSeconds}
              onChange={(e) => updateRow(r.id, { atSeconds: Number(e.target.value) })} title="offset (s)" />
            <input className="void-input" style={{ width: 70 }} type="number" value={r.durationSeconds}
              onChange={(e) => updateRow(r.id, { durationSeconds: Number(e.target.value) })} title="duration (s)" />
            <input className="void-input void-mono" style={{ flex: 1 }} value={r.paramsText}
              onChange={(e) => updateRow(r.id, { paramsText: e.target.value })} />
            <button className="void-icon-btn" onClick={() => removeRow(r.id)}>
              <Trash2 size={14} />
            </button>
          </div>
        ))}
        {error && <div style={{ color: 'var(--danger)', fontSize: 12 }}>{error}</div>}
        {!universe && <div style={{ fontSize: 12, color: 'var(--fg-tertiary)' }}>Create a Universe first.</div>}
      </div>
    </div>
  );
}
