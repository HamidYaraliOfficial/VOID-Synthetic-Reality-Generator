'use client';

// Behavior Editor: build a Behavior Graph as a list of connected nodes
// (Event / Condition / Probability / Action / Delay / State Change / API
// Call / DB Operation / Loop), each pointing at the IDs of its "next" nodes
// — a compact, table-based rendering of the same graph the visual canvas
// editor targets on the backend (behavior.Graph).

import React, { useState } from 'react';
import { Plus, Save, Trash2, Workflow } from 'lucide-react';
import { useI18n } from '../../lib/i18n/I18nProvider';
import { useAppStore } from '../../lib/store';
import { api, ApiError } from '../../lib/api';
import type { BehaviorNode, NodeKind } from '../../lib/types';

const NODE_KINDS: NodeKind[] = ['event', 'condition', 'probability', 'action', 'delay', 'state_change', 'api_call', 'db_operation', 'loop'];

let nodeSeq = 0;
function blankNode(): BehaviorNode {
  nodeSeq += 1;
  return { id: `node_${nodeSeq}`, kind: 'event', params: {}, next: [] };
}

export default function BehaviorEditor() {
  const { t } = useI18n();
  const universe = useAppStore((s) => s.universe);

  const [graphName, setGraphName] = useState('user-journey');
  const [entry, setEntry] = useState('node_1');
  const [nodes, setNodes] = useState<BehaviorNode[]>([
    { id: 'node_1', kind: 'event', params: { type: 'login' }, next: ['node_2'] },
    { id: 'node_2', kind: 'probability', params: { p: 0.3 }, next: ['node_3'], onFailure: [] },
    { id: 'node_3', kind: 'event', params: { type: 'purchase' }, next: [] },
  ]);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const updateNode = (i: number, patch: Partial<BehaviorNode>) =>
    setNodes((ns) => ns.map((n, idx) => (idx === i ? { ...n, ...patch } : n)));
  const removeNode = (i: number) => setNodes((ns) => ns.filter((_, idx) => idx !== i));

  const save = async () => {
    if (!universe) return;
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      const nodeMap: Record<string, BehaviorNode> = {};
      nodes.forEach((n) => { nodeMap[n.id] = n; });
      await api.addBehavior(universe.id, { name: graphName, entry, nodes: nodeMap });
      setMessage(`Behavior "${graphName}" saved — attach it to entities via a Timeline "attach" action.`);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to save behavior');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="void-scrollable" style={{ height: '100%', padding: 20 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 16 }}>
        <Workflow size={18} color="var(--accent)" />
        <h2 style={{ margin: 0, fontSize: 17 }}>{t('behavior.title')}</h2>
      </div>

      <div className="void-card">
        <div style={{ display: 'flex', gap: 10, marginBottom: 14, flexWrap: 'wrap' }}>
          <input className="void-input" value={graphName} onChange={(e) => setGraphName(e.target.value)} style={{ width: 200 }} placeholder="graph name" />
          <input className="void-input" value={entry} onChange={(e) => setEntry(e.target.value)} style={{ width: 140 }} placeholder="entry node id" />
          <div style={{ flex: 1 }} />
          <button className="void-btn" onClick={() => setNodes((ns) => [...ns, blankNode()])}>
            <Plus size={14} /> node
          </button>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {nodes.map((n, i) => (
            <div key={i} style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
              <input className="void-input void-mono" style={{ width: 110 }} value={n.id}
                onChange={(e) => updateNode(i, { id: e.target.value })} />
              <select className="void-select" value={n.kind} onChange={(e) => updateNode(i, { kind: e.target.value as NodeKind })} style={{ width: 130 }}>
                {NODE_KINDS.map((k) => <option key={k} value={k}>{k}</option>)}
              </select>
              <input className="void-input void-mono" style={{ flex: 1 }}
                value={JSON.stringify(n.params ?? {})}
                onChange={(e) => {
                  try { updateNode(i, { params: JSON.parse(e.target.value) }); } catch { /* keep typing */ }
                }} placeholder="params JSON" />
              <input className="void-input void-mono" style={{ width: 150 }}
                value={(n.next ?? []).join(',')}
                onChange={(e) => updateNode(i, { next: e.target.value.split(',').map((s) => s.trim()).filter(Boolean) })}
                placeholder="next ids" />
              <button className="void-icon-btn" onClick={() => removeNode(i)}><Trash2 size={14} /></button>
            </div>
          ))}
        </div>

        <div style={{ marginTop: 16 }}>
          <button className="void-btn void-btn-primary" onClick={save} disabled={!universe || busy}>
            <Save size={14} /> {t('action.save')}
          </button>
        </div>
        {message && <div style={{ color: 'var(--success)', fontSize: 12, marginTop: 8 }}>{message}</div>}
        {error && <div style={{ color: 'var(--danger)', fontSize: 12, marginTop: 8 }}>{error}</div>}
        {!universe && <div style={{ fontSize: 12, color: 'var(--fg-tertiary)', marginTop: 8 }}>Create a Universe first.</div>}
      </div>
    </div>
  );
}
