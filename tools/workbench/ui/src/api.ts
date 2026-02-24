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
export interface GlobalStatus {
  connectors_count: number;
  connectors: ConnectorSummary[];
  debug: {
    total_entries: number;
    total_http_requests: number;
    total_plugin_calls: number;
    error_count: number;
  };
}

export interface ConnectorSummary {
  id: string;
  provider: string;
  name: string;
  connector_id: string;
  installed: boolean;
  created_at: string;
  accounts_count?: number;
  payments_count?: number;
  balances_count?: number;
}

export interface ConnectorStatus {
  id: string;
  provider: string;
  name: string;
  connector_id: string;
  installed: boolean;
  created_at: string;
  storage?: {
    accounts_count: number;
    payments_count: number;
    balances_count: number;
    external_accounts_count: number;
    last_updated: string;
  };
  fetch_status?: {
    accounts: { has_more: boolean; pages_fetched: number; total_items: number };
    payments: Record<string, { has_more: boolean; pages_fetched: number; total_items: number }>;
    balances: Record<string, { has_more: boolean; pages_fetched: number; total_items: number }>;
  };
}

export interface AvailableConnector {
  provider: string;
  config: Record<string, ConfigParameter>;
  plugin_type: number; // 0=PSP, 1=OpenBanking, 2=Both
}

export interface ConfigParameter {
  dataType: string;
  required: boolean;
  defaultValue?: string;
}

