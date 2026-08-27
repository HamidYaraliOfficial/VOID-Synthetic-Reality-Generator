'use client';

// Entity Designer: build a Schema by adding typed Fields (with a Generator
// per field), then save it to the active Universe and Spawn instances of it.

import React, { useState } from 'react';
import { Play, Plus, Save, Trash2 } from 'lucide-react';
import { useI18n } from '../../lib/i18n/I18nProvider';
import { useAppStore } from '../../lib/store';
import { api, ApiError } from '../../lib/api';
import type { EntityField, FieldType, GeneratorKind } from '../../lib/types';

const FIELD_TYPES: FieldType[] = ['string', 'integer', 'float', 'boolean', 'datetime', 'uuid', 'enum', 'array', 'json', 'binary', 'custom'];
const GENERATORS: GeneratorKind[] = ['random', 'sequential', 'weighted_random', 'uuid', 'name', 'email', 'phone', 'address', 'date', 'time', 'number', 'pattern', 'distribution', 'dependent', 'derived', 'custom_function'];

let fieldSeq = 0;
function blankField(): EntityField {
  fieldSeq += 1;
  return { name: `field_${fieldSeq}`, type: 'string', generator: 'random' };
}

export default function EntityDesigner() {
  const { t } = useI18n();
  const universe = useAppStore((s) => s.universe);
  const setUniverse = useAppStore((s) => s.setUniverse);
  const upsertSchema = useAppStore((s) => s.upsertSchema);

  const [schemaName, setSchemaName] = useState('User');
  const [fields, setFields] = useState<EntityField[]>([
    { name: 'id', type: 'uuid', generator: 'uuid' },
    { name: 'name', type: 'string', generator: 'name' },
    { name: 'email', type: 'string', generator: 'email' },
  ]);
  const [count, setCount] = useState('1000');
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const updateField = (i: number, patch: Partial<EntityField>) =>
    setFields((fs) => fs.map((f, idx) => (idx === i ? { ...f, ...patch } : f)));
  const removeField = (i: number) => setFields((fs) => fs.filter((_, idx) => idx !== i));

  const saveSchema = async () => {
    if (!universe) return;
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      const schema = { name: schemaName, fields };
      const saved = await api.addSchema(universe.id, schema);
      upsertSchema(saved);
      setMessage(`Schema "${schemaName}" saved.`);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to save schema');
    } finally {
      setBusy(false);
    }
  };

  const spawnNow = async () => {
    if (!universe) return;
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      const result = await api.spawn(universe.id, schemaName, Number(count) || 0);
      const refreshed = await api.getUniverse(universe.id);
      setUniverse(refreshed);
      setMessage(`Spawned ${result.spawned} ${schemaName} entities.`);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to spawn entities');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="void-scrollable" style={{ height: '100%', padding: 20 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 17 }}>{t('entity.title')}</h2>
      </div>

      <div className="void-card" style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', gap: 10, alignItems: 'center', marginBottom: 14, flexWrap: 'wrap' }}>
          <label className="void-label">{t('label.entityType')}</label>
          <input className="void-input" value={schemaName} onChange={(e) => setSchemaName(e.target.value)} style={{ width: 200 }} />
          <div style={{ flex: 1 }} />
          <button className="void-btn" onClick={() => setFields((fs) => [...fs, blankField()])}>
            <Plus size={14} /> {t('action.addField')}
          </button>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {fields.map((f, i) => (
            <div key={i} style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
              <input className="void-input" style={{ flex: 1 }} value={f.name}
                onChange={(e) => updateField(i, { name: e.target.value })} placeholder={t('label.fieldName')} />
              <select className="void-select" value={f.type} onChange={(e) => updateField(i, { type: e.target.value as FieldType })} style={{ width: 110 }}>
                {FIELD_TYPES.map((ty) => <option key={ty} value={ty}>{ty}</option>)}
              </select>
              <select className="void-select" value={f.generator} onChange={(e) => updateField(i, { generator: e.target.value as GeneratorKind })} style={{ width: 150 }}>
                {GENERATORS.map((g) => <option key={g} value={g}>{g}</option>)}
              </select>
              {f.generator === 'dependent' && (
                <input className="void-input" style={{ width: 140 }} placeholder="relatedType"
                  value={(f.params?.relatedType as string) || ''}
                  onChange={(e) => updateField(i, { params: { ...f.params, relatedType: e.target.value } })} />
              )}
              <button className="void-icon-btn" onClick={() => removeField(i)}><Trash2 size={14} /></button>
            </div>
          ))}
        </div>

        <div style={{ display: 'flex', gap: 8, marginTop: 16 }}>
          <button className="void-btn void-btn-primary" onClick={saveSchema} disabled={!universe || busy}>
            <Save size={14} /> {t('action.save')}
          </button>
          <input className="void-input" style={{ width: 100 }} value={count} onChange={(e) => setCount(e.target.value)} />
          <button className="void-btn" onClick={spawnNow} disabled={!universe || busy}>
            <Play size={14} /> {t('action.spawn')}
          </button>
        </div>
        {message && <div style={{ color: 'var(--success)', fontSize: 12, marginTop: 8 }}>{message}</div>}
        {error && <div style={{ color: 'var(--danger)', fontSize: 12, marginTop: 8 }}>{error}</div>}
        {!universe && <div style={{ fontSize: 12, color: 'var(--fg-tertiary)', marginTop: 8 }}>Create a Universe first.</div>}
      </div>
    </div>
  );
}
