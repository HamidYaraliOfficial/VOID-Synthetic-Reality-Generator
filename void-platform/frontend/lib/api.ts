// Thin fetch wrapper around the VOID API Server (see backend/internal/api).
// Every call attaches the stored bearer token (if any) and throws a typed
// ApiError on non-2xx responses so callers can surface real backend errors
// instead of guessing.

import type { BehaviorGraph, BusinessHours, EntitySchema, MetricsSnapshot, SchedulerStatus, Template, Universe } from './types';

const BASE = process.env.NEXT_PUBLIC_VOID_API_BASE || 'http://localhost:8080';

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

let authToken: string | null = null;
export function setAuthToken(token: string | null) {
  authToken = token;
  try {
    if (token) window.localStorage?.setItem('void.token', token);
    else window.localStorage?.removeItem('void.token');
  } catch {
    /* ignore */
  }
}
export function loadStoredToken(): string | null {
  try {
    authToken = window.localStorage?.getItem('void.token') ?? null;
  } catch {
    authToken = null;
  }
  return authToken;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json', ...(init?.headers as Record<string, string>) };
  if (authToken) headers['Authorization'] = `Bearer ${authToken}`;
  const res = await fetch(`${BASE}${path}`, { ...init, headers });
  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = await res.json();
      message = body.error || message;
    } catch {
      /* non-JSON error body */
    }
    throw new ApiError(res.status, message);
  }
  if (res.status === 204) return undefined as T;
  const contentType = res.headers.get('content-type') || '';
  if (contentType.includes('application/json')) return res.json();
  return (await res.text()) as unknown as T;
}

export const api = {
  health: () => request<{ status: string }>('/api/health'),

  login: (username: string, role: string) =>
    request<{ token: string; role: string }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, role }),
    }),

  listUniverses: () => request<Universe[]>('/api/universes'),
  createUniverse: (id: string, name: string, seed: number) =>
    request<Universe>('/api/universes', { method: 'POST', body: JSON.stringify({ id, name, seed }) }),
  getUniverse: (id: string) => request<Universe>(`/api/universes/${id}`),

  addSchema: (universeId: string, schema: EntitySchema) =>
    request<EntitySchema>(`/api/universes/${universeId}/schemas`, { method: 'POST', body: JSON.stringify(schema) }),
  listSchemas: (universeId: string) => request<EntitySchema[]>(`/api/universes/${universeId}/schemas`),

  addBehavior: (universeId: string, graph: BehaviorGraph) =>
    request<BehaviorGraph>(`/api/universes/${universeId}/behaviors`, { method: 'POST', body: JSON.stringify(graph) }),

  spawn: (universeId: string, type: string, count: number) =>
    request<{ spawned: number; type: string }>(`/api/universes/${universeId}/entities/spawn`, {
      method: 'POST',
      body: JSON.stringify({ type, count }),
    }),
  listEntities: (universeId: string, type?: string, limit = 100) =>
    request<unknown>(`/api/universes/${universeId}/entities${type ? `?type=${encodeURIComponent(type)}&limit=${limit}` : ''}`),

  runScenario: (universeId: string, scenario: Record<string, unknown>) =>
    request<{ status: string }>(`/api/universes/${universeId}/scenario/run`, { method: 'POST', body: JSON.stringify(scenario) }),
  pause: (universeId: string) => request(`/api/universes/${universeId}/scenario/pause`, { method: 'POST' }),
  resume: (universeId: string) => request(`/api/universes/${universeId}/scenario/resume`, { method: 'POST' }),
  stop: (universeId: string) => request(`/api/universes/${universeId}/scenario/stop`, { method: 'POST' }),

  snapshot: (universeId: string, label: string) =>
    request(`/api/universes/${universeId}/snapshot`, { method: 'POST', body: JSON.stringify({ label }) }),

  metrics: (universeId: string) => request<MetricsSnapshot>(`/api/universes/${universeId}/metrics`),
  console: (universeId: string) => request<string[]>(`/api/universes/${universeId}/console`),

  getHours: () => request<BusinessHours>('/api/scheduler/hours'),
  setHours: (hours: BusinessHours) => request<BusinessHours>('/api/scheduler/hours', { method: 'POST', body: JSON.stringify(hours) }),
  schedulerStatus: () => request<SchedulerStatus>('/api/scheduler/status'),

  aiGenerate: (prompt: string) => request<unknown>('/api/ai/generate', { method: 'POST', body: JSON.stringify({ prompt }) }),

  templates: () => request<Template[]>('/api/templates'),
  plugins: () => request<Record<string, string[]>>('/api/plugins'),
};

export function metricsSocketUrl(universeId: string): string {
  const wsBase = BASE.replace(/^http/, 'ws');
  const token = authToken ? `&token=${encodeURIComponent(authToken)}` : '';
  return `${wsBase}/api/ws/metrics?universeId=${encodeURIComponent(universeId)}${token}`;
}
