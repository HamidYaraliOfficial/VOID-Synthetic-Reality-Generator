'use client';

// Central app state (zustand): the active Universe, its live entity counts,
// schemas, behaviors, console tail and run status. Kept intentionally flat —
// components read only the slices they need via selectors.

import { create } from 'zustand';
import type { EntitySchema, MetricsSnapshot, Universe } from './types';

export type PanelId = 'canvas' | 'entities' | 'behaviors' | 'dashboard' | 'scheduler';

interface AppState {
  activePanel: PanelId;
  setActivePanel: (p: PanelId) => void;

  universe: Universe | null;
  setUniverse: (u: Universe | null) => void;

  schemas: EntitySchema[];
  setSchemas: (s: EntitySchema[]) => void;
  upsertSchema: (s: EntitySchema) => void;

  metrics: MetricsSnapshot | null;
  setMetrics: (m: MetricsSnapshot) => void;

  consoleLines: string[];
  setConsoleLines: (lines: string[]) => void;

  selectedEntityType: string | null;
  setSelectedEntityType: (t: string | null) => void;

  inspectorContent: { title: string; body: Record<string, unknown> } | null;
  setInspectorContent: (c: { title: string; body: Record<string, unknown> } | null) => void;

  commandPaletteOpen: boolean;
  setCommandPaletteOpen: (v: boolean) => void;

  authed: boolean;
  setAuthed: (v: boolean) => void;
}

export const useAppStore = create<AppState>((set) => ({
  activePanel: 'canvas',
  setActivePanel: (p) => set({ activePanel: p }),

  universe: null,
  setUniverse: (u) => set({ universe: u }),

  schemas: [],
  setSchemas: (s) => set({ schemas: s }),
  upsertSchema: (s) =>
    set((state) => ({
      schemas: [...state.schemas.filter((sc) => sc.name !== s.name), s],
    })),

  metrics: null,
  setMetrics: (m) => set({ metrics: m }),

  consoleLines: [],
  setConsoleLines: (lines) => set({ consoleLines: lines }),

  selectedEntityType: null,
  setSelectedEntityType: (t) => set({ selectedEntityType: t }),

  inspectorContent: null,
  setInspectorContent: (c) => set({ inspectorContent: c }),

  commandPaletteOpen: false,
  setCommandPaletteOpen: (v) => set({ commandPaletteOpen: v }),

  authed: false,
  setAuthed: (v) => set({ authed: v }),
}));
