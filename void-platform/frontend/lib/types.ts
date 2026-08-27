// Shared TypeScript types mirroring the Go backend's JSON shapes
// (internal/entity, internal/scenario, internal/scheduler, ...). Kept
// hand-written and dependency-free so the frontend has zero build-time
// coupling to the backend beyond the REST/WS contract itself.

export type FieldType =
  | 'string' | 'integer' | 'float' | 'boolean' | 'datetime'
  | 'uuid' | 'enum' | 'array' | 'json' | 'binary' | 'custom';

export type GeneratorKind =
  | 'random' | 'sequential' | 'weighted_random' | 'uuid' | 'name' | 'email'
  | 'phone' | 'address' | 'date' | 'time' | 'number' | 'pattern'
  | 'distribution' | 'dependent' | 'derived' | 'custom_function';

export interface EntityField {
  name: string;
  type: FieldType;
  generator: GeneratorKind;
  required?: boolean;
  unique?: boolean;
  enumValues?: string[];
  params?: Record<string, unknown>;
}

export interface EntitySchema {
  name: string;
  description?: string;
  fields: EntityField[];
  states?: string[];
  initialState?: string;
}

export type NodeKind =
  | 'event' | 'condition' | 'probability' | 'action' | 'delay'
  | 'state_change' | 'api_call' | 'db_operation' | 'loop';

export interface BehaviorNode {
  id: string;
  kind: NodeKind;
  params?: Record<string, unknown>;
  next?: string[];
  onFailure?: string[];
}

export interface BehaviorGraph {
  name: string;
  entry: string;
  nodes: Record<string, BehaviorNode>;
}

export interface Universe {
  id: string;
  name: string;
  seed: number;
  status: 'idle' | 'running' | 'paused' | 'stopped';
  entityCounts: Record<string, number>;
}

export interface MetricsSnapshot {
  counters: Record<string, number>;
  gauges: Record<string, number>;
  goroutineCount: number;
  memoryAllocMB: number;
  uptimeSeconds: number;
  timestamp: string;
}

export interface Window {
  start: string;
  end: string;
}

export interface BusinessHours {
  timezone: string;
  days: Record<string, Window[]>;
}

export interface SchedulerStatus {
  now: string;
  isOpen: boolean;
  nextChangeAt: string;
  timeUntilNext: number;
  timeUntilNextHuman: string;
  currentWindow?: Window;
}

export interface Template {
  id: string;
  name: string;
  domain: string;
  description: string;
}
