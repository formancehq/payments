const API_BASE = '/api';

async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(API_BASE + url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(error.error || 'Request failed');
  }
  return res.json();
}

// Types
export interface Status {
  connector_id: string;
  provider: string;
  storage: {
    accounts_count: number;
    payments_count: number;
    balances_count: number;
    external_accounts_count: number;
    last_updated: string;
  };
  debug: {
    total_entries: number;
    total_http_requests: number;
    total_plugin_calls: number;
    error_count: number;
  };
  fetch_status: {
    accounts: { has_more: boolean; pages_fetched: number; total_items: number };
    payments: Record<string, { has_more: boolean; pages_fetched: number; total_items: number }>;
    balances: Record<string, { has_more: boolean; pages_fetched: number; total_items: number }>;
  };
}

export interface Account {
  reference: string;
  created_at: string;
  name?: string;
  default_asset?: string;
  metadata?: Record<string, string>;
  raw?: unknown;
}

export interface Payment {
  reference: string;
  created_at: string;
  type: string;
  status: string;
  amount: number;
  asset: string;
  scheme?: string;
  source_account_reference?: string;
  destination_account_reference?: string;
  metadata?: Record<string, string>;
  raw?: unknown;
}

export interface Balance {
  account_reference: string;
  asset: string;
  amount: number;
  created_at: string;
}

export interface DebugEntry {
  id: string;
  timestamp: string;
  type: 'log' | 'error' | 'plugin_call' | 'plugin_result' | 'state_change' | 'request';
  operation?: string;
  message?: string;
  data?: unknown;
  duration?: number;
  error?: string;
}

export interface PluginCall {
  id: string;
  timestamp: string;
  method: string;
  input: unknown;
  output?: unknown;
  duration: number;
  error?: string;
}

export interface HTTPRequest {
  id: string;
  timestamp: string;
  method: string;
  url: string;
  request_headers?: Record<string, string>;
  request_body?: string;
  response_status?: number;
  response_headers?: Record<string, string>;
  response_body?: string;
  duration: number;
  error?: string;
}

