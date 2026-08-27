'use client';

// Center "Simulation Canvas": a pannable/zoomable SVG view of the Universe's
// synthetic Network Topology / Service Dependency Graph. Nodes are laid out
// on a simple ring (a full force-directed layout is a natural next step;
// this already gives real pan+zoom+inspect over live backend data).

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useI18n } from '../../lib/i18n/I18nProvider';
import { useAppStore } from '../../lib/store';

interface NetNode { id: string; name: string; kind: string; available: boolean }
interface NetLink { from: string; to: string; latencyMs: number; packetLoss: number; up: boolean }

export default function SimulationCanvas() {
  const { t } = useI18n();
  const universe = useAppStore((s) => s.universe);
  const setInspector = useAppStore((s) => s.setInspectorContent);
  const [nodes, setNodes] = useState<NetNode[]>([]);
  const [links, setLinks] = useState<NetLink[]>([]);
  const [transform, setTransform] = useState({ x: 0, y: 0, scale: 1 });
  const dragRef = useRef<{ x: number; y: number } | null>(null);

  useEffect(() => {
    if (!universe) return;
    let cancelled = false;
    const load = async () => {
      try {
        const base = process.env.NEXT_PUBLIC_VOID_API_BASE || 'http://localhost:8080';
        const token = typeof window !== 'undefined' ? window.localStorage.getItem('void.token') : null;
        const res = await fetch(`${base}/api/universes/${universe.id}/network`, {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        });
        if (!res.ok) return;
        const data = await res.json();
        if (!cancelled) {
          setNodes(data.nodes ?? []);
          setLinks(data.links ?? []);
        }
      } catch {
        /* ignore transient errors, keep last layout */
      }
    };
    load();
    const id = setInterval(load, 3000);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [universe]);

  const positions = useCallback(() => {
    const cx = 400, cy = 260, r = Math.min(220, 60 + nodes.length * 12);
    const map = new Map<string, { x: number; y: number }>();
    nodes.forEach((n, i) => {
      const angle = (i / Math.max(1, nodes.length)) * Math.PI * 2;
      map.set(n.id, { x: cx + r * Math.cos(angle), y: cy + r * Math.sin(angle) });
    });
    return map;
  }, [nodes]);
  const pos = positions();

  const onWheel = (e: React.WheelEvent) => {
    e.preventDefault();
    setTransform((tr) => ({ ...tr, scale: Math.min(3, Math.max(0.4, tr.scale - e.deltaY * 0.001)) }));
  };
  const onMouseDown = (e: React.MouseEvent) => { dragRef.current = { x: e.clientX - transform.x, y: e.clientY - transform.y }; };
  const onMouseMove = (e: React.MouseEvent) => {
    if (!dragRef.current) return;
    setTransform((tr) => ({ ...tr, x: e.clientX - dragRef.current!.x, y: e.clientY - dragRef.current!.y }));
  };
  const onMouseUp = () => { dragRef.current = null; };

  return (
    <div style={{ height: '100%', position: 'relative', overflow: 'hidden', background: 'var(--bg-canvas)' }}>
      <div style={{ position: 'absolute', top: 10, insetInlineStart: 14, zIndex: 2 }} className="void-label">
        {t('nav.canvas')} {nodes.length > 0 ? `· ${nodes.length} nodes / ${links.length} links` : ''}
      </div>
      {nodes.length === 0 && (
        <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--fg-tertiary)', fontSize: 13 }}>
          {universe ? 'No network topology yet — add nodes/links via the API to visualize your Service Dependency Graph.' : 'Create a Universe to begin.'}
        </div>
      )}
      <svg
        width="100%" height="100%"
        onWheel={onWheel} onMouseDown={onMouseDown} onMouseMove={onMouseMove} onMouseUp={onMouseUp} onMouseLeave={onMouseUp}
        style={{ cursor: dragRef.current ? 'grabbing' : 'grab' }}
      >
        <g transform={`translate(${transform.x},${transform.y}) scale(${transform.scale})`}>
          {links.map((l, i) => {
            const a = pos.get(l.from), b = pos.get(l.to);
            if (!a || !b) return null;
            return (
              <line key={i} x1={a.x} y1={a.y} x2={b.x} y2={b.y}
                stroke={l.up ? 'var(--accent)' : 'var(--danger)'}
                strokeWidth={Math.max(1, 3 - l.latencyMs / 300)}
                opacity={0.55} />
            );
          })}
          {nodes.map((n) => {
            const p = pos.get(n.id);
            if (!p) return null;
            return (
              <g key={n.id} transform={`translate(${p.x},${p.y})`}
                style={{ cursor: 'pointer' }}
                onClick={() => setInspector({ title: n.name, body: { id: n.id, kind: n.kind, available: n.available } })}
              >
                <circle r={16} fill={n.available ? 'var(--bg-panel-solid)' : 'var(--danger)'} stroke="var(--accent)" strokeWidth={2} />
                <text textAnchor="middle" dy={4} fontSize={9} fill="var(--fg-primary)">{n.kind.slice(0, 3).toUpperCase()}</text>
                <text textAnchor="middle" dy={30} fontSize={11} fill="var(--fg-secondary)">{n.name}</text>
              </g>
            );
          })}
        </g>
      </svg>
    </div>
  );
}
