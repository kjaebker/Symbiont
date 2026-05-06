/**
 * Centralised TanStack Query keys.
 *
 * Use `qk` everywhere instead of inline string-array literals so:
 *   - Renaming a key is a single edit, type-checked everywhere.
 *   - Hierarchical invalidation works as designed: invalidating
 *     `qk.outlets.all` also invalidates `qk.outlets.configs`,
 *     `qk.outlets.history(...)`, etc.
 *
 * The shape rule is: each resource gets a namespace whose first array
 * element is the resource name. Sub-resources extend it. Parameterised
 * keys are functions; static keys are tuples typed `as const` so consumers
 * get readonly tuple types back.
 */
export const qk = {
  probes: {
    all: ['probes'] as const,
    history: (name: string, params?: unknown) =>
      ['probes', 'history', name, params] as const,
    sparkline: (name: string | null) =>
      ['probes', 'sparkline', name] as const,
    configs: ['probes', 'configs'] as const,
  },
  outlets: {
    all: ['outlets'] as const,
    history: (id: string, params?: unknown) =>
      ['outlets', 'history', id, params] as const,
    intensityHistory: (id: string, params?: unknown) =>
      ['outlets', 'intensity-history', id, params] as const,
    configs: ['outlets', 'configs'] as const,
  },
  programs: {
    all: ['programs'] as const,
  },
  feed: {
    current: ['feed'] as const,
  },
  system: {
    status: ['system'] as const,
    log: (params?: unknown) => ['system', 'log', params] as const,
  },
  alerts: {
    all: ['alerts'] as const,
    events: (params?: unknown) => ['alerts', 'events', params] as const,
  },
  tokens: {
    all: ['tokens'] as const,
  },
  notifications: {
    targets: ['notifications', 'targets'] as const,
  },
  backups: {
    all: ['backups'] as const,
    config: ['backups', 'config'] as const,
  },
  devices: {
    all: ['devices'] as const,
    suggestions: ['devices', 'suggestions'] as const,
  },
  dashboard: {
    layout: ['dashboard', 'layout'] as const,
  },
  measurements: {
    list: (params?: unknown) => ['measurements', 'list', params] as const,
    parameters: ['measurements', 'parameters'] as const,
    kits: ['measurements', 'kits'] as const,
  },
  livestock: {
    all: ['livestock'] as const,
    list: (params?: unknown) => ['livestock', 'list', params] as const,
    item: (id: number) => ['livestock', 'item', id] as const,
    species: ['livestock', 'species'] as const,
    observations: (livestockId: number) =>
      ['livestock', 'observations', livestockId] as const,
  },
  tank: {
    profile: ['tank', 'profile'] as const,
  },
  journal: {
    list: (params?: unknown) => ['journal', 'list', params] as const,
    templates: ['journal', 'templates'] as const,
  },
  events: {
    audit: (params?: unknown) => ['events', 'audit', params] as const,
    busStats: ['events', 'bus-stats'] as const,
  },
  agent: {
    settings: ['agent', 'settings'] as const,
    context: ['agent', 'context'] as const,
    skills: ['agent', 'skills'] as const,
  },
  dosing: {
    all: ['dosing'] as const,
    products: ['dosing', 'products'] as const,
    schedules: ['dosing', 'schedules'] as const,
    logs: (params?: unknown) => ['dosing', 'logs', params] as const,
  },
  maintenance: {
    all: ['maintenance'] as const,
    tasks: ['maintenance', 'tasks'] as const,
    logs: (taskId: number, limit?: number) =>
      ['maintenance', 'logs', taskId, limit] as const,
  },
  /** Aggregated due-tasks endpoint — fed by both dosing and maintenance. */
  due: {
    all: ['due-items'] as const,
  },
  dailyPrompt: {
    current: ['daily-prompt'] as const,
  },
} as const
