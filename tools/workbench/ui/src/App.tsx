import { useState, useEffect, useCallback } from 'react';
import api, { Status, Account, Payment, Balance, DebugEntry, PluginCall, HTTPRequest, ConnectorInfo, FileNode, SourceFile, SearchResult, TaskTreeSummary, TaskNodeSummary, TaskExecution, Snapshot, SnapshotStats, GenerateResult, InferredSchema, SchemaDiff, DataBaseline, BaselineDiff } from './api';
import './App.css';

type Tab = 'dashboard' | 'accounts' | 'payments' | 'balances' | 'tasks' | 'debug' | 'snapshots' | 'analysis' | 'code' | 'state';

function App() {
  const [tab, setTab] = useState<Tab>('dashboard');
  const [status, setStatus] = useState<Status | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Data
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [payments, setPayments] = useState<Payment[]>([]);
  const [balances, setBalances] = useState<Balance[]>([]);
  const [debugLogs, setDebugLogs] = useState<DebugEntry[]>([]);
  const [pluginCalls, setPluginCalls] = useState<PluginCall[]>([]);
  const [httpRequests, setHttpRequests] = useState<HTTPRequest[]>([]);
  const [httpCaptureEnabled, setHttpCaptureEnabled] = useState(true);
  const [states, setStates] = useState<Record<string, unknown>>({});
  const [tasksTree, setTasksTree] = useState<unknown>(null);
  
  // Snapshot modal state
  const [showSnapshotModal, setShowSnapshotModal] = useState(false);
  const [snapshotCaptureId, setSnapshotCaptureId] = useState<string | null>(null);

  const refreshStatus = useCallback(async () => {
    try {
      const s = await api.getStatus();
      setStatus(s);
    } catch (e) {
      console.error('Failed to fetch status:', e);
    }
  }, []);

  const refreshData = useCallback(async () => {
    try {
      const [acc, pay, bal, logs, calls, httpReqs, httpStatus, st, tree] = await Promise.all([
        api.getAccounts(),
        api.getPayments(),
        api.getBalances(),
        api.getDebugLogs(100),
        api.getPluginCalls(50),
        api.getHTTPRequests(50),
        api.getHTTPCaptureStatus(),
        api.getStates(),
        api.getTasksTree(),
      ]);
      setAccounts(acc.accounts || []);
      setPayments(pay.payments || []);
      setBalances(bal.balances || []);
      setDebugLogs(logs.entries || []);
      setPluginCalls(calls.plugin_calls || []);
      setHttpRequests(httpReqs.requests || []);
      setHttpCaptureEnabled(httpStatus.enabled);
      setStates(st.states || {});
      setTasksTree(tree.tasks_tree);
    } catch (e) {
      console.error('Failed to refresh data:', e);
    }
  }, []);

  useEffect(() => {
    refreshStatus();
    refreshData();
    const interval = setInterval(() => {
      refreshStatus();
      refreshData();
    }, 2000);
    return () => clearInterval(interval);
  }, [refreshStatus, refreshData]);

  const runAction = async (action: () => Promise<unknown>, successMsg?: string) => {
    setLoading(true);
    setError(null);
    try {
      await action();
      if (successMsg) setError(null);
      await refreshStatus();
      await refreshData();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="app">
      <header className="header">
        <div className="header-left">
          <h1>CONNECTOR WORKBENCH</h1>
          {status && (
            <span className="provider-badge">{status.provider}</span>
          )}
        </div>
        <div className="header-right">
          {status && (
            <code className="connector-id">{status.connector_id}</code>
          )}
        </div>
      </header>

      <nav className="tabs">
        <button className={`tab ${tab === 'dashboard' ? 'active' : ''}`} onClick={() => setTab('dashboard')}>
          Dashboard
        </button>
        <button className={`tab ${tab === 'accounts' ? 'active' : ''}`} onClick={() => setTab('accounts')}>
          Accounts {accounts.length > 0 && <span className="tab-count">{accounts.length}</span>}
        </button>
        <button className={`tab ${tab === 'payments' ? 'active' : ''}`} onClick={() => setTab('payments')}>
          Payments {payments.length > 0 && <span className="tab-count">{payments.length}</span>}
        </button>
        <button className={`tab ${tab === 'balances' ? 'active' : ''}`} onClick={() => setTab('balances')}>
          Balances {balances.length > 0 && <span className="tab-count">{balances.length}</span>}
        </button>
        <button className={`tab ${tab === 'tasks' ? 'active' : ''}`} onClick={() => setTab('tasks')}>
          Tasks
        </button>
        <button className={`tab ${tab === 'debug' ? 'active' : ''}`} onClick={() => setTab('debug')}>
          Debug {status && status.debug.error_count > 0 && (
            <span className="tab-count error">{status.debug.error_count}</span>
          )}
        </button>
        <button className={`tab ${tab === 'snapshots' ? 'active' : ''}`} onClick={() => setTab('snapshots')}>
          Snapshots
        </button>
        <button className={`tab ${tab === 'analysis' ? 'active' : ''}`} onClick={() => setTab('analysis')}>
          Analysis
        </button>
        <button className={`tab ${tab === 'code' ? 'active' : ''}`} onClick={() => setTab('code')}>
          Code
        </button>
        <button className={`tab ${tab === 'state' ? 'active' : ''}`} onClick={() => setTab('state')}>
          State
        </button>
      </nav>

      {error && (
        <div className="error-banner">
          <span>{error}</span>
          <button onClick={() => setError(null)}>×</button>
        </div>
      )}

      <main className="main">
        {tab === 'dashboard' && (
          <DashboardTab
            status={status}
            loading={loading}
            onFetchAccounts={() => runAction(() => api.fetchAccounts())}
            onFetchPayments={() => runAction(() => api.fetchPayments())}
            onFetchBalances={() => runAction(() => api.fetchBalances())}
            onFetchAll={() => runAction(() => api.fetchAll())}
            onReset={() => runAction(() => api.reset())}
          />
        )}
        {tab === 'accounts' && <AccountsTab accounts={accounts} />}
        {tab === 'payments' && <PaymentsTab payments={payments} />}
        {tab === 'balances' && <BalancesTab balances={balances} />}
        {tab === 'tasks' && <TasksTab />}
        {tab === 'debug' && (
          <DebugTab 
            logs={debugLogs} 
            pluginCalls={pluginCalls}
            httpRequests={httpRequests}
            httpCaptureEnabled={httpCaptureEnabled}
            onClear={() => runAction(() => api.clearDebug())}
            onToggleCapture={() => runAction(async () => {
              if (httpCaptureEnabled) {
                await api.disableHTTPCapture();
              } else {
                await api.enableHTTPCapture();
              }
            })}
            onSaveSnapshot={(captureId) => {
              setSnapshotCaptureId(captureId);
              setShowSnapshotModal(true);
            }}
          />
        )}
        {tab === 'snapshots' && <SnapshotsTab />}
        {tab === 'analysis' && <AnalysisTab />}
        {tab === 'code' && <CodeTab />}
        {tab === 'state' && <StateTab states={states} tasksTree={tasksTree} />}
      </main>

      {showSnapshotModal && snapshotCaptureId && (
        <SaveSnapshotModal
          captureId={snapshotCaptureId}
          onClose={() => {
            setShowSnapshotModal(false);
            setSnapshotCaptureId(null);
          }}
          onSaved={() => {
            setShowSnapshotModal(false);
            setSnapshotCaptureId(null);
            setTab('snapshots');
          }}
        />
      )}
    </div>
  );
}

// Dashboard Tab
interface DashboardTabProps {
  status: Status | null;
  loading: boolean;
  onFetchAccounts: () => void;
  onFetchPayments: () => void;
  onFetchBalances: () => void;
  onFetchAll: () => void;
  onReset: () => void;
}

function DashboardTab({ status, loading, onFetchAccounts, onFetchPayments, onFetchBalances, onFetchAll, onReset }: DashboardTabProps) {
  return (
    <div className="dashboard">
      <section className="actions-section">
        <h2>Actions</h2>
        <div className="actions-grid">
          <button className="btn-primary btn-large" onClick={onFetchAll} disabled={loading}>
            {loading ? <span className="spinner" /> : '[>]'} RUN FULL CYCLE
          </button>
          <button className="btn-secondary" onClick={onFetchAccounts} disabled={loading}>
            FETCH ACCOUNTS
          </button>
          <button className="btn-secondary" onClick={onFetchPayments} disabled={loading}>
            FETCH PAYMENTS
          </button>
          <button className="btn-secondary" onClick={onFetchBalances} disabled={loading}>
            FETCH BALANCES
          </button>
          <button className="btn-danger" onClick={onReset} disabled={loading}>
            [x] RESET ALL
          </button>
        </div>
      </section>

      <section className="stats-section">
        <h2>Storage</h2>
        <div className="grid grid-4">
          <StatCard label="Accounts" value={status?.storage.accounts_count ?? 0} icon="[A]" />
          <StatCard label="Payments" value={status?.storage.payments_count ?? 0} icon="[P]" />
          <StatCard label="Balances" value={status?.storage.balances_count ?? 0} icon="[$]" />
          <StatCard label="External Accounts" value={status?.storage.external_accounts_count ?? 0} icon="[E]" />
        </div>
      </section>

      <section className="stats-section">
        <h2>Fetch Status</h2>
        <div className="grid grid-3">
          <FetchStatusCard
            label="Accounts"
            status={status?.fetch_status.accounts}
          />
          <div className="card">
            <div className="card-title">Payments</div>
            {status?.fetch_status.payments && Object.keys(status.fetch_status.payments).length > 0 ? (
              Object.entries(status.fetch_status.payments).map(([key, val]) => (
                <div key={key} className="fetch-status-item">
                  <span className="mono truncate">{key === '_root' ? 'Root' : key}</span>
                  <span>{val.total_items} items</span>
                  <span className={val.has_more ? 'badge badge-info' : 'badge badge-success'}>
                    {val.has_more ? 'More' : 'Done'}
                  </span>
                </div>
              ))
            ) : (
              <div className="text-muted">Not started</div>
            )}
          </div>
          <div className="card">
            <div className="card-title">Balances</div>
            {status?.fetch_status.balances && Object.keys(status.fetch_status.balances).length > 0 ? (
              Object.entries(status.fetch_status.balances).map(([key, val]) => (
                <div key={key} className="fetch-status-item">
                  <span className="mono truncate">{key === '_root' ? 'Root' : key}</span>
                  <span>{val.total_items} items</span>
                  <span className={val.has_more ? 'badge badge-info' : 'badge badge-success'}>
                    {val.has_more ? 'More' : 'Done'}
                  </span>
                </div>
              ))
            ) : (
              <div className="text-muted">Not started</div>
            )}
          </div>
        </div>
      </section>

      <section className="stats-section">
        <h2>Debug Stats</h2>
        <div className="grid grid-4">
          <StatCard label="Total Entries" value={status?.debug.total_entries ?? 0} icon="[#]" />
          <StatCard label="Plugin Calls" value={status?.debug.total_plugin_calls ?? 0} icon="[>]" />
          <StatCard label="HTTP Requests" value={status?.debug.total_http_requests ?? 0} icon="[~]" />
          <StatCard 
            label="Errors" 
            value={status?.debug.error_count ?? 0} 
            icon="[!]"
            variant={status?.debug.error_count ? 'danger' : undefined}
          />
        </div>
      </section>
    </div>
  );
}

function StatCard({ label, value, icon, variant }: { label: string; value: number; icon: string; variant?: 'danger' }) {
  return (
    <div className={`stat-card ${variant ? `stat-card-${variant}` : ''}`}>
      <div className="stat-icon">{icon}</div>
      <div className="stat-content">
        <div className="stat-value">{value.toLocaleString()}</div>
        <div className="stat-label">{label}</div>
      </div>
    </div>
  );
}

function FetchStatusCard({ label, status }: { label: string; status?: { has_more: boolean; pages_fetched: number; total_items: number } }) {
  return (
    <div className="card">
      <div className="card-title">{label}</div>
      {status ? (
        <>
          <div className="fetch-status-item">
            <span>Pages fetched</span>
            <span className="mono">{status.pages_fetched}</span>
          </div>
          <div className="fetch-status-item">
            <span>Total items</span>
            <span className="mono">{status.total_items}</span>
          </div>
          <div className="fetch-status-item">
            <span>Status</span>
            <span className={status.has_more ? 'badge badge-info' : 'badge badge-success'}>
              {status.has_more ? 'More available' : 'Complete'}
            </span>
          </div>
        </>
      ) : (
        <div className="text-muted">Not started</div>
      )}
    </div>
  );
}

// Accounts Tab
function AccountsTab({ accounts }: { accounts: Account[] }) {
  const [selected, setSelected] = useState<Account | null>(null);

  if (accounts.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">[A]</div>
        <p>No accounts fetched yet</p>
        <p className="text-muted">Use "Fetch Accounts" to get started</p>
      </div>
    );
  }

  return (
    <div className="data-tab">
      <div className="data-list">
        <table>
          <thead>
            <tr>
              <th>Reference</th>
              <th>Name</th>
              <th>Asset</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            {accounts.map((acc) => (
              <tr 
                key={acc.reference} 
                onClick={() => setSelected(acc)}
                className={selected?.reference === acc.reference ? 'selected' : ''}
              >
                <td className="mono">{acc.reference}</td>
                <td>{acc.name || '-'}</td>
                <td>{acc.default_asset || '-'}</td>
                <td>{new Date(acc.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {selected && (
        <div className="data-detail">
          <div className="card">
            <div className="card-header">
              <div className="card-title">Account Details</div>
              <button className="btn-secondary btn-small" onClick={() => setSelected(null)}>×</button>
            </div>
            <pre>{JSON.stringify(selected, null, 2)}</pre>
          </div>
        </div>
      )}
    </div>
  );
}

// Payments Tab
function PaymentsTab({ payments }: { payments: Payment[] }) {
  const [selected, setSelected] = useState<Payment | null>(null);

  if (payments.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">[P]</div>
        <p>No payments fetched yet</p>
        <p className="text-muted">Use "Fetch Payments" to get started</p>
      </div>
    );
  }

  return (
    <div className="data-tab">
      <div className="data-list">
        <table>
          <thead>
            <tr>
              <th>Reference</th>
              <th>Type</th>
              <th>Status</th>
              <th>Amount</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            {payments.map((pay) => (
              <tr 
                key={pay.reference}
                onClick={() => setSelected(pay)}
                className={selected?.reference === pay.reference ? 'selected' : ''}
              >
                <td className="mono truncate">{pay.reference}</td>
                <td><span className="badge badge-info">{pay.type}</span></td>
                <td>
                  <span className={`badge ${
                    pay.status === 'SUCCEEDED' ? 'badge-success' :
                    pay.status === 'FAILED' ? 'badge-danger' : 'badge-warning'
                  }`}>
                    {pay.status}
                  </span>
                </td>
                <td className="mono">{pay.amount} {pay.asset}</td>
                <td>{new Date(pay.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {selected && (
        <div className="data-detail">
          <div className="card">
            <div className="card-header">
              <div className="card-title">Payment Details</div>
              <button className="btn-secondary btn-small" onClick={() => setSelected(null)}>×</button>
            </div>
            <pre>{JSON.stringify(selected, null, 2)}</pre>
          </div>
        </div>
      )}
    </div>
  );
}

// Balances Tab
function BalancesTab({ balances }: { balances: Balance[] }) {
  if (balances.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">[$]</div>
        <p>No balances fetched yet</p>
        <p className="text-muted">Use "Fetch Balances" to get started</p>
      </div>
    );
  }

  return (
    <div className="data-tab">
      <table>
        <thead>
          <tr>
            <th>Account</th>
            <th>Asset</th>
            <th>Amount</th>
            <th>As Of</th>
          </tr>
        </thead>
        <tbody>
          {balances.map((bal, i) => (
            <tr key={i}>
              <td className="mono">{bal.account_reference}</td>
              <td>{bal.asset}</td>
              <td className="mono">{bal.amount}</td>
              <td>{new Date(bal.created_at).toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// Debug Tab
interface DebugTabProps {
  logs: DebugEntry[];
  pluginCalls: PluginCall[];
  httpRequests: HTTPRequest[];
  httpCaptureEnabled: boolean;
  onClear: () => void;
  onToggleCapture: () => void;
  onSaveSnapshot?: (captureId: string) => void;
}

function DebugTab({ logs, pluginCalls, httpRequests, httpCaptureEnabled, onClear, onToggleCapture, onSaveSnapshot }: DebugTabProps) {
  const [view, setView] = useState<'logs' | 'calls' | 'http'>('http');
  const [selectedCall, setSelectedCall] = useState<PluginCall | null>(null);
  const [selectedRequest, setSelectedRequest] = useState<HTTPRequest | null>(null);

  return (
    <div className="debug-tab">
      <div className="debug-header">
        <div className="tabs">
          <button className={`tab ${view === 'http' ? 'active' : ''}`} onClick={() => setView('http')}>
            HTTP Traffic ({httpRequests.length})
          </button>
          <button className={`tab ${view === 'calls' ? 'active' : ''}`} onClick={() => setView('calls')}>
            Plugin Calls ({pluginCalls.length})
          </button>
          <button className={`tab ${view === 'logs' ? 'active' : ''}`} onClick={() => setView('logs')}>
            Logs ({logs.length})
          </button>
        </div>
        <div className="debug-actions">
          <button 
            className={`btn-small ${httpCaptureEnabled ? 'btn-primary' : 'btn-secondary'}`}
            onClick={onToggleCapture}
          >
            {httpCaptureEnabled ? '[*] CAPTURE ON' : '[ ] CAPTURE OFF'}
          </button>
          <button className="btn-secondary btn-small" onClick={onClear}>CLEAR ALL</button>
        </div>
      </div>

      {view === 'http' && (
        <div className="http-requests">
          <div className="requests-list">
            {httpRequests.length === 0 ? (
              <div className="empty-state">
                <div className="empty-state-icon">[~]</div>
                <p className="text-muted">No HTTP requests captured yet</p>
                <p className="text-muted">Run a fetch operation to see traffic</p>
              </div>
            ) : (
              httpRequests.map((req) => (
                <div 
                  key={req.id}
                  className={`request-entry ${req.error || (req.response_status && req.response_status >= 400) ? 'request-error' : ''} ${selectedRequest?.id === req.id ? 'selected' : ''}`}
                  onClick={() => setSelectedRequest(req)}
                >
                  <div className="request-header">
                    <span className={`request-method method-${req.method.toLowerCase()}`}>{req.method}</span>
                    <span className="request-url">{new URL(req.url).pathname}</span>
                    {req.response_status && (
                      <span className={`request-status status-${Math.floor(req.response_status / 100)}xx`}>
                        {req.response_status}
                      </span>
                    )}
                    <span className="request-duration">{(req.duration / 1000000).toFixed(0)}ms</span>
                  </div>
                  <div className="request-host">{new URL(req.url).host}</div>
                </div>
              ))
            )}
          </div>
          {selectedRequest && (
            <div className="request-detail">
              <div className="card">
                <div className="card-header">
                  <div className="card-title">
                    <span className={`request-method method-${selectedRequest.method.toLowerCase()}`}>
                      {selectedRequest.method}
                    </span>
                    {selectedRequest.response_status && (
                      <span className={`request-status status-${Math.floor(selectedRequest.response_status / 100)}xx`}>
                        {selectedRequest.response_status}
                      </span>
                    )}
                  </div>
                  <div className="card-actions">
                    {onSaveSnapshot && (
                      <button 
                        className="btn-primary btn-small" 
                        onClick={() => onSaveSnapshot(selectedRequest.id)}
                      >
                        [+] SAVE SNAPSHOT
                      </button>
                    )}
                    <button className="btn-secondary btn-small" onClick={() => setSelectedRequest(null)}>×</button>
                  </div>
                </div>
                
                <div className="request-url-full">{selectedRequest.url}</div>
                <div className="request-timing">
                  {new Date(selectedRequest.timestamp).toLocaleTimeString()} · {(selectedRequest.duration / 1000000).toFixed(2)}ms
                </div>

                {selectedRequest.error && (
                  <>
                    <h4>Error</h4>
                    <pre className="text-danger">{selectedRequest.error}</pre>
                  </>
                )}

                <h4>Request Headers</h4>
                <pre>{JSON.stringify(selectedRequest.request_headers, null, 2)}</pre>

                {selectedRequest.request_body && (
                  <>
                    <h4>Request Body</h4>
                    <pre>{formatBody(selectedRequest.request_body)}</pre>
                  </>
                )}

                {selectedRequest.response_headers && (
                  <>
                    <h4>Response Headers</h4>
                    <pre>{JSON.stringify(selectedRequest.response_headers, null, 2)}</pre>
                  </>
                )}

                {selectedRequest.response_body && (
                  <>
                    <h4>Response Body</h4>
                    <pre>{formatBody(selectedRequest.response_body)}</pre>
                  </>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {view === 'logs' && (
        <div className="log-list">
          {logs.length === 0 ? (
            <div className="empty-state">
              <p className="text-muted">No logs yet</p>
            </div>
          ) : (
            logs.map((log) => (
              <div key={log.id} className={`log-entry log-${log.type}`}>
                <span className="log-time">{new Date(log.timestamp).toLocaleTimeString()}</span>
                <span className={`log-type badge badge-${
                  log.type === 'error' ? 'danger' :
                  log.type === 'plugin_call' ? 'info' :
                  log.type === 'state_change' ? 'warning' : 'success'
                }`}>
                  {log.type}
                </span>
                <span className="log-operation">{log.operation}</span>
                {log.message && <span className="log-message">{log.message}</span>}
                {log.error && <span className="log-error text-danger">{log.error}</span>}
                {log.duration && <span className="log-duration">{log.duration}ns</span>}
              </div>
            ))
          )}
        </div>
      )}

      {view === 'calls' && (
        <div className="plugin-calls">
          <div className="calls-list">
            {pluginCalls.length === 0 ? (
              <div className="empty-state">
                <p className="text-muted">No plugin calls yet</p>
              </div>
            ) : (
              pluginCalls.map((call) => (
                <div 
                  key={call.id}
                  className={`call-entry ${call.error ? 'call-error' : ''} ${selectedCall?.id === call.id ? 'selected' : ''}`}
                  onClick={() => setSelectedCall(call)}
                >
                  <div className="call-header">
                    <span className="call-method">{call.method}</span>
                    <span className="call-duration">{(call.duration / 1000000).toFixed(2)}ms</span>
                  </div>
                  <div className="call-time">{new Date(call.timestamp).toLocaleTimeString()}</div>
                  {call.error && <div className="call-error-msg text-danger">{call.error}</div>}
                </div>
              ))
            )}
          </div>
          {selectedCall && (
            <div className="call-detail">
              <div className="card">
                <div className="card-header">
                  <div className="card-title">{selectedCall.method}</div>
                  <button className="btn-secondary btn-small" onClick={() => setSelectedCall(null)}>×</button>
                </div>
                <h4>Input</h4>
                <pre>{JSON.stringify(selectedCall.input, null, 2)}</pre>
                {selectedCall.output !== undefined && selectedCall.output !== null && (
                  <>
                    <h4>Output</h4>
                    <pre>{JSON.stringify(selectedCall.output, null, 2)}</pre>
                  </>
                )}
                {selectedCall.error && (
                  <>
                    <h4>Error</h4>
                    <pre className="text-danger">{selectedCall.error}</pre>
                  </>
                )}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// Helper to format JSON bodies nicely
function formatBody(body: string): string {
  try {
    const parsed = JSON.parse(body);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return body;
  }
}

// Save Snapshot Modal
interface SaveSnapshotModalProps {
  captureId: string;
  onClose: () => void;
  onSaved: () => void;
}

function SaveSnapshotModal({ captureId, onClose, onSaved }: SaveSnapshotModalProps) {
  const [name, setName] = useState('');
  const [operation, setOperation] = useState('');
  const [description, setDescription] = useState('');
  const [tags, setTags] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSave = async () => {
    if (!name.trim()) {
      setError('Name is required');
      return;
    }
    if (!operation.trim()) {
      setError('Operation is required');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      await api.createSnapshotFromCapture(captureId, {
        name: name.trim(),
        operation: operation.trim(),
        description: description.trim() || undefined,
        tags: tags.trim() ? tags.split(',').map(t => t.trim()) : undefined,
      });
      onSaved();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h3>Save as Test Snapshot</h3>
          <button className="btn-icon" onClick={onClose}>×</button>
        </div>
        <div className="modal-body">
          {error && <div className="alert alert-danger">{error}</div>}
          
          <div className="form-group">
            <label>Name *</label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="e.g., fetch_accounts_page_1"
              autoFocus
            />
          </div>

          <div className="form-group">
            <label>Operation *</label>
            <select value={operation} onChange={e => setOperation(e.target.value)}>
              <option value="">Select operation...</option>
              <option value="fetch_accounts">fetch_accounts</option>
              <option value="fetch_payments">fetch_payments</option>
              <option value="fetch_balances">fetch_balances</option>
              <option value="fetch_external_accounts">fetch_external_accounts</option>
              <option value="create_webhooks">create_webhooks</option>
              <option value="create_transfer">create_transfer</option>
              <option value="create_payout">create_payout</option>
              <option value="other">other</option>
            </select>
          </div>

          <div className="form-group">
            <label>Description</label>
            <textarea
              value={description}
              onChange={e => setDescription(e.target.value)}
              placeholder="Optional description of this test case"
              rows={2}
            />
          </div>

          <div className="form-group">
            <label>Tags (comma-separated)</label>
            <input
              type="text"
              value={tags}
              onChange={e => setTags(e.target.value)}
              placeholder="e.g., pagination, error-case"
            />
          </div>
        </div>
        <div className="modal-footer">
          <button className="btn-secondary" onClick={onClose} disabled={loading}>
            Cancel
          </button>
          <button className="btn-primary" onClick={handleSave} disabled={loading}>
            {loading ? <span className="spinner" /> : 'Save Snapshot'}
          </button>
        </div>
      </div>
    </div>
  );
}

// Code Tab
function CodeTab() {
  const [info, setInfo] = useState<ConnectorInfo | null>(null);
  const [files, setFiles] = useState<FileNode[]>([]);
  const [selectedFile, setSelectedFile] = useState<SourceFile | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [view, setView] = useState<'overview' | 'files' | 'search'>('overview');
  const [loading, setLoading] = useState(false);
  const [expandedDirs, setExpandedDirs] = useState<Set<string>>(new Set());

  useEffect(() => {
    loadInfo();
  }, []);

  const loadInfo = async () => {
    try {
      const [infoRes, filesRes] = await Promise.all([
        api.getConnectorInfo(),
        api.getFileTree(),
      ]);
      setInfo(infoRes);
      setFiles(filesRes.files || []);
    } catch (e) {
      console.error('Failed to load connector info:', e);
    }
  };

  const loadFile = async (path: string) => {
    setLoading(true);
    try {
      const file = await api.getFile(path);
      setSelectedFile(file);
    } catch (e) {
      console.error('Failed to load file:', e);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = async () => {
    if (!searchQuery.trim()) return;
    setLoading(true);
    try {
      const res = await api.searchCode(searchQuery);
      setSearchResults(res.results || []);
      setView('search');
    } catch (e) {
      console.error('Search failed:', e);
    } finally {
      setLoading(false);
    }
  };

  const toggleDir = (path: string) => {
    const newExpanded = new Set(expandedDirs);
    if (newExpanded.has(path)) {
      newExpanded.delete(path);
    } else {
      newExpanded.add(path);
    }
    setExpandedDirs(newExpanded);
  };

  const renderFileTree = (nodes: FileNode[], depth = 0): JSX.Element[] => {
    return nodes.map((node) => (
      <div key={node.path}>
        <div
          className={`file-tree-item ${selectedFile?.path === node.path ? 'selected' : ''}`}
          style={{ paddingLeft: `${12 + depth * 16}px` }}
          onClick={() => {
            if (node.is_dir) {
              toggleDir(node.path);
            } else {
              loadFile(node.path);
            }
          }}
        >
          <span className="file-icon">
            {node.is_dir ? (expandedDirs.has(node.path) ? '[-]' : '[+]') : getFileIcon(node.name)}
          </span>
          <span className="file-name">{node.name}</span>
        </div>
        {node.is_dir && node.children && expandedDirs.has(node.path) && (
          <div className="file-tree-children">
            {renderFileTree(node.children, depth + 1)}
          </div>
        )}
      </div>
    ));
  };

  return (
    <div className="code-tab">
      <div className="code-sidebar">
        <div className="tabs" style={{ padding: '0 12px' }}>
          <button className={`tab ${view === 'overview' ? 'active' : ''}`} onClick={() => setView('overview')}>
            Overview
          </button>
          <button className={`tab ${view === 'files' ? 'active' : ''}`} onClick={() => setView('files')}>
            Files
          </button>
        </div>

        <div className="search-box">
          <input
            type="text"
            placeholder="Search code..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          />
          <button onClick={handleSearch} disabled={loading}>🔍</button>
        </div>

        {view === 'overview' && info && (
          <div className="code-overview">
            <section>
              <h4>Capabilities</h4>
              <div className="capability-list">
                {info.capabilities.map((cap) => (
                  <div key={cap.name} className={`capability-item ${cap.supported ? 'supported' : 'unsupported'}`}>
                    <span className="capability-icon">{cap.supported ? '✅' : '❌'}</span>
                    <span className="capability-name">{cap.name}</span>
                  </div>
                ))}
              </div>
            </section>

            <section>
              <h4>Configuration</h4>
              <div className="config-list">
                {info.config.map((param) => (
                  <div key={param.name} className="config-item">
                    <span className="config-name">{param.name}</span>
                    <span className="config-type">{param.type}</span>
                    {param.required && <span className="config-required">required</span>}
                  </div>
                ))}
              </div>
            </section>

            {info.methods && info.methods.length > 0 && (
              <section>
                <h4>Plugin Methods</h4>
                <div className="methods-list">
                  {info.methods.map((method) => (
                    <div
                      key={method.name}
                      className={`method-item ${method.implemented ? 'implemented' : 'not-implemented'}`}
                      onClick={() => {
                        if (method.file) {
                          loadFile(method.file);
                          setView('files');
                        }
                      }}
                    >
                      <span className="method-icon">{method.implemented ? '✓' : '○'}</span>
                      <span className="method-name">{method.name}</span>
                      {method.file && <span className="method-file">{method.file}:{method.line}</span>}
                    </div>
                  ))}
                </div>
              </section>
            )}
          </div>
        )}

        {view === 'files' && (
          <div className="file-tree">
            {files.length === 0 ? (
              <div className="empty-state">
                <p className="text-muted">Source files not available</p>
              </div>
            ) : (
              renderFileTree(files)
            )}
          </div>
        )}

        {view === 'search' && (
          <div className="search-results">
            <div className="search-header">
              <span>{searchResults.length} results for "{searchQuery}"</span>
              <button className="btn-small btn-secondary" onClick={() => setView('files')}>×</button>
            </div>
            {searchResults.map((result, i) => (
              <div
                key={i}
                className="search-result"
                onClick={() => {
                  loadFile(result.file);
                }}
              >
                <div className="search-result-file">{result.file}:{result.line}</div>
                <div className="search-result-content">{result.content}</div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="code-main">
        {loading && (
          <div className="code-loading">
            <span className="spinner" />
            Loading...
          </div>
        )}
        {!loading && !selectedFile && (
          <div className="empty-state">
            <div className="empty-state-icon">[F]</div>
            <p>Select a file to view its contents</p>
          </div>
        )}
        {!loading && selectedFile && (
          <div className="code-viewer">
            <div className="code-header">
              <span className="code-path">{selectedFile.path}</span>
              <span className="code-info">{selectedFile.lines} lines · {selectedFile.language}</span>
            </div>
            {selectedFile.symbols && selectedFile.symbols.length > 0 && (
              <div className="code-symbols">
                <span className="symbols-label">Symbols:</span>
                {selectedFile.symbols.slice(0, 20).map((sym, i) => (
                  <span key={i} className={`symbol symbol-${sym.kind}`} title={sym.doc || sym.signature}>
                    {sym.name}
                  </span>
                ))}
                {selectedFile.symbols.length > 20 && (
                  <span className="symbols-more">+{selectedFile.symbols.length - 20} more</span>
                )}
              </div>
            )}
            <pre className="code-content">
              <code>{addLineNumbers(selectedFile.content)}</code>
            </pre>
          </div>
        )}
      </div>
    </div>
  );
}

function getFileIcon(name: string): string {
  if (name.endsWith('_test.go')) return '[T]';
  if (name.endsWith('.go')) return '[G]';
  if (name.endsWith('.json')) return '[J]';
  if (name.endsWith('.yaml') || name.endsWith('.yml')) return '[Y]';
  if (name.endsWith('.md')) return '[M]';
  return '[F]';
}

function addLineNumbers(content: string): string {
  const lines = content.split('\n');
  const padding = String(lines.length).length;
  return lines
    .map((line, i) => {
      const num = String(i + 1).padStart(padding, ' ');
      return `${num} │ ${line}`;
    })
    .join('\n');
}

// Tasks Tab
function TasksTab() {
  const [taskData, setTaskData] = useState<TaskTreeSummary | null>(null);
  const [executions, setExecutions] = useState<TaskExecution[]>([]);
  const [loading, setLoading] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const [tasks, execs] = await Promise.all([
        api.getTasks(),
        api.getTaskExecutions(30),
      ]);
      setTaskData(tasks);
      setExecutions(execs.executions || []);
    } catch (e) {
      console.error('Failed to load tasks:', e);
    }
  }, []);

  useEffect(() => {
    refresh();
    const interval = setInterval(refresh, 1000);
    return () => clearInterval(interval);
  }, [refresh]);

  const handleStep = async () => {
    setLoading(true);
    try {
      await api.stepTask();
    } finally {
      setLoading(false);
    }
  };

  const toggleStepMode = async () => {
    if (!taskData) return;
    setLoading(true);
    try {
      await api.setStepMode(!taskData.step_mode);
      refresh();
    } finally {
      setLoading(false);
    }
  };

  const handleReset = async () => {
    setLoading(true);
    try {
      await api.resetTasks();
      refresh();
    } finally {
      setLoading(false);
    }
  };

  const renderTaskNode = (node: TaskNodeSummary, depth = 0): JSX.Element => {
    const statusColors: Record<string, string> = {
      pending: 'var(--text-muted)',
      running: 'var(--accent)',
      completed: '#3fb950',
      failed: '#f85149',
      skipped: 'var(--text-muted)',
    };

    const statusIcons: Record<string, string> = {
      pending: '○',
      running: '◉',
      completed: '✓',
      failed: '✗',
      skipped: '⊘',
    };

    return (
      <div key={node.id} className="task-node" style={{ marginLeft: depth * 20 }}>
        <div className={`task-node-header task-${node.status}`}>
          <span className="task-status-icon" style={{ color: statusColors[node.status] }}>
            {node.status === 'running' ? <span className="spinner" /> : statusIcons[node.status]}
          </span>
          <span className="task-type">{formatTaskType(node.type)}</span>
          {node.name && <span className="task-name">{node.name}</span>}
          <span className="task-stats">
            {node.items_count > 0 && <span className="task-items">{node.items_count} items</span>}
            {node.duration && <span className="task-duration">{node.duration}</span>}
            {node.last_exec_time && <span className="task-time">{node.last_exec_time}</span>}
          </span>
        </div>
        {node.error && <div className="task-error">{node.error}</div>}
        {node.children && node.children.length > 0 && (
          <div className="task-children">
            {node.children.map((child) => renderTaskNode(child, depth + 1))}
          </div>
        )}
        {node.child_count > 0 && (!node.children || node.children.length === 0) && (
          <div className="task-child-count" style={{ marginLeft: (depth + 1) * 20 }}>
            {node.child_count} child tasks...
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="tasks-tab">
      <div className="tasks-header">
        <div className="tasks-controls">
          <button 
            className={`btn-small ${taskData?.step_mode ? 'btn-primary' : 'btn-secondary'}`}
            onClick={toggleStepMode}
            disabled={loading}
          >
            {taskData?.step_mode ? '[||] STEP MODE ON' : '[>] STEP MODE OFF'}
          </button>
          {taskData?.step_mode && (
            <button 
              className="btn-primary btn-small"
              onClick={handleStep}
              disabled={loading || !taskData?.is_running}
            >
              {'[>>]'} NEXT STEP
            </button>
          )}
          <button 
            className="btn-secondary btn-small"
            onClick={handleReset}
            disabled={loading}
          >
            [x] RESET
          </button>
        </div>
        {taskData && (
          <div className="tasks-stats">
            <span className="stat">
              <strong>{taskData.stats.total_executions}</strong> executions
            </span>
            <span className="stat success">
              <strong>{taskData.stats.success_count}</strong> success
            </span>
            {taskData.stats.failure_count > 0 && (
              <span className="stat error">
                <strong>{taskData.stats.failure_count}</strong> failed
              </span>
            )}
            <span className="stat">
              <strong>{taskData.stats.total_items_fetched}</strong> items
            </span>
            {taskData.is_running && <span className="stat running">● Running</span>}
          </div>
        )}
      </div>

      <div className="tasks-content">
        <div className="tasks-tree-panel">
          <h3>Task Tree</h3>
          {taskData?.current_task && (
            <div className="current-task">
              <span className="current-task-label">Currently running:</span>
              <span className="current-task-name">{formatTaskType(taskData.current_task.type)}</span>
            </div>
          )}
          <div className="task-tree">
            {taskData?.tree && taskData.tree.length > 0 ? (
              taskData.tree.map((node) => renderTaskNode(node))
            ) : (
              <div className="empty-state">
                <p className="text-muted">No tasks yet</p>
                <p className="text-muted">Run a fetch cycle to see the task tree</p>
              </div>
            )}
          </div>
        </div>

        <div className="tasks-history-panel">
          <h3>Execution History</h3>
          <div className="execution-list">
            {executions.length === 0 ? (
              <div className="empty-state">
                <p className="text-muted">No executions yet</p>
              </div>
            ) : (
              executions.map((exec) => (
                <div key={exec.id} className={`execution-item exec-${exec.status}`}>
                  <div className="execution-header">
                    <span className={`execution-status status-${exec.status}`}>
                      {exec.status === 'completed' ? '✓' : exec.status === 'failed' ? '✗' : '○'}
                    </span>
                    <span className="execution-page">Page {exec.page_number}</span>
                    <span className="execution-items">{exec.items_count} items</span>
                    <span className="execution-duration">{(exec.duration / 1000000).toFixed(0)}ms</span>
                    {exec.has_more && <span className="execution-more">more →</span>}
                  </div>
                  <div className="execution-time">
                    {new Date(exec.started_at).toLocaleTimeString()}
                  </div>
                  {exec.error && <div className="execution-error">{exec.error}</div>}
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function formatTaskType(type: string): string {
  // Handle byte values (old format) - show as hex
  if (type.length === 1 && type.charCodeAt(0) < 32) {
    const taskNames: Record<number, string> = {
      0: 'Fetch Others',
      1: 'Fetch Accounts',
      2: 'Fetch Balances',
      3: 'Fetch External Accounts',
      4: 'Fetch Payments',
      5: 'Create Webhooks',
    };
    return taskNames[type.charCodeAt(0)] || `Task ${type.charCodeAt(0)}`;
  }
  // Handle new string format
  return type
    .replace('TASK_', '')
    .replace('FETCH_', 'Fetch ')
    .replace('CREATE_', 'Create ')
    .replace(/_/g, ' ')
    .toLowerCase()
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

// Snapshots Tab
function SnapshotsTab() {
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [stats, setStats] = useState<SnapshotStats | null>(null);
  const [selectedSnapshot, setSelectedSnapshot] = useState<Snapshot | null>(null);
  const [showTestPreview, setShowTestPreview] = useState(false);
  const [testPreview, setTestPreview] = useState<GenerateResult | null>(null);
  const [loading, setLoading] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const [snapshotData, statsData] = await Promise.all([
        api.listSnapshots(),
        api.getSnapshotStats(),
      ]);
      setSnapshots(snapshotData.snapshots || []);
      setStats(statsData);
    } catch (e) {
      console.error('Failed to load snapshots:', e);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this snapshot?')) return;
    setLoading(true);
    try {
      await api.deleteSnapshot(id);
      refresh();
      if (selectedSnapshot?.id === id) {
        setSelectedSnapshot(null);
      }
    } finally {
      setLoading(false);
    }
  };

  const handlePreviewTests = async () => {
    setLoading(true);
    try {
      const preview = await api.previewTests();
      setTestPreview(preview);
      setShowTestPreview(true);
    } catch (e: unknown) {
      alert('Failed to generate preview: ' + (e instanceof Error ? e.message : String(e)));
    } finally {
      setLoading(false);
    }
  };

  const handleGenerateTests = async () => {
    const outputDir = prompt('Output directory (leave empty to preview only):');
    setLoading(true);
    try {
      const result = await api.generateTests(outputDir || undefined);
      setTestPreview(result);
      setShowTestPreview(true);
      if (outputDir) {
        alert('Tests generated successfully!');
      }
    } catch (e: unknown) {
      alert('Failed to generate tests: ' + (e instanceof Error ? e.message : String(e)));
    } finally {
      setLoading(false);
    }
  };

  const handleClearAll = async () => {
    if (!confirm('Delete all snapshots?')) return;
    setLoading(true);
    try {
      await api.clearSnapshots();
      refresh();
      setSelectedSnapshot(null);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="snapshots-tab">
      <div className="snapshots-header">
        <div className="snapshots-controls">
          <button 
            className="btn-primary btn-small" 
            onClick={handlePreviewTests}
            disabled={loading || snapshots.length === 0}
          >
            [?] PREVIEW TESTS
          </button>
          <button 
            className="btn-primary btn-small" 
            onClick={handleGenerateTests}
            disabled={loading || snapshots.length === 0}
          >
            [!] GENERATE TESTS
          </button>
          <button 
            className="btn-secondary btn-small" 
            onClick={handleClearAll}
            disabled={loading || snapshots.length === 0}
          >
            [x] CLEAR ALL
          </button>
        </div>
        {stats && (
          <div className="snapshots-stats">
            <span className="stat"><strong>{stats.total}</strong> snapshots</span>
            {Object.entries(stats.by_operation).map(([op, count]) => (
              <span key={op} className="stat">{op}: {count}</span>
            ))}
          </div>
        )}
      </div>

      {showTestPreview && testPreview && (
        <div className="test-preview-modal">
          <div className="test-preview-content">
            <div className="test-preview-header">
              <h3>Generated Test Preview</h3>
              <button className="btn-secondary btn-small" onClick={() => setShowTestPreview(false)}>×</button>
            </div>
            <div className="test-preview-body">
              <div className="test-file">
                <h4>{testPreview.test_file.filename}</h4>
                <pre className="code-block">{testPreview.test_file.content}</pre>
              </div>
              <div className="test-fixtures">
                <h4>Fixtures ({testPreview.fixtures.length})</h4>
                {testPreview.fixtures.map((f) => (
                  <details key={f.filename}>
                    <summary>{f.filename}</summary>
                    <pre className="code-block">{f.content}</pre>
                  </details>
                ))}
              </div>
              <div className="test-instructions">
                <h4>Instructions</h4>
                <pre>{testPreview.instructions}</pre>
              </div>
            </div>
          </div>
        </div>
      )}

      <div className="snapshots-content">
        <div className="snapshots-list-panel">
          <h3>Saved Snapshots</h3>
          <div className="snapshots-list">
            {snapshots.length === 0 ? (
              <div className="empty-state">
                <div className="empty-state-icon">[S]</div>
                <p className="text-muted">No snapshots saved yet</p>
                <p className="text-muted">Go to Debug {'->'} HTTP Traffic and click "Save Snapshot" on a request</p>
              </div>
            ) : (
              snapshots.map((snap) => (
                <div 
                  key={snap.id}
                  className={`snapshot-item ${selectedSnapshot?.id === snap.id ? 'selected' : ''}`}
                  onClick={() => setSelectedSnapshot(snap)}
                >
                  <div className="snapshot-header">
                    <span className="snapshot-name">{snap.name}</span>
                    <button 
                      className="btn-icon" 
                      onClick={(e) => { e.stopPropagation(); handleDelete(snap.id); }}
                      title="Delete"
                    >
                      [x]
                    </button>
                  </div>
                  <div className="snapshot-meta">
                    <span className="snapshot-operation">{snap.operation}</span>
                    <span className="snapshot-method">{snap.request.method}</span>
                    <span className="snapshot-status">{snap.response.status_code}</span>
                  </div>
                  {snap.tags && snap.tags.length > 0 && (
                    <div className="snapshot-tags">
                      {snap.tags.map((tag) => (
                        <span key={tag} className="tag">{tag}</span>
                      ))}
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </div>

        {selectedSnapshot && (
          <div className="snapshot-detail-panel">
            <div className="card">
              <div className="card-header">
                <div className="card-title">{selectedSnapshot.name}</div>
              </div>
              {selectedSnapshot.description && (
                <p className="snapshot-description">{selectedSnapshot.description}</p>
              )}
              <div className="snapshot-info">
                <div><strong>Operation:</strong> {selectedSnapshot.operation}</div>
                <div><strong>Created:</strong> {new Date(selectedSnapshot.created_at).toLocaleString()}</div>
              </div>

              <h4>Request</h4>
              <div className="request-summary">
                <span className={`request-method method-${selectedSnapshot.request.method.toLowerCase()}`}>
                  {selectedSnapshot.request.method}
                </span>
                <span className="request-url">{selectedSnapshot.request.url}</span>
              </div>
              {selectedSnapshot.request.body && (
                <>
                  <h5>Request Body</h5>
                  <pre className="code-block">{formatBody(selectedSnapshot.request.body)}</pre>
                </>
              )}

              <h4>Response ({selectedSnapshot.response.status_code})</h4>
              {selectedSnapshot.response.body && (
                <pre className="code-block">{formatBody(selectedSnapshot.response.body)}</pre>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// Analysis Tab - Schema Inference & Baseline Comparison
function AnalysisTab() {
  const [view, setView] = useState<'schemas' | 'baselines'>('schemas');
  const [schemas, setSchemas] = useState<InferredSchema[]>([]);
  const [schemaBaselines, setSchemaBaselines] = useState<InferredSchema[]>([]);
  const [selectedSchema, setSelectedSchema] = useState<InferredSchema | null>(null);
  const [schemaDiff, setSchemaDiff] = useState<SchemaDiff | null>(null);
  
  const [baselines, setBaselines] = useState<DataBaseline[]>([]);
  const [selectedBaseline, setSelectedBaseline] = useState<DataBaseline | null>(null);
  const [baselineDiff, setBaselineDiff] = useState<BaselineDiff | null>(null);
  
  const [loading, setLoading] = useState(false);

  const refreshSchemas = useCallback(async () => {
    try {
      const [schemaData, baselineData] = await Promise.all([
        api.listSchemas(),
        api.listSchemaBaselines(),
      ]);
      setSchemas(schemaData.schemas || []);
      setSchemaBaselines(baselineData.baselines || []);
    } catch (e) {
      console.error('Failed to load schemas:', e);
    }
  }, []);

  const refreshBaselines = useCallback(async () => {
    try {
      const data = await api.listBaselines();
      setBaselines(data.baselines || []);
    } catch (e) {
      console.error('Failed to load baselines:', e);
    }
  }, []);

  useEffect(() => {
    refreshSchemas();
    refreshBaselines();
  }, [refreshSchemas, refreshBaselines]);

  const handleSaveSchemaBaselines = async () => {
    setLoading(true);
    try {
      await api.saveAllSchemaBaselines();
      await refreshSchemas();
      alert('Schema baselines saved!');
    } finally {
      setLoading(false);
    }
  };

  const handleCompareSchema = async (operation: string) => {
    setLoading(true);
    try {
      const diff = await api.compareSchema(operation);
      setSchemaDiff(diff);
    } catch (e: unknown) {
      alert('Error: ' + (e instanceof Error ? e.message : String(e)));
    } finally {
      setLoading(false);
    }
  };

  const handleSaveBaseline = async () => {
    const name = prompt('Baseline name (optional):');
    setLoading(true);
    try {
      await api.saveBaseline(name || undefined);
      await refreshBaselines();
    } finally {
      setLoading(false);
    }
  };

  const handleCompareBaseline = async (id: string) => {
    setLoading(true);
    try {
      const diff = await api.compareBaseline(id);
      setBaselineDiff(diff);
    } catch (e: unknown) {
      alert('Error: ' + (e instanceof Error ? e.message : String(e)));
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteBaseline = async (id: string) => {
    if (!confirm('Delete this baseline?')) return;
    setLoading(true);
    try {
      await api.deleteBaseline(id);
      await refreshBaselines();
      if (selectedBaseline?.id === id) {
        setSelectedBaseline(null);
        setBaselineDiff(null);
      }
    } finally {
      setLoading(false);
    }
  };

  const renderFieldSchema = (field: InferredSchema['root'], depth = 0): JSX.Element => {
    if (!field) return <></>;
    
    return (
      <div className="schema-field" style={{ marginLeft: depth * 16 }}>
        <div className="schema-field-row">
          <span className="schema-field-name">{field.name || 'root'}</span>
          <span className={`schema-type schema-type-${field.type}`}>{field.type}</span>
          {field.nullable && <span className="schema-nullable">nullable</span>}
          {field.examples && field.examples.length > 0 && (
            <span className="schema-example">e.g. {JSON.stringify(field.examples[0])}</span>
          )}
        </div>
        {field.properties && Object.entries(field.properties).map(([key, child]) => (
          <div key={key}>{renderFieldSchema(child, depth + 1)}</div>
        ))}
        {field.array_item && (
          <div className="schema-array-item">
            {renderFieldSchema(field.array_item, depth + 1)}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="analysis-tab">
      <div className="analysis-header">
        <div className="tabs">
          <button className={`tab ${view === 'schemas' ? 'active' : ''}`} onClick={() => setView('schemas')}>
            Schema Drift ({schemas.length})
          </button>
          <button className={`tab ${view === 'baselines' ? 'active' : ''}`} onClick={() => setView('baselines')}>
            Data Baselines ({baselines.length})
          </button>
        </div>
      </div>

      {view === 'schemas' && (
        <div className="schemas-view">
          <div className="schemas-actions">
            <button 
              className="btn-primary btn-small" 
              onClick={handleSaveSchemaBaselines}
              disabled={loading || schemas.length === 0}
            >
              [+] SAVE ALL AS BASELINES
            </button>
            <span className="text-muted">
              {schemaBaselines.length} baselines saved
            </span>
          </div>

          <div className="schemas-content">
            <div className="schemas-list-panel">
              <h3>Inferred Schemas</h3>
              <p className="text-muted">Schemas are auto-learned from HTTP responses</p>
              <div className="schemas-list">
                {schemas.length === 0 ? (
                  <div className="empty-state">
                    <p className="text-muted">No schemas yet</p>
                    <p className="text-muted">Run a fetch cycle to auto-learn API schemas</p>
                  </div>
                ) : (
                  schemas.map((schema) => {
                    const hasBaseline = schemaBaselines.some(b => b.operation === schema.operation);
                    return (
                      <div 
                        key={schema.id}
                        className={`schema-item ${selectedSchema?.id === schema.id ? 'selected' : ''}`}
                        onClick={() => { setSelectedSchema(schema); setSchemaDiff(null); }}
                      >
                        <div className="schema-item-header">
                          <span className="schema-operation">{schema.operation}</span>
                          {hasBaseline && <span className="schema-has-baseline">[*]</span>}
                        </div>
                        <div className="schema-item-meta">
                          <span>{schema.sample_count} samples</span>
                          <span>{schema.method} {schema.endpoint}</span>
                        </div>
                        {hasBaseline && (
                          <button 
                            className="btn-small btn-secondary"
                            onClick={(e) => { e.stopPropagation(); handleCompareSchema(schema.operation); }}
                          >
                            Compare
                          </button>
                        )}
                      </div>
                    );
                  })
                )}
              </div>
            </div>

            <div className="schema-detail-panel">
              {schemaDiff ? (
                <div className="schema-diff">
                  <h3>Schema Comparison: {schemaDiff.operation}</h3>
                  <div className={`diff-summary ${schemaDiff.has_changes ? 'has-changes' : 'no-changes'}`}>
                    {schemaDiff.has_changes ? '[!] ' : '[OK] '}{schemaDiff.summary}
                  </div>
                  
                  {schemaDiff.added_fields && schemaDiff.added_fields.length > 0 && (
                    <div className="diff-section diff-added">
                      <h4>[+] Added Fields</h4>
                      {schemaDiff.added_fields.map((f) => (
                        <div key={f.path} className="diff-field">
                          <span className="diff-path">{f.path}</span>
                          <span className="diff-type">{f.type}</span>
                        </div>
                      ))}
                    </div>
                  )}
                  
                  {schemaDiff.removed_fields && schemaDiff.removed_fields.length > 0 && (
                    <div className="diff-section diff-removed">
                      <h4>[-] Removed Fields</h4>
                      {schemaDiff.removed_fields.map((f) => (
                        <div key={f.path} className="diff-field">
                          <span className="diff-path">{f.path}</span>
                          <span className="diff-type">{f.type}</span>
                        </div>
                      ))}
                    </div>
                  )}
                  
                  {schemaDiff.type_changes && schemaDiff.type_changes.length > 0 && (
                    <div className="diff-section diff-changed">
                      <h4>[~] Type Changes</h4>
                      {schemaDiff.type_changes.map((c) => (
                        <div key={c.path} className="diff-field">
                          <span className="diff-path">{c.path}</span>
                          <span className="diff-type">{c.old_type} → {c.new_type}</span>
                        </div>
                      ))}
                    </div>
                  )}
                  
                  <button className="btn-secondary btn-small" onClick={() => setSchemaDiff(null)}>
                    Close
                  </button>
                </div>
              ) : selectedSchema ? (
                <div className="schema-detail">
                  <h3>{selectedSchema.operation}</h3>
                  <div className="schema-info">
                    <div><strong>Endpoint:</strong> {selectedSchema.method} {selectedSchema.endpoint}</div>
                    <div><strong>Samples:</strong> {selectedSchema.sample_count}</div>
                    <div><strong>Updated:</strong> {new Date(selectedSchema.updated_at).toLocaleString()}</div>
                  </div>
                  <h4>Schema Structure</h4>
                  <div className="schema-tree">
                    {renderFieldSchema(selectedSchema.root)}
                  </div>
                </div>
              ) : (
                <div className="empty-state">
                  <p className="text-muted">Select a schema to view details</p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {view === 'baselines' && (
        <div className="baselines-view">
          <div className="baselines-actions">
            <button 
              className="btn-primary btn-small" 
              onClick={handleSaveBaseline}
              disabled={loading}
            >
              [+] SAVE CURRENT AS BASELINE
            </button>
          </div>

          <div className="baselines-content">
            <div className="baselines-list-panel">
              <h3>Saved Baselines</h3>
              <div className="baselines-list">
                {baselines.length === 0 ? (
                  <div className="empty-state">
                    <p className="text-muted">No baselines saved</p>
                    <p className="text-muted">Save current data as a baseline to detect changes later</p>
                  </div>
                ) : (
                  baselines.map((baseline) => (
                    <div 
                      key={baseline.id}
                      className={`baseline-item ${selectedBaseline?.id === baseline.id ? 'selected' : ''}`}
                      onClick={() => { setSelectedBaseline(baseline); setBaselineDiff(null); }}
                    >
                      <div className="baseline-header">
                        <span className="baseline-name">{baseline.name}</span>
                        <button 
                          className="btn-icon"
                          onClick={(e) => { e.stopPropagation(); handleDeleteBaseline(baseline.id); }}
                        >
                          [x]
                        </button>
                      </div>
                      <div className="baseline-meta">
                        {new Date(baseline.created_at).toLocaleString()}
                      </div>
                      <div className="baseline-counts">
                        <span>{baseline.account_count} accounts</span>
                        <span>{baseline.payment_count} payments</span>
                        <span>{baseline.balance_count} balances</span>
                      </div>
                      <button 
                        className="btn-small btn-primary"
                        onClick={(e) => { e.stopPropagation(); handleCompareBaseline(baseline.id); }}
                      >
                        Compare with Current
                      </button>
                    </div>
                  ))
                )}
              </div>
            </div>

            <div className="baseline-detail-panel">
              {baselineDiff ? (
                <div className="baseline-diff">
                  <h3>Comparison Results</h3>
                  <div className={`diff-summary ${baselineDiff.has_changes ? 'has-changes' : 'no-changes'}`}>
                    {baselineDiff.has_changes ? '[!] Changes detected' : '[OK] No changes'}: {baselineDiff.summary}
                  </div>

                  {renderDataDiff('Accounts', baselineDiff.accounts)}
                  {renderDataDiff('Payments', baselineDiff.payments)}
                  {renderDataDiff('Balances', baselineDiff.balances)}
                  {renderDataDiff('External Accounts', baselineDiff.external_accounts)}
                  
                  <button className="btn-secondary btn-small" onClick={() => setBaselineDiff(null)}>
                    Close
                  </button>
                </div>
              ) : selectedBaseline ? (
                <div className="baseline-detail">
                  <h3>{selectedBaseline.name}</h3>
                  <div className="baseline-info">
                    <div><strong>Created:</strong> {new Date(selectedBaseline.created_at).toLocaleString()}</div>
                    <div><strong>Accounts:</strong> {selectedBaseline.account_count}</div>
                    <div><strong>Payments:</strong> {selectedBaseline.payment_count}</div>
                    <div><strong>Balances:</strong> {selectedBaseline.balance_count}</div>
                    <div><strong>External Accounts:</strong> {selectedBaseline.external_account_count}</div>
                  </div>
                </div>
              ) : (
                <div className="empty-state">
                  <p className="text-muted">Select a baseline to view details or compare</p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function renderDataDiff(name: string, diff: BaselineDiff['accounts']): JSX.Element | null {
  const hasChanges = (diff.added && diff.added.length > 0) || 
                     (diff.removed && diff.removed.length > 0) || 
                     (diff.modified && diff.modified.length > 0);
  
  if (!hasChanges && diff.baseline_count === diff.current_count) {
    return null;
  }

  return (
    <div className="data-diff-section">
      <h4>{name} ({diff.baseline_count} → {diff.current_count})</h4>
      
      {diff.added && diff.added.length > 0 && (
        <div className="diff-group diff-added">
          <span className="diff-label">Added:</span>
          {diff.added.map((ref) => (
            <span key={ref} className="diff-item">+{ref}</span>
          ))}
        </div>
      )}
      
      {diff.removed && diff.removed.length > 0 && (
        <div className="diff-group diff-removed">
          <span className="diff-label">Removed:</span>
          {diff.removed.map((ref) => (
            <span key={ref} className="diff-item">-{ref}</span>
          ))}
        </div>
      )}
      
      {diff.modified && diff.modified.length > 0 && (
        <div className="diff-group diff-modified">
          <span className="diff-label">Modified:</span>
          {diff.modified.map((item) => (
            <div key={item.reference} className="diff-modified-item">
              <span className="diff-ref">{item.reference}</span>
              {item.changes.map((change, i) => (
                <span key={i} className="diff-change">
                  {change.field}: {String(change.old_value)} → {String(change.new_value)}
                </span>
              ))}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// State Tab
function StateTab({ states, tasksTree }: { states: Record<string, unknown>; tasksTree: unknown }) {
  const [view, setView] = useState<'states' | 'tree'>('states');

  return (
    <div className="state-tab">
      <div className="tabs">
        <button className={`tab ${view === 'states' ? 'active' : ''}`} onClick={() => setView('states')}>
          Fetch States
        </button>
        <button className={`tab ${view === 'tree' ? 'active' : ''}`} onClick={() => setView('tree')}>
          Tasks Tree
        </button>
      </div>

      {view === 'states' && (
        <div className="states-list">
          {Object.keys(states).length === 0 ? (
            <div className="empty-state">
              <p className="text-muted">No state saved yet</p>
            </div>
          ) : (
            Object.entries(states).map(([key, value]) => (
              <div key={key} className="card state-card">
                <div className="card-title mono">{key}</div>
                <pre>{JSON.stringify(value, null, 2)}</pre>
              </div>
            ))
          )}
        </div>
      )}

      {view === 'tree' && (
        <div className="tasks-tree">
          {tasksTree !== null && tasksTree !== undefined ? (
            <pre>{JSON.stringify(tasksTree, null, 2)}</pre>
          ) : (
            <div className="empty-state">
              <p className="text-muted">No tasks tree available</p>
              <p className="text-muted">This is set when the connector is installed</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default App;
