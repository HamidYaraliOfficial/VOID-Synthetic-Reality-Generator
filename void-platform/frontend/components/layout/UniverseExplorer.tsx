'use client';

// Left "Universe Explorer" panel: create/connect a Universe, and browse its
// live Entity population as an expandable tree of entity types + counts.

import React, { useState } from 'react';
import { Boxes, Plus, Radio } from 'lucide-react';
import { useI18n } from '../../lib/i18n/I18nProvider';
import { useAppStore } from '../../lib/store';
import { api, ApiError } from '../../lib/api';

export default function UniverseExplorer() {
  const { t } = useI18n();
  const universe = useAppStore((s) => s.universe);
  const setUniverse = useAppStore((s) => s.setUniverse);
  const selectedType = useAppStore((s) => s.selectedEntityType);
  const setSelectedType = useAppStore((s) => s.setSelectedEntityType);
  const setInspector = useAppStore((s) => s.setInspectorContent);

  const [name, setName] = useState('My Universe');
  const [seed, setSeed] = useState('42');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createUniverse = async () => {
    setBusy(true);
    setError(null);
    try {
      const id = name.trim().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '') || `universe-${Date.now()}`;
      const u = await api.createUniverse(id, name, Number(seed) || 0);
      setUniverse(u);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to create universe');
    } finally {
      setBusy(false);
    }
  };

  const entityTypes = universe ? Object.entries(universe.entityCounts ?? {}) : [];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ padding: 'var(--space-3) var(--space-4)', borderBottom: '1px solid var(--border)' }}>
        <div className="void-label" style={{ marginBottom: 8 }}>{t('nav.universe')}</div>
        {!universe ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <input className="void-input" placeholder={t('label.universeName')} value={name} onChange={(e) => setName(e.target.value)} />
            <input className="void-input" placeholder={t('label.seed')} value={seed} onChange={(e) => setSeed(e.target.value)} />
            <button className="void-btn void-btn-primary" onClick={createUniverse} disabled={busy}>
              <Plus size={14} /> {t('action.add')}
            </button>
            {error && <div style={{ color: 'var(--danger)', fontSize: 12 }}>{error}</div>}
          </div>
        ) : (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Radio size={14} color="var(--accent)" />
            <div>
              <div style={{ fontWeight: 700, fontSize: 13 }}>{universe.name}</div>
              <div style={{ fontSize: 11, color: 'var(--fg-tertiary)' }} className="void-mono">
                seed {universe.seed} · {universe.id}
              </div>
            </div>
          </div>
        )}
      </div>

      <div className="void-scrollable" style={{ flex: 1, padding: 'var(--space-2) var(--space-3)' }}>
        <div className="void-label" style={{ margin: '8px 4px' }}>Entities</div>
        {entityTypes.length === 0 && (
          <div style={{ fontSize: 12, color: 'var(--fg-tertiary)', padding: '0 4px' }}>
            {universe ? 'No entities yet — use Entity Designer to spawn some.' : 'Create a Universe to begin.'}
          </div>
        )}
        {entityTypes.map(([type, count]) => (
          <button
            key={type}
            onClick={() => {
              setSelectedType(type);
              setInspector({ title: type, body: { entityType: type, count } });
            }}
            style={{
              display: 'flex', alignItems: 'center', gap: 8, width: '100%',
              padding: '7px 8px', borderRadius: 6, border: 'none', cursor: 'pointer',
              background: selectedType === type ? 'var(--bg-pressed)' : 'transparent',
              color: 'var(--fg-primary)', fontSize: 13, textAlign: 'left', marginBottom: 2,
            }}
          >
            <Boxes size={14} color="var(--accent)" />
            <span style={{ flex: 1 }}>{type}</span>
            <span className="void-mono" style={{ fontSize: 11, color: 'var(--fg-tertiary)' }}>{count}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