export interface CreateConnectorRequest {
  provider: string;
  name?: string;
  config: Record<string, unknown>;
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

// Open Banking types
export interface OBConnection {
  connection_id: string;
  connector_id: unknown;
  psu_id: string;
  access_token?: { token: string; expiresAt: string };
  metadata?: Record<string, string>;
  created_at: string;
}

export interface OBCreateUserResponse {
  psu_id: string;
  psp_user_id?: string;
  permanent_token?: { token: string; expiresAt: string };
  metadata?: Record<string, string>;
}

export interface OBCompleteUserLinkResponse {
  connections?: OBConnection[];
  count?: number;
  error?: string;
}

// Generic server types
export interface GenericServerStatus {
  enabled: boolean;
  connector_id: string;
  connector_provider?: string;
  connector_installed?: boolean;
  has_api_key: boolean;
  endpoint: string;
}

// Global API functions (no connector context needed)
export const globalApi = {
  // Global status
  getStatus: () => fetchJSON<GlobalStatus>('/status'),
  
  // Available connectors
  getAvailableConnectors: () => 
    fetchJSON<{ connectors: AvailableConnector[]; count: number }>('/connectors/available'),
  
  // Connector instances management
  listConnectors: () =>
    fetchJSON<{ connectors: ConnectorSummary[]; count: number }>('/connectors'),
  
  createConnector: (req: CreateConnectorRequest) =>
    fetchJSON<ConnectorSummary>('/connectors', {
      method: 'POST',
      body: JSON.stringify(req),
    }),
  
  // Generic server
  getGenericServerStatus: () =>
    fetchJSON<GenericServerStatus>('/generic-server/status'),
  
  setGenericServerConnector: (connectorId: string, apiKey?: string) =>
    fetchJSON<GenericServerStatus>('/generic-server/connector', {
      method: 'POST',
      body: JSON.stringify({ connector_id: connectorId, api_key: apiKey || '' }),
    }),
  
  // Debug (global)
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
};

// Connector-specific API functions
export function connectorApi(connectorId: string) {
  const prefix = `/connectors/${connectorId}`;
  
  return {
    // Status
    getStatus: () => fetchJSON<ConnectorStatus>(`${prefix}/`),
    
    // Lifecycle
    install: () => fetchJSON<{ status: string }>(`${prefix}/install`, { method: 'POST' }),
    uninstall: () => fetchJSON<{ status: string }>(`${prefix}/uninstall`, { method: 'POST' }),
    delete: () => fetchJSON<{ status: string }>(`${prefix}/`, { method: 'DELETE' }),
    reset: () => fetchJSON<{ status: string }>(`${prefix}/reset`, { method: 'POST' }),
    
    // Open Banking operations
    obCreateUser: () =>
      fetchJSON<OBCreateUserResponse>(`${prefix}/ob/create-user`, { method: 'POST' }),

    obCreateUserLink: (psuId: string) =>
      fetchJSON<{ link: string; temporary_token?: unknown }>(`${prefix}/ob/create-link`, {
        method: 'POST',
        body: JSON.stringify({ psu_id: psuId }),
      }),

    obCompleteUserLink: (psuId: string, queryValues: Record<string, string[]>, headers?: Record<string, string[]>, body?: string) =>
      fetchJSON<OBCompleteUserLinkResponse>(`${prefix}/ob/complete-link`, {
        method: 'POST',
        body: JSON.stringify({ psu_id: psuId, query_values: queryValues, headers, body }),
      }),

    obListConnections: () =>
      fetchJSON<{ connections: OBConnection[]; count: number }>(`${prefix}/ob/connections`),

    obDeleteConnection: (connectionId: string) =>
      fetchJSON<{ status: string }>(`${prefix}/ob/connections/${connectionId}`, { method: 'DELETE' }),

    // Fetch operations (with OB connection_id support)
    fetchAccounts: (fromPayload?: unknown, connectionId?: string, innerPayload?: unknown) =>
      fetchJSON<{ accounts: Account[]; has_more: boolean; count: number }>(
        `${prefix}/fetch/accounts`,
        { method: 'POST', body: JSON.stringify({ from_payload: fromPayload, connection_id: connectionId, inner_payload: innerPayload }) }
      ),
    
    fetchPayments: (fromPayload?: unknown, connectionId?: string, innerPayload?: unknown) =>
      fetchJSON<{ payments: Payment[]; has_more: boolean; count: number }>(
        `${prefix}/fetch/payments`,
        { method: 'POST', body: JSON.stringify({ from_payload: fromPayload, connection_id: connectionId, inner_payload: innerPayload }) }
      ),

    fetchBalances: (fromPayload?: unknown, connectionId?: string, innerPayload?: unknown) =>
      fetchJSON<{ balances: Balance[]; has_more: boolean; count: number }>(
        `${prefix}/fetch/balances`,
        { method: 'POST', body: JSON.stringify({ from_payload: fromPayload, connection_id: connectionId, inner_payload: innerPayload }) }
      ),
    
    fetchExternalAccounts: (fromPayload?: unknown) =>
      fetchJSON<{ external_accounts: Account[]; has_more: boolean; count: number }>(
        `${prefix}/fetch/external-accounts`,
        { method: 'POST', body: JSON.stringify({ from_payload: fromPayload }) }
      ),
    
    fetchAll: () =>
      fetchJSON<{ status: string; accounts: number; payments: number; balances: number }>(
        `${prefix}/fetch/all`,
        { method: 'POST' }
      ),
    
    // Data
    getAccounts: () => 
      fetchJSON<{ accounts: Account[]; count: number }>(`${prefix}/data/accounts`),
    
    getPayments: () =>
      fetchJSON<{ payments: Payment[]; count: number }>(`${prefix}/data/payments`),
    
    getBalances: (account?: string) =>
      fetchJSON<{ balances: Balance[] }>(`${prefix}/data/balances${account ? `?account=${account}` : ''}`),
    
    getExternalAccounts: () =>
      fetchJSON<{ external_accounts: Account[]; count: number }>(`${prefix}/data/external-accounts`),
    
    getStates: () =>
      fetchJSON<{ states: Record<string, unknown> }>(`${prefix}/data/states`),
    
    getTasksTree: () =>
      fetchJSON<{ tasks_tree: unknown }>(`${prefix}/data/tasks-tree`),
    
    // Config
    setPageSize: (pageSize: number) =>
      fetchJSON<{ page_size: number }>(`${prefix}/config/page-size`, {
        method: 'PUT',
        body: JSON.stringify({ page_size: pageSize }),
      }),
    
    // Export
    exportData: () => fetchJSON<unknown>(`${prefix}/data/export`),
    
    // Introspection
    getConnectorInfo: () => fetchJSON<ConnectorInfo>(`${prefix}/introspect/info`),
    
    getFileTree: () => fetchJSON<{ files: FileNode[] }>(`${prefix}/introspect/files`),
    
    getFile: (path: string) => fetchJSON<SourceFile>(`${prefix}/introspect/file?path=${encodeURIComponent(path)}`),
    
    searchCode: (query: string) => fetchJSON<{ query: string; results: SearchResult[]; count: number }>(
      `${prefix}/introspect/search?q=${encodeURIComponent(query)}`
    ),
    
    // Tasks
    getTasks: () => fetchJSON<TaskTreeSummary>(`${prefix}/tasks`),
    
    getTaskExecutions: (limit = 50) => 
      fetchJSON<{ executions: TaskExecution[]; count: number }>(`${prefix}/tasks/executions?limit=${limit}`),
    
    stepTask: () => fetchJSON<{ status: string }>(`${prefix}/tasks/step`, { method: 'POST' }),
    
    setStepMode: (enabled: boolean) => 
      fetchJSON<{ step_mode: boolean }>(`${prefix}/tasks/step-mode`, {
        method: 'PUT',
        body: JSON.stringify({ enabled }),
      }),
    
    resetTasks: () => fetchJSON<{ status: string }>(`${prefix}/tasks/reset`, { method: 'POST' }),
    
    // Snapshots
    listSnapshots: (operation?: string) => {
      const params = operation ? `?operation=${encodeURIComponent(operation)}` : '';
      return fetchJSON<{ snapshots: Snapshot[]; count: number }>(`${prefix}/snapshots${params}`);
    },
    
    getSnapshotStats: () => fetchJSON<SnapshotStats>(`${prefix}/snapshots/stats`),
    
    getSnapshot: (id: string) => fetchJSON<Snapshot>(`${prefix}/snapshots/${id}`),
    
    createSnapshotFromCapture: (captureId: string, data: { name: string; operation: string; description?: string; tags?: string[] }) =>
      fetchJSON<Snapshot>(`${prefix}/snapshots/from-capture/${captureId}`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    
    deleteSnapshot: (id: string) => fetchJSON<{ status: string }>(`${prefix}/snapshots/${id}`, { method: 'DELETE' }),
    
    clearSnapshots: () => fetchJSON<{ status: string }>(`${prefix}/snapshots`, { method: 'DELETE' }),
    
    // Test Generation
    previewTests: () => fetchJSON<GenerateResult>(`${prefix}/tests/preview`),
    
    generateTests: (outputDir?: string) =>
      fetchJSON<GenerateResult>(`${prefix}/tests/generate`, {
        method: 'POST',
        body: JSON.stringify({ output_dir: outputDir }),
      }),
    
    // Schema inference
    listSchemas: () => fetchJSON<{ schemas: InferredSchema[]; count: number }>(`${prefix}/schemas`),
    
    getSchema: (operation: string) => fetchJSON<InferredSchema>(`${prefix}/schemas/${encodeURIComponent(operation)}`),
    
    saveAllSchemaBaselines: () => fetchJSON<{ status: string; count: number }>(`${prefix}/schemas/baselines/all`, { method: 'POST' }),
    
    listSchemaBaselines: () => fetchJSON<{ baselines: InferredSchema[]; count: number }>(`${prefix}/schemas/baselines`),
    
    compareSchema: (operation: string) => fetchJSON<SchemaDiff>(`${prefix}/schemas/compare/${encodeURIComponent(operation)}`),
    
    // Data baselines
    listBaselines: () => fetchJSON<{ baselines: DataBaseline[]; count: number }>(`${prefix}/baselines`),
    
    saveBaseline: (name?: string) =>
      fetchJSON<DataBaseline>(`${prefix}/baselines`, {
        method: 'POST',
        body: JSON.stringify({ name }),
      }),
    
    deleteBaseline: (id: string) => fetchJSON<{ status: string }>(`${prefix}/baselines/${id}`, { method: 'DELETE' }),
    
    compareBaseline: (id: string) => fetchJSON<BaselineDiff>(`${prefix}/baselines/${id}/compare`),
  };
}

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
  from_payload?: unknown;
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

// For backwards compatibility, export a default api that uses a selected connector
// This will be used when a connector is selected in the UI
export let selectedConnectorId: string | null = null;

export function setSelectedConnector(id: string | null) {
  selectedConnectorId = id;
}

// Legacy api export - uses selected connector
export const api = {
  // Global operations that don't need connector context
  getStatus: globalApi.getStatus,
  getAvailableConnectors: globalApi.getAvailableConnectors,
  listConnectors: globalApi.listConnectors,
  createConnector: globalApi.createConnector,
  getDebugLogs: globalApi.getDebugLogs,
  getPluginCalls: globalApi.getPluginCalls,
  getHTTPRequests: globalApi.getHTTPRequests,
  getHTTPCaptureStatus: globalApi.getHTTPCaptureStatus,
  enableHTTPCapture: globalApi.enableHTTPCapture,
  disableHTTPCapture: globalApi.disableHTTPCapture,
  clearDebug: globalApi.clearDebug,
  getGenericServerStatus: globalApi.getGenericServerStatus,
  setGenericServerConnector: globalApi.setGenericServerConnector,
  
  // Connector-specific operations (require selectedConnectorId)
  getConnectorStatus: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getStatus();
  },
  install: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).install();
  },
  uninstall: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).uninstall();
  },
  deleteConnector: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).delete();
  },
  reset: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).reset();
  },
  fetchAccounts: (fromPayload?: unknown) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).fetchAccounts(fromPayload);
  },
  fetchPayments: (fromPayload?: unknown) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).fetchPayments(fromPayload);
  },
  fetchBalances: (fromPayload?: unknown) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).fetchBalances(fromPayload);
  },
  fetchExternalAccounts: (fromPayload?: unknown) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).fetchExternalAccounts(fromPayload);
  },
  fetchAll: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).fetchAll();
  },
  getAccounts: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getAccounts();
  },
  getPayments: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getPayments();
  },
  getBalances: (account?: string) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getBalances(account);
  },
  getExternalAccounts: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getExternalAccounts();
  },
  getStates: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getStates();
  },
  getTasksTree: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getTasksTree();
  },
  setPageSize: (pageSize: number) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).setPageSize(pageSize);
  },
  exportData: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).exportData();
  },
  getConnectorInfo: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getConnectorInfo();
  },
  getFileTree: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getFileTree();
  },
  getFile: (path: string) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getFile(path);
  },
  searchCode: (query: string) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).searchCode(query);
  },
  getTasks: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getTasks();
  },
  getTaskExecutions: (limit = 50) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getTaskExecutions(limit);
  },
  stepTask: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).stepTask();
  },
  setStepMode: (enabled: boolean) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).setStepMode(enabled);
  },
  resetTasks: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).resetTasks();
  },
  listSnapshots: (operation?: string) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).listSnapshots(operation);
  },
  getSnapshotStats: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getSnapshotStats();
  },
  getSnapshot: (id: string) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getSnapshot(id);
  },
  createSnapshotFromCapture: (captureId: string, data: { name: string; operation: string; description?: string; tags?: string[] }) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).createSnapshotFromCapture(captureId, data);
  },
  deleteSnapshot: (id: string) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).deleteSnapshot(id);
  },
  clearSnapshots: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).clearSnapshots();
  },
  previewTests: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).previewTests();
  },
  generateTests: (outputDir?: string) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).generateTests(outputDir);
  },
  listSchemas: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).listSchemas();
  },
  getSchema: (operation: string) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).getSchema(operation);
  },
  saveAllSchemaBaselines: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).saveAllSchemaBaselines();
  },
  listSchemaBaselines: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).listSchemaBaselines();
  },
  compareSchema: (operation: string) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).compareSchema(operation);
  },
  listBaselines: () => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).listBaselines();
  },
  saveBaseline: (name?: string) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).saveBaseline(name);
  },
  deleteBaseline: (id: string) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).deleteBaseline(id);
  },
  compareBaseline: (id: string) => {
    if (!selectedConnectorId) throw new Error('No connector selected');
    return connectorApi(selectedConnectorId).compareBaseline(id);
  },
};

export default api;