// API functions
export const api = {
  // Status
  getStatus: () => fetchJSON<Status>('/status'),
  
  // Fetch operations
  fetchAccounts: (fromPayload?: unknown) => 
    fetchJSON<{ accounts: Account[]; has_more: boolean; count: number }>(
      '/fetch/accounts', 
      { method: 'POST', body: JSON.stringify({ from_payload: fromPayload }) }
    ),
  
  fetchPayments: (fromPayload?: unknown) =>
    fetchJSON<{ payments: Payment[]; has_more: boolean; count: number }>(
      '/fetch/payments',
      { method: 'POST', body: JSON.stringify({ from_payload: fromPayload }) }
    ),
  
  fetchBalances: (fromPayload?: unknown) =>
    fetchJSON<{ balances: Balance[]; has_more: boolean; count: number }>(
      '/fetch/balances',
      { method: 'POST', body: JSON.stringify({ from_payload: fromPayload }) }
    ),
  
  fetchAll: () =>
    fetchJSON<{ status: string; accounts: number; payments: number; balances: number }>(
      '/fetch/all',
      { method: 'POST' }
    ),
  
  // Data
  getAccounts: () => 
    fetchJSON<{ accounts: Account[]; count: number }>('/data/accounts'),
  
  getPayments: () =>
    fetchJSON<{ payments: Payment[]; count: number }>('/data/payments'),
  
  getBalances: (account?: string) =>
    fetchJSON<{ balances: Balance[] }>(`/data/balances${account ? `?account=${account}` : ''}`),
  
  getExternalAccounts: () =>
    fetchJSON<{ external_accounts: Account[]; count: number }>('/data/external-accounts'),
  
  getStates: () =>
    fetchJSON<{ states: Record<string, unknown> }>('/data/states'),
  
  getTasksTree: () =>
    fetchJSON<{ tasks_tree: unknown }>('/data/tasks-tree'),
  
  // Debug
  getDebugLogs: (limit = 100, type?: string) =>
    fetchJSON<{ entries: DebugEntry[]; count: number }>(
      `/debug/logs?limit=${limit}${type ? `&type=${type}` : ''}`
    ),
  
  getPluginCalls: (limit = 50) =>
    fetchJSON<{ plugin_calls: PluginCall[]; count: number }>(`/debug/plugin-calls?limit=${limit}`),
  
  getHTTPRequests: (limit = 50) =>
    fetchJSON<{ requests: HTTPRequest[]; count: number }>(`/debug/requests?limit=${limit}`),
  
  getHTTPCaptureStatus: () =>
    fetchJSON<{ enabled: boolean; max_body_size: number }>('/debug/http-capture'),
  
  enableHTTPCapture: () =>
    fetchJSON<{ status: string }>('/debug/http-capture/enable', { method: 'POST' }),
  
  disableHTTPCapture: () =>
    fetchJSON<{ status: string }>('/debug/http-capture/disable', { method: 'POST' }),
  
  clearDebug: () =>
    fetchJSON<{ status: string }>('/debug/clear', { method: 'DELETE' }),
  
  // Control
  install: () =>
    fetchJSON<{ status: string }>('/install', { method: 'POST' }),
  
  uninstall: () =>
    fetchJSON<{ status: string }>('/uninstall', { method: 'POST' }),
  
  reset: () =>
    fetchJSON<{ status: string }>('/reset', { method: 'POST' }),
  
  setPageSize: (pageSize: number) =>
    fetchJSON<{ page_size: number }>('/config/page-size', {
      method: 'PUT',
      body: JSON.stringify({ page_size: pageSize }),
    }),
  
  // Export
  exportData: () => fetchJSON<unknown>('/data/export'),
  
  // Introspection
  getConnectorInfo: () => fetchJSON<ConnectorInfo>('/introspect/info'),
  
  getFileTree: () => fetchJSON<{ files: FileNode[] }>('/introspect/files'),
  
  getFile: (path: string) => fetchJSON<SourceFile>(`/introspect/file?path=${encodeURIComponent(path)}`),
  
  searchCode: (query: string) => fetchJSON<{ query: string; results: SearchResult[]; count: number }>(
    `/introspect/search?q=${encodeURIComponent(query)}`
  ),
  
  // Tasks
  getTasks: () => fetchJSON<TaskTreeSummary>('/tasks'),
  
  getTaskExecutions: (limit = 50) => 
    fetchJSON<{ executions: TaskExecution[]; count: number }>(`/tasks/executions?limit=${limit}`),
  
  stepTask: () => fetchJSON<{ status: string }>('/tasks/step', { method: 'POST' }),
  
  setStepMode: (enabled: boolean) => 
    fetchJSON<{ step_mode: boolean }>('/tasks/step-mode', {
      method: 'PUT',
      body: JSON.stringify({ enabled }),
    }),
  
  resetTasks: () => fetchJSON<{ status: string }>('/tasks/reset', { method: 'POST' }),
  
  // Snapshots
  listSnapshots: (operation?: string) => {
    const params = operation ? `?operation=${encodeURIComponent(operation)}` : '';
    return fetchJSON<{ snapshots: Snapshot[]; count: number }>(`/snapshots${params}`);
  },
  
  getSnapshotStats: () => fetchJSON<SnapshotStats>('/snapshots/stats'),
  
  getSnapshot: (id: string) => fetchJSON<Snapshot>(`/snapshots/${id}`),
  
  createSnapshotFromCapture: (captureId: string, data: { name: string; operation: string; description?: string; tags?: string[] }) =>
    fetchJSON<Snapshot>(`/snapshots/from-capture/${captureId}`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  
  deleteSnapshot: (id: string) => fetchJSON<{ status: string }>(`/snapshots/${id}`, { method: 'DELETE' }),
  
  clearSnapshots: () => fetchJSON<{ status: string }>('/snapshots', { method: 'DELETE' }),
  
  // Test Generation
  previewTests: () => fetchJSON<GenerateResult>('/tests/preview'),
  
  generateTests: (outputDir?: string) =>
    fetchJSON<GenerateResult>('/tests/generate', {
      method: 'POST',
      body: JSON.stringify({ output_dir: outputDir }),
    }),
  
  // Schema inference
  listSchemas: () => fetchJSON<{ schemas: InferredSchema[]; count: number }>('/schemas'),
  
  getSchema: (operation: string) => fetchJSON<InferredSchema>(`/schemas/${encodeURIComponent(operation)}`),
  
  saveAllSchemaBaselines: () => fetchJSON<{ status: string; count: number }>('/schemas/baselines/all', { method: 'POST' }),
  
  listSchemaBaselines: () => fetchJSON<{ baselines: InferredSchema[]; count: number }>('/schemas/baselines'),
  
  compareSchema: (operation: string) => fetchJSON<SchemaDiff>(`/schemas/compare/${encodeURIComponent(operation)}`),
  
  // Data baselines
  listBaselines: () => fetchJSON<{ baselines: DataBaseline[]; count: number }>('/baselines'),
  
  saveBaseline: (name?: string) =>
    fetchJSON<DataBaseline>('/baselines', {
      method: 'POST',
      body: JSON.stringify({ name }),
    }),
  
  deleteBaseline: (id: string) => fetchJSON<{ status: string }>(`/baselines/${id}`, { method: 'DELETE' }),
  
  compareBaseline: (id: string) => fetchJSON<BaselineDiff>(`/baselines/${id}/compare`),
};

// Introspection types
export interface ConnectorInfo {
  provider: string;
  connector_id: string;
  capabilities: CapabilityInfo[];
  config: ConfigParam[];
  source_path?: string;
  has_source: boolean;
  files?: FileNode[];
  methods?: MethodInfo[];
}

export interface CapabilityInfo {
  name: string;
  supported: boolean;
  description?: string;
}

export interface ConfigParam {
  name: string;
  type: string;
  required: boolean;
}

export interface MethodInfo {
  name: string;
  implemented: boolean;
  file?: string;
  line?: number;
}

export interface FileNode {
  name: string;
  path: string;
  size: number;
  is_dir: boolean;
  children?: FileNode[];
}

export interface SourceFile {
  path: string;
  name: string;
  content: string;
  language: string;
  lines: number;
  symbols?: CodeSymbol[];
}

export interface CodeSymbol {
  name: string;
  kind: string;
  line: number;
  doc?: string;
  signature?: string;
}

export interface SearchResult {
  file: string;
  line: number;
  content: string;
}

// Task types
export interface TaskTreeSummary {
  tree: TaskNodeSummary[];
  current_task?: TaskNodeSummary;
  stats: TaskStats;
  step_mode: boolean;
  is_running: boolean;
}

export interface TaskNodeSummary {
  id: string;
  type: string;
  name?: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped';
  duration?: string;
  items_count: number;
  error?: string;
  children?: TaskNodeSummary[];
  child_count: number;
  last_exec_time?: string;
}

export interface TaskStats {
  total_executions: number;
  success_count: number;
  failure_count: number;
  total_items_fetched: number;
  total_duration: number;
  last_execution_at?: string;
}

export interface TaskExecution {
  id: string;
  started_at: string;
  completed_at?: string;
  duration: number;
  status: string;
  error?: string;
  items_count: number;
  page_number: number;
  has_more: boolean;
}

// Snapshot types
export interface Snapshot {
  id: string;
  name: string;
  description?: string;
  created_at: string;
  tags?: string[];
  provider: string;
  operation: string;
  request: SnapshotRequest;
  response: SnapshotResponse;
  capture_id?: string;
}

export interface SnapshotRequest {
  method: string;
  url: string;
  headers?: Record<string, string>;
  body?: string;
}

export interface SnapshotResponse {
  status_code: number;
  status?: string;
  headers?: Record<string, string>;
  body?: string;
}

export interface SnapshotStats {
  total: number;
  by_operation: Record<string, number>;
  by_tag: Record<string, number>;
}

export interface GeneratedTest {
  filename: string;
  content: string;
  package: string;
}

export interface GeneratedFixture {
  filename: string;
  content: string;
}

export interface GenerateResult {
  test_file: GeneratedTest;
  fixtures: GeneratedFixture[];
  instructions: string;
}

// Schema types
export interface InferredSchema {
  id: string;
  name: string;
  operation: string;
  endpoint: string;
  method: string;
  created_at: string;
  updated_at: string;
  sample_count: number;
  root: FieldSchema;
}

export interface FieldSchema {
  name: string;
  path: string;
  type: string;
  types?: string[];
  required: boolean;
  nullable: boolean;
  array_item?: FieldSchema;
  properties?: Record<string, FieldSchema>;
  examples?: unknown[];
  seen_count: number;
}

export interface SchemaDiff {
  timestamp: string;
  baseline_id: string;
  current_id?: string;
  operation: string;
  has_changes: boolean;
  added_fields?: FieldChange[];
  removed_fields?: FieldChange[];
  type_changes?: TypeChange[];
  summary: string;
}

export interface FieldChange {
  path: string;
  type: string;
}

export interface TypeChange {
  path: string;
  old_type: string;
  new_type: string;
}

// Data baseline types
export interface DataBaseline {
  id: string;
  name: string;
  created_at: string;
  provider: string;
  account_count: number;
  payment_count: number;
  balance_count: number;
  external_account_count: number;
}

export interface BaselineDiff {
  timestamp: string;
  baseline_id: string;
  has_changes: boolean;
  accounts: DataDiff;
  payments: DataDiff;
  balances: DataDiff;
  external_accounts: DataDiff;
  summary: string;
}

export interface DataDiff {
  baseline_count: number;
  current_count: number;
  added?: string[];
  removed?: string[];
  modified?: ItemDiff[];
}

export interface ItemDiff {
  reference: string;
  changes: Change[];
}

export interface Change {
  field: string;
  old_value: unknown;
  new_value: unknown;
}

// Replay types
export interface ReplayRequest {
  original_id?: string;
  method: string;
  url: string;
  headers?: Record<string, string>;
  body?: string;
}

export interface ReplayResponse {
  id: string;
  timestamp: string;
  duration: number;
  original_id?: string;
  request: ReplayRequest;
  status_code: number;
  status: string;
  headers?: Record<string, string>;
  body?: string;
  error?: string;
}

export interface ResponseComparison {
  original_id: string;
  replay_id: string;
  status_match: boolean;
  original_status: number;
  replay_status: number;
  body_match: boolean;
  body_diff?: string;
  header_diffs?: HeaderDiff[];
}

export interface HeaderDiff {
  key: string;
  original_value?: string;
  replay_value?: string;
  type: 'added' | 'removed' | 'changed';
}

export default api;
