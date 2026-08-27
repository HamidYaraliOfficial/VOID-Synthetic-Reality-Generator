'use client';

// Right "Inspector" panel: shows details for whatever is currently selected
// (an entity type, a behavior node, an event) as a simple key/value view —
// the Event Inspector / entity History surface from the product spec.

import React from 'react';
import { useI18n } from '../../lib/i18n/I18nProvider';
import { useAppStore } from '../../lib/store';

export default function Inspector() {
  const { t } = useI18n();
  const content = useAppStore((s) => s.inspectorContent);

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: '10px 14px', borderBottom: '1px solid var(--border)', fontWeight: 700, fontSize: 13 }}>
        {t('nav.inspector')}
      </div>
      <div className="void-scrollable" style={{ flex: 1, padding: 14 }}>
        {!content ? (
          <div style={{ fontSize: 12, color: 'var(--fg-tertiary)' }}>{t('inspector.empty')}</div>
        ) : (
          <div>
            <div style={{ fontWeight: 700, marginBottom: 10, fontSize: 14 }}>{content.title}</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {Object.entries(content.body).map(([k, v]) => (
                <div key={k} style={{ display: 'flex', justifyContent: 'space-between', gap: 8, fontSize: 12.5, borderBottom: '1px dashed var(--border)', paddingBottom: 6 }}>
                  <span style={{ color: 'var(--fg-tertiary)' }}>{k}</span>
                  <span className="void-mono" style={{ textAlign: 'right', wordBreak: 'break-all' }}>
                    {typeof v === 'object' ? JSON.stringify(v) : String(v)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
