'use client';

// Live Dashboard: polls Universe metrics and renders Counter / Line Chart /
// Bar Chart widgets — a starter Dashboard Builder layout. Real-time values
// come straight from GET /api/universes/{id}/metrics (also streamed over
// the WebSocket at /api/ws/metrics for lower-latency widgets if desired).

import React, { useEffect, useState } from 'react';
import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis, Bar, BarChart, CartesianGrid } from 'recharts';
import { Activity, Database, Gauge as GaugeIcon, Zap } from 'lucide-react';
import { useI18n } from '../../lib/i18n/I18nProvider';
import { useAppStore } from '../../lib/store';
import { api } from '../../lib/api';

interface Point { t: string; events: number; entities: number }

function Counter({ icon, label, value }: { icon: React.ReactNode; label: string; value: string | number }) {
  return (
    <div className="void-card" style={{ display: 'flex', alignItems: 'center', gap: 12, flex: 1, minWidth: 160 }}>
      <div style={{ width: 36, height: 36, borderRadius: 8, background: 'var(--bg-hover)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        {icon}
      </div>
      <div>
        <div style={{ fontSize: 20, fontWeight: 700 }} className="void-mono">{value}</div>
        <div style={{ fontSize: 11, color: 'var(--fg-tertiary)' }}>{label}</div>
      </div>
    </div>
  );
}

export default function Dashboard() {
  const { t } = useI18n();
  const universe = useAppStore((s) => s.universe);
  const metrics = useAppStore((s) => s.metrics);
  const setMetrics = useAppStore((s) => s.setMetrics);
  const [history, setHistory] = useState<Point[]>([]);

  useEffect(() => {
    if (!universe) return;
    let cancelled = false;
    const poll = async () => {
      try {
        const m = await api.metrics(universe.id);
        if (cancelled) return;
        setMetrics(m);
        const totalEntities = Object.values(universe.entityCounts ?? {}).reduce((a, b) => a + b, 0);
        setHistory((h) => [
          ...h.slice(-29),
          { t: new Date(m.timestamp).toLocaleTimeString(), events: m.counters['events_processed_total'] ?? 0, entities: m.counters['entities_created_total'] ?? totalEntities },
        ]);
      } catch {
        /* keep last snapshot on transient failure */
      }
    };
    poll();
    const id = setInterval(poll, 2000);
    return () => { cancelled = true; clearInterval(id); };
  }, [universe, setMetrics]);

  const entityBars = universe ? Object.entries(universe.entityCounts ?? {}).map(([type, count]) => ({ type, count })) : [];

  return (
    <div className="void-scrollable" style={{ height: '100%', padding: 20 }}>
      <h2 style={{ margin: '0 0 16px', fontSize: 17 }}>{t('dashboard.title')}</h2>

      <div style={{ display: 'flex', gap: 12, marginBottom: 16, flexWrap: 'wrap' }}>
        <Counter icon={<Database size={16} color="var(--accent)" />} label="Entities" value={metrics?.counters['entities_created_total'] ?? 0} />
        <Counter icon={<Zap size={16} color="var(--accent)" />} label="Events processed" value={metrics?.counters['events_processed_total'] ?? 0} />
        <Counter icon={<Activity size={16} color="var(--accent)" />} label="Goroutines" value={metrics?.goroutineCount ?? 0} />
        <Counter icon={<GaugeIcon size={16} color="var(--accent)" />} label="Memory (MB)" value={metrics ? metrics.memoryAllocMB.toFixed(1) : '0.0'} />
      </div>

      <div className="void-card" style={{ marginBottom: 16, height: 220 }}>
        <div className="void-label" style={{ marginBottom: 8 }}>Event throughput</div>
        <ResponsiveContainer width="100%" height="88%">
          <LineChart data={history}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
            <XAxis dataKey="t" stroke="var(--fg-tertiary)" fontSize={10} />
            <YAxis stroke="var(--fg-tertiary)" fontSize={10} />
            <Tooltip contentStyle={{ background: 'var(--bg-panel-solid)', border: '1px solid var(--border)', fontSize: 12 }} />
            <Line type="monotone" dataKey="events" stroke="var(--accent)" strokeWidth={2} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>

      <div className="void-card" style={{ height: 220 }}>
        <div className="void-label" style={{ marginBottom: 8 }}>Entity population by type</div>
        <ResponsiveContainer width="100%" height="88%">
          <BarChart data={entityBars}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
            <XAxis dataKey="type" stroke="var(--fg-tertiary)" fontSize={10} />
            <YAxis stroke="var(--fg-tertiary)" fontSize={10} />
            <Tooltip contentStyle={{ background: 'var(--bg-panel-solid)', border: '1px solid var(--border)', fontSize: 12 }} />
            <Bar dataKey="count" fill="var(--accent)" radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
