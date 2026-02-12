import { useState, useEffect, useCallback, createContext, useContext, useRef } from 'react';
import { Routes, Route, useParams, useNavigate, Link, useLocation } from 'react-router-dom';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { 
  globalApi, 
  connectorApi, 
  setSelectedConnector,
  type GlobalStatus, 
  type ConnectorSummary, 
  type ConnectorStatus,
  type AvailableConnector,
  type Account, 
  type Payment, 
  type Balance, 
  type DebugEntry, 
  type PluginCall, 
  type HTTPRequest, 
  type ConnectorInfo, 
  type FileNode, 
  type SourceFile, 
  type SearchResult, 
  type TaskTreeSummary, 
  type TaskNodeSummary, 
  type TaskExecution, 
  type Snapshot, 
  type SnapshotStats, 
  type GenerateResult, 
  type InferredSchema, 
  type SchemaDiff, 
  type DataBaseline, 
  type BaselineDiff,
  type GenericServerStatus
} from './api';
import './App.css';

// =============================================================================
// UTILITY COMPONENTS & HELPERS
// =============================================================================

// Format relative time (e.g., "5 min ago")
function formatRelativeTime(date: Date | string): string {
  const now = new Date();
  const then = new Date(date);
  const diffMs = now.getTime() - then.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);

  if (diffSec < 5) return 'just now';
  if (diffSec < 60) return `${diffSec}s ago`;
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHour < 24) return `${diffHour}h ago`;
  if (diffDay < 7) return `${diffDay}d ago`;
  return then.toLocaleDateString();
}

// Tooltip component
function Tooltip({ text, children }: { text: string; children: React.ReactNode }) {
  return (
    <span className="tooltip-wrapper">
      {children}
      <span className="tooltip-text">{text}</span>
    </span>
  );
}

// Copy button component
function CopyButton({ text, label = 'Copy' }: { text: string; label?: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  return (
    <button 
      className={`btn-copy ${copied ? 'copied' : ''}`} 
      onClick={handleCopy}
      title={label}
    >
      {copied ? 'Copied!' : 'Copy'}
    </button>
  );
}

// Custom syntax theme with dark emerald background
const codeTheme: { [key: string]: React.CSSProperties } = {
  'code[class*="language-"]': {
    color: '#e6edf3',
    background: 'none',
    fontFamily: 'var(--term-font)',
    fontSize: '13px',
    textAlign: 'left',
    whiteSpace: 'pre',
    wordSpacing: 'normal',
    wordBreak: 'normal',
    wordWrap: 'normal',
    lineHeight: '1.6',
  },
  'pre[class*="language-"]': {
    color: '#e6edf3',
    background: 'transparent',
    fontFamily: 'var(--term-font)',
    fontSize: '13px',
    textAlign: 'left',
    whiteSpace: 'pre',
    wordSpacing: 'normal',
    wordBreak: 'normal',
    wordWrap: 'normal',
    lineHeight: '1.6',
    padding: '12px',
    margin: '0',
    overflow: 'auto',
    borderRadius: '0',
  },
  'comment': { color: '#6e8a82' },
  'prolog': { color: '#6e8a82' },
  'doctype': { color: '#6e8a82' },
  'cdata': { color: '#6e8a82' },
  'punctuation': { color: '#7daa9c' },
  'property': { color: '#55d799' },
  'tag': { color: '#55d799' },
  'boolean': { color: '#79c0ff' },
  'number': { color: '#79c0ff' },
  'constant': { color: '#79c0ff' },
  'symbol': { color: '#79c0ff' },
  'deleted': { color: '#ffa198' },
  'selector': { color: '#55d799' },
  'attr-name': { color: '#55d799' },
  'string': { color: '#a5d6ff' },
  'char': { color: '#a5d6ff' },
  'builtin': { color: '#ffa657' },
  'inserted': { color: '#55d799' },
  'operator': { color: '#ff7b72' },
  'entity': { color: '#ffa657', cursor: 'help' },
  'url': { color: '#a5d6ff' },
  'variable': { color: '#ffa657' },
  'atrule': { color: '#79c0ff' },
  'attr-value': { color: '#a5d6ff' },
  'function': { color: '#d2a8ff' },
  'class-name': { color: '#ffa657' },
  'keyword': { color: '#ff7b72' },
  'regex': { color: '#a5d6ff' },
  'important': { color: '#ff7b72', fontWeight: 'bold' },
  'bold': { fontWeight: 'bold' },
  'italic': { fontStyle: 'italic' },
};

// Syntax-highlighted code block
function CodeBlock({ children, language = 'json' }: { children: string; language?: string }) {
  return (
    <SyntaxHighlighter 
      language={language} 
      style={codeTheme}
      customStyle={{
        margin: 0,
        padding: '12px',
        background: 'transparent',
        borderRadius: '0',
        fontSize: '13px',
        lineHeight: '1.6',
        overflow: 'auto',
      }}
      wrapLongLines={true}
    >
      {children}
    </SyntaxHighlighter>
  );
}

// Confirmation Modal
interface ConfirmModalProps {
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: 'danger' | 'warning' | 'default';
  onConfirm: () => void;
  onCancel: () => void;
}

function ConfirmModal({ title, message, confirmText = 'Confirm', cancelText = 'Cancel', variant = 'default', onConfirm, onCancel }: ConfirmModalProps) {
  // Handle Escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel();
      if (e.key === 'Enter') onConfirm();
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onConfirm, onCancel]);

  return (
    <div className="modal-overlay" onClick={onCancel}>
      <div className="modal modal-confirm" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h3>{title}</h3>
        </div>
        <div className="modal-body">
          <p>{message}</p>
        </div>
        <div className="modal-footer">
          <button className="btn-secondary" onClick={onCancel}>
            {cancelText}
          </button>
          <button className={`btn-${variant === 'danger' ? 'danger' : 'primary'}`} onClick={onConfirm} autoFocus>
            {confirmText}
          </button>
        </div>
      </div>
    </div>
  );
}

// Skeleton loader
function Skeleton({ rows = 3, type = 'list' }: { rows?: number; type?: 'list' | 'card' | 'table' }) {
  if (type === 'card') {
    return (
      <div className="skeleton-card">
        <div className="skeleton-line skeleton-title" />
        <div className="skeleton-line" />
        <div className="skeleton-line skeleton-short" />
      </div>
    );
  }
  if (type === 'table') {
    return (
      <div className="skeleton-table">
        {Array.from({ length: rows }).map((_, i) => (
          <div key={i} className="skeleton-row">
            <div className="skeleton-cell" />
            <div className="skeleton-cell" />
            <div className="skeleton-cell skeleton-short" />
          </div>
        ))}
      </div>
    );
  }
  return (
    <div className="skeleton-list">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="skeleton-line" style={{ width: `${70 + Math.random() * 30}%` }} />
      ))}
    </div>
  );
}

// Progress bar component
function ProgressBar({ current, total, label }: { current: number; total: number; label?: string }) {
  const percent = total > 0 ? Math.min(100, (current / total) * 100) : 0;
  return (
    <div className="progress-bar-container">
      {label && <span className="progress-label">{label}</span>}
      <div className="progress-bar">
        <div className="progress-fill" style={{ width: `${percent}%` }} />
      </div>
      <span className="progress-text">{current}/{total}</span>
    </div>
  );
}

// Context for the selected connector
interface ConnectorContextType {
  connectorId: string | null;
  connectorStatus: ConnectorStatus | null;
  setConnectorId: (id: string | null) => void;
  refreshConnectorStatus: () => Promise<void>;
}

const ConnectorContext = createContext<ConnectorContextType>({
  connectorId: null,
  connectorStatus: null,
  setConnectorId: () => {},
  refreshConnectorStatus: async () => {},
});

const useConnector = () => useContext(ConnectorContext);

type Tab = 'dashboard' | 'accounts' | 'payments' | 'balances' | 'tasks' | 'debug' | 'snapshots' | 'analysis' | 'code' | 'state';

const VALID_TABS: Tab[] = ['dashboard', 'accounts', 'payments', 'balances', 'tasks', 'debug', 'snapshots', 'analysis', 'code', 'state'];

function isValidTab(tab: string | undefined): tab is Tab {
  return tab !== undefined && VALID_TABS.includes(tab as Tab);
}

// Root App component that sets up routes
function App() {
  return (
    <Routes>
      <Route path="/" element={<AppContent />} />
      <Route path="/debug" element={<AppContent />} />
      <Route path="/c/:connectorId" element={<AppContent />} />
      <Route path="/c/:connectorId/:tab" element={<AppContent />} />
      <Route path="/c/:connectorId/:tab/:itemId" element={<AppContent />} />
      <Route path="*" element={<AppContent />} />
    </Routes>
  );
}

// Main app content with routing
function AppContent() {
  const navigate = useNavigate();
  const location = useLocation();
  const { connectorId: urlConnectorId, tab: urlTab } = useParams<{ connectorId?: string; tab?: string }>();
  
  // Derive navigation state from URL
  const selectedConnectorId = urlConnectorId || null;
  // Handle /debug route specially
  const isDebugRoute = location.pathname === '/debug' || location.pathname.startsWith('/debug');
  const tab: Tab = isDebugRoute ? 'debug' : (isValidTab(urlTab) ? urlTab : 'dashboard');

  const [connectorStatus, setConnectorStatus] = useState<ConnectorStatus | null>(null);
  const [globalStatus, setGlobalStatus] = useState<GlobalStatus | null>(null);
  const [connectors, setConnectors] = useState<ConnectorSummary[]>([]);
  const [availableConnectors, setAvailableConnectors] = useState<AvailableConnector[]>([]);
  const [showCreateModal, setShowCreateModal] = useState(false);
  
  const [loading, setLoading] = useState(false);
  const [initialLoading, setInitialLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
  // Data (connector-specific)
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [payments, setPayments] = useState<Payment[]>([]);
  const [balances, setBalances] = useState<Balance[]>([]);
  const [states, setStates] = useState<Record<string, unknown>>({});
  const [tasksTree, setTasksTree] = useState<unknown>(null);
  
  // Debug data (global)
  const [debugLogs, setDebugLogs] = useState<DebugEntry[]>([]);
  const [pluginCalls, setPluginCalls] = useState<PluginCall[]>([]);
  const [httpRequests, setHttpRequests] = useState<HTTPRequest[]>([]);
  const [httpCaptureEnabled, setHttpCaptureEnabled] = useState(true);
  
  // Generic server state
  const [genericServerStatus, setGenericServerStatus] = useState<GenericServerStatus | null>(null);
  
  // Theme state
  const [theme, setTheme] = useState<'dark' | 'light'>(() => {
    // Check localStorage or system preference
    const saved = localStorage.getItem('workbench-theme');
    if (saved === 'light' || saved === 'dark') return saved;
    return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
  });

  // Apply theme to document
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('workbench-theme', theme);
  }, [theme]);

  const toggleTheme = () => setTheme(t => t === 'dark' ? 'light' : 'dark');
  
  // Snapshot modal state
  const [showSnapshotModal, setShowSnapshotModal] = useState(false);
  const [snapshotCaptureId, setSnapshotCaptureId] = useState<string | null>(null);

  // Confirmation modal state
  const [confirmModal, setConfirmModal] = useState<{
    title: string;
    message: string;
    variant?: 'danger' | 'warning' | 'default';
    onConfirm: () => void;
  } | null>(null);

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Ctrl/Cmd + number for tab switching
      if ((e.ctrlKey || e.metaKey) && e.key >= '1' && e.key <= '9') {
        e.preventDefault();
        const tabIndex = parseInt(e.key) - 1;
        if (tabIndex < VALID_TABS.length) {
          const targetTab = VALID_TABS[tabIndex];
          // Only switch to connector tabs if a connector is selected
          if (targetTab === 'debug') {
            navigate(selectedConnectorId ? `/c/${selectedConnectorId}/debug` : '/debug');
          } else if (selectedConnectorId) {
            navigate(`/c/${selectedConnectorId}/${targetTab}`);
          }
        }
      }
      // Ctrl/Cmd + N for new connector
      if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
        e.preventDefault();
        setShowCreateModal(true);
      }
      // Escape to close modals
      if (e.key === 'Escape') {
        if (showCreateModal) setShowCreateModal(false);
        if (showSnapshotModal) {
          setShowSnapshotModal(false);
          setSnapshotCaptureId(null);
        }
        if (confirmModal) setConfirmModal(null);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [selectedConnectorId, showCreateModal, showSnapshotModal, confirmModal, navigate]);

  // Update selected connector in api module
  useEffect(() => {
    setSelectedConnector(selectedConnectorId);
  }, [selectedConnectorId]);

  const refreshGlobalStatus = useCallback(async () => {
    try {
      const [status, connectorsList, available, genericStatus] = await Promise.all([
        globalApi.getStatus(),
        globalApi.listConnectors(),
        globalApi.getAvailableConnectors(),
        globalApi.getGenericServerStatus(),
      ]);
      setGlobalStatus(status);
      setConnectors(connectorsList.connectors || []);
      setAvailableConnectors(available.connectors || []);
      setGenericServerStatus(genericStatus);
      setInitialLoading(false);
    } catch (e) {
      console.error('Failed to fetch global status:', e);
      setInitialLoading(false);
    }
  }, []);

  const refreshConnectorStatus = useCallback(async () => {
    if (!selectedConnectorId) {
      setConnectorStatus(null);
      return;
    }
    try {
      const api = connectorApi(selectedConnectorId);
      const status = await api.getStatus();
      setConnectorStatus(status);
    } catch (e) {
      console.error('Failed to fetch connector status:', e);
    }
  }, [selectedConnectorId]);

  const refreshConnectorData = useCallback(async () => {
    if (!selectedConnectorId) {
      setAccounts([]);
      setPayments([]);
      setBalances([]);
      setStates({});
      setTasksTree(null);
      return;
    }
    try {
      const api = connectorApi(selectedConnectorId);
      const [acc, pay, bal, st, tree] = await Promise.all([
        api.getAccounts(),
        api.getPayments(),
        api.getBalances(),
        api.getStates(),
        api.getTasksTree(),
      ]);
      setAccounts(acc.accounts || []);
      setPayments(pay.payments || []);
      setBalances(bal.balances || []);
      setStates(st.states || {});
      setTasksTree(tree.tasks_tree);
    } catch (e) {
      console.error('Failed to refresh connector data:', e);
    }
  }, [selectedConnectorId]);

  const refreshDebugData = useCallback(async () => {
    try {
      const [logs, calls, httpReqs, httpStatus] = await Promise.all([
        globalApi.getDebugLogs(100),
        globalApi.getPluginCalls(50),
        globalApi.getHTTPRequests(50),
        globalApi.getHTTPCaptureStatus(),
      ]);
      setDebugLogs(logs.entries || []);
      setPluginCalls(calls.plugin_calls || []);
      setHttpRequests(httpReqs.requests || []);
      setHttpCaptureEnabled(httpStatus.enabled);
    } catch (e) {
      console.error('Failed to refresh debug data:', e);
    }
  }, []);

  useEffect(() => {
    refreshGlobalStatus();
    refreshDebugData();
    const interval = setInterval(() => {
      refreshGlobalStatus();
      refreshDebugData();
    }, 3000);
    return () => clearInterval(interval);
  }, [refreshGlobalStatus, refreshDebugData]);

  useEffect(() => {
    refreshConnectorStatus();
    refreshConnectorData();
    if (selectedConnectorId) {
      const interval = setInterval(() => {
        refreshConnectorStatus();
        refreshConnectorData();
      }, 2000);
      return () => clearInterval(interval);
    }
  }, [selectedConnectorId, refreshConnectorStatus, refreshConnectorData]);

  const runAction = async (action: () => Promise<unknown>, successMsg?: string) => {
    setLoading(true);
    setError(null);
    try {
      await action();
      if (successMsg) setError(null);
      await refreshConnectorStatus();
      await refreshConnectorData();
      await refreshGlobalStatus();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateConnector = async (provider: string, name: string, config: Record<string, unknown>) => {
    setLoading(true);
    setError(null);
    try {
      const result = await globalApi.createConnector({ provider, name, config });
      await refreshGlobalStatus();
      setShowCreateModal(false);
      navigate(`/c/${result.id}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to instantiate connector');
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteConnector = (id: string) => {
    const connector = connectors.find(c => c.id === id);
    setConfirmModal({
      title: 'Delete Connector',
      message: `Are you sure you want to delete "${connector?.name || connector?.provider || id.slice(0, 8)}"? This action cannot be undone.`,
      variant: 'danger',
      onConfirm: async () => {
        setConfirmModal(null);
        setLoading(true);
        setError(null);
        try {
          await connectorApi(id).delete();
          if (selectedConnectorId === id) {
            navigate('/');
          }
          await refreshGlobalStatus();
        } catch (e) {
          setError(e instanceof Error ? e.message : 'Failed to delete connector');
        } finally {
          setLoading(false);
        }
      }
    });
  };

  const contextValue: ConnectorContextType = {
    connectorId: selectedConnectorId,
    connectorStatus,
    setConnectorId: (id: string | null) => {
      if (id) {
        navigate(`/c/${id}`);
      } else {
        navigate('/');
      }
    },
    refreshConnectorStatus,
  };

  // Get initials from provider name for sidebar icons
  const getProviderInitials = (provider: string) => {
    return provider.slice(0, 2).toUpperCase();
  };

  return (
    <ConnectorContext.Provider value={contextValue}>
      <div className="app">
        <aside className="connector-sidebar">
          <div className="sidebar-header">
            <Tooltip text="Home - View all connectors">
              <Link 
                to="/"
                className={`sidebar-logo ${!selectedConnectorId ? 'active' : ''}`}
              >
                CW
              </Link>
            </Tooltip>
          </div>
          <div className="sidebar-connectors">
            {connectors.map((c) => (
              <Tooltip key={c.id} text={`${c.provider}: ${c.name || c.id.slice(0, 8)}${c.installed ? ' [installed]' : ''}`}>
                <Link
                  to={`/c/${c.id}`}
                  className={`sidebar-connector ${selectedConnectorId === c.id ? 'active' : ''} ${c.installed ? 'installed' : ''}`}
                >
                  <span className="connector-initials">{getProviderInitials(c.provider)}</span>
                  {!c.installed && <span className="connector-uninstalled-dot" />}
                </Link>
              </Tooltip>
            ))}
          </div>
          <div className="sidebar-footer">
            <Tooltip text={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}>
              <button className="sidebar-theme" onClick={toggleTheme}>
                {theme === 'dark' ? '☀' : '☾'}
              </button>
            </Tooltip>
            <Tooltip text="Instantiate new connector (Ctrl+N)">
              <button className="sidebar-add" onClick={() => setShowCreateModal(true)}>
                +
              </button>
            </Tooltip>
          </div>
        </aside>

        <div className="main-content">
        {selectedConnectorId && connectorStatus && (
          <div className="connector-header">
            <span className="provider-badge">{connectorStatus.provider}</span>
            <Tooltip text="Click to copy">
              <code className="connector-id clickable" onClick={() => navigator.clipboard.writeText(connectorStatus.connector_id)}>
                {connectorStatus.connector_id}
              </code>
            </Tooltip>
            <span className={`status-badge ${connectorStatus.installed ? 'installed' : 'not-installed'}`}>
              {connectorStatus.installed ? 'INSTALLED' : 'NOT INSTALLED'}
            </span>
            <div className="connector-actions">
              {!connectorStatus.installed && (
                <Tooltip text="Install the connector to enable data fetching">
                  <button 
                    className="btn-primary btn-small" 
                    onClick={() => runAction(() => connectorApi(selectedConnectorId).install())}
                    disabled={loading}
                  >
                    {loading ? <span className="spinner" /> : 'Install'}
                  </button>
                </Tooltip>
              )}
              {connectorStatus.installed && (
                <Tooltip text="Uninstall but keep configuration">
                  <button 
                    className="btn-secondary btn-small" 
                    onClick={() => runAction(() => connectorApi(selectedConnectorId).uninstall())}
                    disabled={loading}
                  >
                    {loading ? <span className="spinner" /> : 'Uninstall'}
                  </button>
                </Tooltip>
              )}
              <Tooltip text="Permanently delete this connector">
                <button 
                  className="btn-danger btn-small" 
                  onClick={() => handleDeleteConnector(selectedConnectorId)}
                  disabled={loading}
                >
                  Delete
                </button>
              </Tooltip>
            </div>
          </div>
        )}

        {selectedConnectorId && (
          <nav className="tabs">
            <Tooltip text="Ctrl+1">
              <Link to={`/c/${selectedConnectorId}/dashboard`} className={`tab ${tab === 'dashboard' ? 'active' : ''}`}>
                Dashboard
              </Link>
            </Tooltip>
            <Tooltip text="Ctrl+2">
              <Link to={`/c/${selectedConnectorId}/accounts`} className={`tab ${tab === 'accounts' ? 'active' : ''}`}>
                Accounts {accounts.length > 0 && <span className="tab-count">{accounts.length}</span>}
              </Link>
            </Tooltip>
            <Tooltip text="Ctrl+3">
              <Link to={`/c/${selectedConnectorId}/payments`} className={`tab ${tab === 'payments' ? 'active' : ''}`}>
                Payments {payments.length > 0 && <span className="tab-count">{payments.length}</span>}
              </Link>
            </Tooltip>
            <Tooltip text="Ctrl+4">
              <Link to={`/c/${selectedConnectorId}/balances`} className={`tab ${tab === 'balances' ? 'active' : ''}`}>
                Balances {balances.length > 0 && <span className="tab-count">{balances.length}</span>}
              </Link>
            </Tooltip>
            <Tooltip text="Ctrl+5">
              <Link to={`/c/${selectedConnectorId}/tasks`} className={`tab ${tab === 'tasks' ? 'active' : ''}`}>
                Tasks
              </Link>
            </Tooltip>
            <Tooltip text="Ctrl+6">
              <Link to={`/c/${selectedConnectorId}/debug`} className={`tab ${tab === 'debug' ? 'active' : ''}`}>
                Debug {globalStatus && globalStatus.debug.error_count > 0 && (
                  <span className="tab-count error">{globalStatus.debug.error_count}</span>
                )}
              </Link>
            </Tooltip>
            <Tooltip text="Ctrl+7">
              <Link to={`/c/${selectedConnectorId}/snapshots`} className={`tab ${tab === 'snapshots' ? 'active' : ''}`}>
                Snapshots
              </Link>
            </Tooltip>
            <Tooltip text="Ctrl+8">
              <Link to={`/c/${selectedConnectorId}/analysis`} className={`tab ${tab === 'analysis' ? 'active' : ''}`}>
                Analysis
              </Link>
            </Tooltip>
            <Tooltip text="Ctrl+9">
              <Link to={`/c/${selectedConnectorId}/code`} className={`tab ${tab === 'code' ? 'active' : ''}`}>
                Code
              </Link>
            </Tooltip>
            <Link to={`/c/${selectedConnectorId}/state`} className={`tab ${tab === 'state' ? 'active' : ''}`}>
              State
            </Link>
          </nav>
        )}

        {error && (
          <div className="error-banner">
            <span>{error}</span>
            <button onClick={() => setError(null)}>×</button>
          </div>
        )}

        <main className="main">
          {initialLoading ? (
            <div className="loading-container">
              <Skeleton rows={4} type="card" />
              <Skeleton rows={4} type="card" />
            </div>
          ) : !selectedConnectorId && !isDebugRoute ? (
            <NoConnectorView 
              connectors={connectors}
              availableConnectors={availableConnectors}
              onSelect={(id) => navigate(`/c/${id}`)}
              onCreate={() => setShowCreateModal(true)}
              onDelete={handleDeleteConnector}
            />
          ) : (
            <>
              {tab === 'dashboard' && selectedConnectorId && (
                <DashboardTab
                  status={connectorStatus}
                  loading={loading}
                  onFetchAccounts={() => runAction(() => connectorApi(selectedConnectorId).fetchAccounts())}
                  onFetchPayments={() => runAction(() => connectorApi(selectedConnectorId).fetchPayments())}
                  onFetchBalances={() => runAction(() => connectorApi(selectedConnectorId).fetchBalances())}
                  onFetchExternalAccounts={() => runAction(() => connectorApi(selectedConnectorId).fetchExternalAccounts())}
                  onFetchAll={() => runAction(() => connectorApi(selectedConnectorId).fetchAll())}
                  onReset={() => runAction(() => connectorApi(selectedConnectorId).reset())}
                />
              )}
              {tab === 'accounts' && (
                <AccountsTab 
                  accounts={accounts} 
                  onFetch={selectedConnectorId ? () => runAction(() => connectorApi(selectedConnectorId).fetchAccounts()) : undefined}
                />
              )}
              {tab === 'payments' && (
                <PaymentsTab 
                  payments={payments}
                  onFetch={selectedConnectorId ? () => runAction(() => connectorApi(selectedConnectorId).fetchPayments()) : undefined}
                />
              )}
              {tab === 'balances' && (
                <BalancesTab 
                  balances={balances}
                  onFetch={selectedConnectorId ? () => runAction(() => connectorApi(selectedConnectorId).fetchBalances()) : undefined}
                />
              )}
              {tab === 'tasks' && <TasksTab />}
              {tab === 'debug' && (
                <DebugTab 
                  logs={debugLogs} 
                  pluginCalls={pluginCalls}
                  httpRequests={httpRequests}
                  httpCaptureEnabled={httpCaptureEnabled}
                  onClear={() => runAction(() => globalApi.clearDebug())}
                  onToggleCapture={() => runAction(async () => {
                    if (httpCaptureEnabled) {
                      await globalApi.disableHTTPCapture();
                    } else {
                      await globalApi.enableHTTPCapture();
                    }
                  })}
                  onSaveSnapshot={selectedConnectorId ? (captureId) => {
                    setSnapshotCaptureId(captureId);
                    setShowSnapshotModal(true);
                  } : undefined}
                  genericServerStatus={genericServerStatus}
                  connectors={connectors}
                  onSetGenericConnector={async (connectorId: string, apiKey?: string) => {
                    try {
                      const newStatus = await globalApi.setGenericServerConnector(connectorId, apiKey);
                      setGenericServerStatus(newStatus);
                    } catch (e) {
                      console.error('Failed to set generic connector:', e);
                    }
                  }}
                />
              )}
              {tab === 'snapshots' && <SnapshotsTab />}
              {tab === 'analysis' && <AnalysisTab />}
              {tab === 'code' && <CodeTab />}
              {tab === 'state' && <StateTab states={states} tasksTree={tasksTree} />}
            </>
          )}
        </main>

        {showCreateModal && (
          <CreateConnectorModal
            availableConnectors={availableConnectors}
            onClose={() => setShowCreateModal(false)}
            onCreate={handleCreateConnector}
          />
        )}

        {showSnapshotModal && snapshotCaptureId && selectedConnectorId && (
          <SaveSnapshotModal
            connectorId={selectedConnectorId}
            captureId={snapshotCaptureId}
            onClose={() => {
              setShowSnapshotModal(false);
              setSnapshotCaptureId(null);
            }}
            onSaved={() => {
              setShowSnapshotModal(false);
              setSnapshotCaptureId(null);
              navigate(`/c/${selectedConnectorId}/snapshots`);
            }}
          />
        )}

        {confirmModal && (
          <ConfirmModal
            title={confirmModal.title}
            message={confirmModal.message}
            variant={confirmModal.variant}
            confirmText="Delete"
            onConfirm={confirmModal.onConfirm}
            onCancel={() => setConfirmModal(null)}
          />
        )}
        </div>{/* end main-content */}
      </div>
    </ConnectorContext.Provider>
  );
}

// No Connector Selected View
interface NoConnectorViewProps {
  connectors: ConnectorSummary[];
  availableConnectors: AvailableConnector[];
  onSelect: (id: string) => void;
  onCreate: () => void;
  onDelete: (id: string) => void;
}

function NoConnectorView({ connectors, availableConnectors, onSelect, onCreate, onDelete }: NoConnectorViewProps) {
  return (
    <div className="no-connector-view">
      <div className="no-connector-header">
        <h2>Connector Instances</h2>
        <button className="btn-primary" onClick={onCreate}>
          + Instantiate New Connector
        </button>
      </div>

      {connectors.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">[C]</div>
          <p>No connector instances yet</p>
          <p className="text-muted">Instantiate a connector to get started</p>
          <button className="btn-primary" onClick={onCreate} style={{ marginTop: '16px' }}>
            + Instantiate Your First Connector
          </button>
        </div>
      ) : (
        <div className="connectors-grid">
          {connectors.map((c) => (
            <div key={c.id} className="connector-card" onClick={() => onSelect(c.id)}>
              <div className="connector-card-header">
                <span className="provider-badge">{c.provider}</span>
                <span className={`status-badge ${c.installed ? 'installed' : 'not-installed'}`}>
                  {c.installed ? 'INSTALLED' : 'NOT INSTALLED'}
                </span>
              </div>
              <div className="connector-card-body">
                <div className="connector-name">{c.name || c.id.slice(0, 8)}</div>
                <code className="connector-id-small">{c.connector_id}</code>
              </div>
              <div className="connector-card-stats">
                {c.accounts_count !== undefined && <span>{c.accounts_count} accounts</span>}
                {c.payments_count !== undefined && <span>{c.payments_count} payments</span>}
                {c.balances_count !== undefined && <span>{c.balances_count} balances</span>}
              </div>
              <div className="connector-card-footer">
                <Tooltip text={new Date(c.created_at).toLocaleString()}>
                  <span className="connector-date">{formatRelativeTime(c.created_at)}</span>
                </Tooltip>
                <button 
                  className="btn-danger btn-small"
                  onClick={(e) => { e.stopPropagation(); onDelete(c.id); }}
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="available-connectors">
        <h3>Available Connector Types ({availableConnectors.length})</h3>
        <div className="available-list">
          {availableConnectors.map((c) => (
            <div key={c.provider} className="available-item">
              <span className="available-provider">{c.provider}</span>
              <span className={`available-type ${c.plugin_type === 1 ? 'type-openbanking' : 'type-psp'}`}>
                {formatPluginType(c.plugin_type)}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// Create Connector Modal
interface CreateConnectorModalProps {
  availableConnectors: AvailableConnector[];
  onClose: () => void;
  onCreate: (provider: string, name: string, config: Record<string, unknown>) => Promise<void>;
}

function CreateConnectorModal({ availableConnectors, onClose, onCreate }: CreateConnectorModalProps) {
  const [selectedProvider, setSelectedProvider] = useState('');
  const [name, setName] = useState('');
  const [config, setConfig] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const selectedConnector = availableConnectors.find(c => c.provider === selectedProvider);

  const handleProviderChange = (provider: string) => {
    setSelectedProvider(provider);
    setConfig({});
  };

  const handleConfigChange = (key: string, value: string) => {
    setConfig(prev => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async () => {
    if (!selectedProvider) {
      setError('Please select a connector type');
      return;
    }
    
    // Check required fields
    if (selectedConnector) {
      for (const [key, param] of Object.entries(selectedConnector.config)) {
        if (param.required && !config[key]) {
          setError(`${key} is required`);
          return;
        }
      }
    }

    setLoading(true);
    setError(null);
    try {
      await onCreate(selectedProvider, name, config);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to instantiate connector');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal modal-large" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h3>Instantiate Connector</h3>
          <button className="btn-icon" onClick={onClose}>×</button>
        </div>
        <div className="modal-body">
          {error && <div className="alert alert-danger">{error}</div>}
          
          <div className="form-group">
            <label>Connector Type *</label>
            <div className="connector-type-grid">
              {availableConnectors.map(c => (
                <div 
                  key={c.provider} 
                  className={`connector-type-tile ${selectedProvider === c.provider ? 'selected' : ''}`}
                  onClick={() => handleProviderChange(c.provider)}
                >
                  <span className="tile-provider">{c.provider}</span>
                  <span className={`tile-type ${c.plugin_type === 1 ? 'type-openbanking' : 'type-psp'}`}>
                    {formatPluginType(c.plugin_type)}
                  </span>
                </div>
              ))}
            </div>
          </div>

          {selectedProvider && (
            <div className="form-group">
              <label>Instance Name (optional)</label>
              <input
                type="text"
                value={name}
                onChange={e => setName(e.target.value)}
                placeholder="e.g., my-stripe-sandbox"
              />
            </div>
          )}

          {selectedConnector && (
            <div className="config-section">
              <h4>Configuration</h4>
              {Object.entries(selectedConnector.config).map(([key, param]) => (
                <div key={key} className="form-group">
                  <label>
                    {key}
                    {param.required && <span className="required">*</span>}
                    <span className="config-type">({param.dataType})</span>
                  </label>
                  {param.dataType === 'boolean' ? (
                    <select 
                      value={config[key] || param.defaultValue || ''} 
                      onChange={e => handleConfigChange(key, e.target.value)}
                    >
                      <option value="">Select...</option>
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </select>
                  ) : key.toLowerCase().includes('privatekey') || key.toLowerCase().includes('certificate') || key.toLowerCase().includes('pem') ? (
                    <textarea
                      value={config[key] || ''}
                      onChange={e => handleConfigChange(key, e.target.value)}
                      placeholder={param.defaultValue || 'Paste PEM content here...'}
                      rows={6}
                      className="config-textarea"
                    />
                  ) : (
                    <input
                      type={param.dataType === 'password' ? 'password' : 'text'}
                      value={config[key] || ''}
                      onChange={e => handleConfigChange(key, e.target.value)}
                      placeholder={param.defaultValue || ''}
                    />
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
        <div className="modal-footer">
          <button className="btn-secondary" onClick={onClose} disabled={loading}>
            Cancel
          </button>
          <button className="btn-primary" onClick={handleSubmit} disabled={loading || !selectedProvider}>
            {loading ? <span className="spinner" /> : 'Instantiate'}
          </button>
        </div>
      </div>
    </div>
  );
}

// Dashboard Tab
interface DashboardTabProps {
  status: ConnectorStatus | null;
  loading: boolean;
  onFetchAccounts: () => void;
  onFetchPayments: () => void;
  onFetchBalances: () => void;
  onFetchExternalAccounts: () => void;
  onFetchAll: () => void;
  onReset: () => void;
}

function DashboardTab({ status, loading, onFetchAccounts, onFetchPayments, onFetchBalances, onFetchExternalAccounts, onFetchAll, onReset }: DashboardTabProps) {
  return (
    <div className="dashboard">
      <section className="actions-section">
        <h2>Actions</h2>
        <div className="actions-grid">
          <Tooltip text="Fetch all data types in sequence">
            <button className="btn-primary btn-large" onClick={onFetchAll} disabled={loading}>
              {loading ? <span className="spinner" /> : '>'} Run Full Cycle
            </button>
          </Tooltip>
          <Tooltip text="Fetch account data from the provider">
            <button className="btn-secondary" onClick={onFetchAccounts} disabled={loading}>
              Fetch Accounts
            </button>
          </Tooltip>
          <Tooltip text="Fetch payment transactions">
            <button className="btn-secondary" onClick={onFetchPayments} disabled={loading}>
              Fetch Payments
            </button>
          </Tooltip>
          <Tooltip text="Fetch current balances">
            <button className="btn-secondary" onClick={onFetchBalances} disabled={loading}>
              Fetch Balances
            </button>
          </Tooltip>
          <Tooltip text="Fetch external/counterparty accounts">
            <button className="btn-secondary" onClick={onFetchExternalAccounts} disabled={loading}>
              Fetch Ext. Accounts
            </button>
          </Tooltip>
          <Tooltip text="Clear all fetched data and reset state">
            <button className="btn-danger" onClick={onReset} disabled={loading}>
              Reset All
            </button>
          </Tooltip>
        </div>
      </section>

      <section className="stats-section">
        <h2>Storage</h2>
        <div className="grid grid-4">
          <StatCard label="Accounts" value={status?.storage?.accounts_count ?? 0} icon="[A]" />
          <StatCard label="Payments" value={status?.storage?.payments_count ?? 0} icon="[P]" />
          <StatCard label="Balances" value={status?.storage?.balances_count ?? 0} icon="[$]" />
          <StatCard label="External Accounts" value={status?.storage?.external_accounts_count ?? 0} icon="[E]" />
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


// Accounts Tab
interface AccountsTabProps {
  accounts: Account[];
  onFetch?: () => void;
}

function AccountsTab({ accounts, onFetch }: AccountsTabProps) {
  const [selected, setSelected] = useState<Account | null>(null);
  const [selectedIndex, setSelectedIndex] = useState<number>(-1);
  const listRef = useRef<HTMLTableSectionElement>(null);

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (accounts.length === 0) return;
      
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        const newIndex = Math.min(selectedIndex + 1, accounts.length - 1);
        setSelectedIndex(newIndex);
        setSelected(accounts[newIndex]);
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        const newIndex = Math.max(selectedIndex - 1, 0);
        setSelectedIndex(newIndex);
        setSelected(accounts[newIndex]);
      } else if (e.key === 'Escape') {
        setSelected(null);
        setSelectedIndex(-1);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [accounts, selectedIndex]);

  if (accounts.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">[A]</div>
        <p>No accounts fetched yet</p>
        <p className="text-muted">Fetch accounts from the connected provider</p>
        {onFetch && (
          <button className="btn-primary" onClick={onFetch} style={{ marginTop: '16px' }}>
            Fetch Accounts
          </button>
        )}
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
          <tbody ref={listRef}>
            {accounts.map((acc, index) => (
              <tr 
                key={acc.reference} 
                onClick={() => { setSelected(acc); setSelectedIndex(index); }}
                className={selectedIndex === index ? 'selected' : ''}
              >
                <td className="mono">{acc.reference}</td>
                <td>{acc.name || '-'}</td>
                <td>{acc.default_asset || '-'}</td>
                <td>
                  <Tooltip text={new Date(acc.created_at).toLocaleString()}>
                    <span>{formatRelativeTime(acc.created_at)}</span>
                  </Tooltip>
                </td>
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
              <div className="card-actions">
                <CopyButton text={JSON.stringify(selected, null, 2)} label="Copy JSON" />
                <button className="btn-icon" onClick={() => { setSelected(null); setSelectedIndex(-1); }} title="Close (Esc)">×</button>
              </div>
            </div>
            <CodeBlock>{JSON.stringify(selected, null, 2)}</CodeBlock>
          </div>
        </div>
      )}
    </div>
  );
}

// Payments Tab
interface PaymentsTabProps {
  payments: Payment[];
  onFetch?: () => void;
}

function PaymentsTab({ payments, onFetch }: PaymentsTabProps) {
  const [selected, setSelected] = useState<Payment | null>(null);
  const [selectedIndex, setSelectedIndex] = useState<number>(-1);

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (payments.length === 0) return;
      
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        const newIndex = Math.min(selectedIndex + 1, payments.length - 1);
        setSelectedIndex(newIndex);
        setSelected(payments[newIndex]);
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        const newIndex = Math.max(selectedIndex - 1, 0);
        setSelectedIndex(newIndex);
        setSelected(payments[newIndex]);
      } else if (e.key === 'Escape') {
        setSelected(null);
        setSelectedIndex(-1);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [payments, selectedIndex]);

  if (payments.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">[P]</div>
        <p>No payments fetched yet</p>
        <p className="text-muted">Fetch payment transactions from the provider</p>
        {onFetch && (
          <button className="btn-primary" onClick={onFetch} style={{ marginTop: '16px' }}>
            Fetch Payments
          </button>
        )}
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
            {payments.map((pay, index) => (
              <tr 
                key={pay.reference}
                onClick={() => { setSelected(pay); setSelectedIndex(index); }}
                className={selectedIndex === index ? 'selected' : ''}
              >
                <td className="mono">{pay.reference}</td>
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
                <td>
                  <Tooltip text={new Date(pay.created_at).toLocaleString()}>
                    <span>{formatRelativeTime(pay.created_at)}</span>
                  </Tooltip>
                </td>
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
              <div className="card-actions">
                <CopyButton text={JSON.stringify(selected, null, 2)} label="Copy JSON" />
                <button className="btn-icon" onClick={() => { setSelected(null); setSelectedIndex(-1); }} title="Close (Esc)">×</button>
              </div>
            </div>
            <CodeBlock>{JSON.stringify(selected, null, 2)}</CodeBlock>
          </div>
        </div>
      )}
    </div>
  );
}

// Balances Tab
interface BalancesTabProps {
  balances: Balance[];
  onFetch?: () => void;
}

function BalancesTab({ balances, onFetch }: BalancesTabProps) {
  const [selected, setSelected] = useState<Balance | null>(null);
  const [selectedIndex, setSelectedIndex] = useState<number>(-1);

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (balances.length === 0) return;
      
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        const newIndex = Math.min(selectedIndex + 1, balances.length - 1);
        setSelectedIndex(newIndex);
        setSelected(balances[newIndex]);
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        const newIndex = Math.max(selectedIndex - 1, 0);
        setSelectedIndex(newIndex);
        setSelected(balances[newIndex]);
      } else if (e.key === 'Escape') {
        setSelected(null);
        setSelectedIndex(-1);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [balances, selectedIndex]);

  if (balances.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">[$]</div>
        <p>No balances fetched yet</p>
        <p className="text-muted">Fetch current balances from the provider</p>
        {onFetch && (
          <button className="btn-primary" onClick={onFetch} style={{ marginTop: '16px' }}>
            Fetch Balances
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="data-tab">
      <div className="data-list">
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
            {balances.map((bal, index) => (
              <tr 
                key={index}
                onClick={() => { setSelected(bal); setSelectedIndex(index); }}
                className={selectedIndex === index ? 'selected' : ''}
              >
                <td className="mono">{bal.account_reference}</td>
                <td>{bal.asset}</td>
                <td className="mono">{bal.amount}</td>
                <td>
                  <Tooltip text={new Date(bal.created_at).toLocaleString()}>
                    <span>{formatRelativeTime(bal.created_at)}</span>
                  </Tooltip>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {selected && (
        <div className="data-detail">
          <div className="card">
            <div className="card-header">
              <div className="card-title">Balance Details</div>
              <div className="card-actions">
                <CopyButton text={JSON.stringify(selected, null, 2)} label="Copy JSON" />
                <button className="btn-icon" onClick={() => { setSelected(null); setSelectedIndex(-1); }} title="Close (Esc)">×</button>
              </div>
            </div>
            <CodeBlock>{JSON.stringify(selected, null, 2)}</CodeBlock>
          </div>
        </div>
      )}
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
  genericServerStatus?: GenericServerStatus | null;
  connectors?: ConnectorSummary[];
  onSetGenericConnector?: (connectorId: string, apiKey?: string) => void;
}

function DebugTab({ logs, pluginCalls, httpRequests, httpCaptureEnabled, onClear, onToggleCapture, onSaveSnapshot, genericServerStatus, connectors, onSetGenericConnector }: DebugTabProps) {
  const [view, setView] = useState<'logs' | 'calls' | 'http' | 'generic'>('http');
  const [selectedCall, setSelectedCall] = useState<PluginCall | null>(null);
  const [selectedRequest, setSelectedRequest] = useState<HTTPRequest | null>(null);
  const [selectedLog, setSelectedLog] = useState<DebugEntry | null>(null);
  const [genericApiKey, setGenericApiKey] = useState('');
  const [selectedGenericConnector, setSelectedGenericConnector] = useState<string>('');

  // Initialize selected connector from status
  useEffect(() => {
    if (genericServerStatus?.connector_id) {
      setSelectedGenericConnector(genericServerStatus.connector_id);
    }
  }, [genericServerStatus?.connector_id]);

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
          <button className={`tab ${view === 'generic' ? 'active' : ''}`} onClick={() => setView('generic')}>
            Generic Server {genericServerStatus?.enabled && <span className="status-dot active" />}
          </button>
        </div>
        <div className="debug-actions">
          <Tooltip text={httpCaptureEnabled ? 'Stop capturing HTTP requests' : 'Start capturing HTTP requests'}>
            <button 
              className={`btn-small ${httpCaptureEnabled ? 'btn-primary' : 'btn-secondary'}`}
              onClick={onToggleCapture}
            >
              {httpCaptureEnabled ? '● Capture On' : '○ Capture Off'}
            </button>
          </Tooltip>
          <Tooltip text="Clear all debug logs, calls, and requests">
            <button className="btn-secondary btn-small" onClick={onClear}>Clear All</button>
          </Tooltip>
        </div>
      </div>

      {view === 'http' && (
        <div className="http-requests">
          <div className={`requests-list ${!selectedRequest ? 'expanded' : ''}`}>
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
                      <Tooltip text="Save this request/response as a test snapshot">
                        <button 
                          className="btn-primary btn-small" 
                          onClick={() => onSaveSnapshot(selectedRequest.id)}
                        >
                          + Save Snapshot
                        </button>
                      </Tooltip>
                    )}
                    <button className="btn-icon" onClick={() => setSelectedRequest(null)} title="Close (Esc)">×</button>
                  </div>
                </div>
                
                <div className="request-url-full">
                  <span>{selectedRequest.url}</span>
                  <CopyButton text={selectedRequest.url} label="Copy URL" />
                </div>
                <div className="request-timing">
                  <Tooltip text={new Date(selectedRequest.timestamp).toLocaleString()}>
                    <span>{formatRelativeTime(selectedRequest.timestamp)}</span>
                  </Tooltip>
                  {' · '}{(selectedRequest.duration / 1000000).toFixed(2)}ms
                </div>

                {selectedRequest.error && (
                  <>
                    <h4>Error</h4>
                    <pre className="text-danger">{selectedRequest.error}</pre>
                  </>
                )}

                <h4>Request Headers</h4>
                <CodeBlock>{JSON.stringify(selectedRequest.request_headers, null, 2)}</CodeBlock>

                {selectedRequest.request_body && (
                  <>
                    <h4>Request Body</h4>
                    <CodeBlock>{formatBody(selectedRequest.request_body)}</CodeBlock>
                  </>
                )}

                {selectedRequest.response_headers && (
                  <>
                    <h4>Response Headers</h4>
                    <CodeBlock>{JSON.stringify(selectedRequest.response_headers, null, 2)}</CodeBlock>
                  </>
                )}

                {selectedRequest.response_body && (
                  <>
                    <h4>Response Body</h4>
                    <CodeBlock>{formatBody(selectedRequest.response_body)}</CodeBlock>
                  </>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {view === 'logs' && (
        <div className="logs-panel">
          <div className="log-list">
            {logs.length === 0 ? (
              <div className="empty-state">
                <p className="text-muted">No logs yet</p>
              </div>
            ) : (
              logs.map((log) => (
                <div 
                  key={log.id} 
                  className={`log-entry log-${log.type} ${selectedLog?.id === log.id ? 'selected' : ''}`}
                  onClick={() => setSelectedLog(log)}
                >
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
          {selectedLog && (
            <div className="log-detail">
              <div className="card">
                <div className="card-header">
                  <div className="card-title">
                    <span className={`badge badge-${
                      selectedLog.type === 'error' ? 'danger' :
                      selectedLog.type === 'plugin_call' ? 'info' :
                      selectedLog.type === 'state_change' ? 'warning' : 'success'
                    }`}>
                      {selectedLog.type}
                    </span>
                    {selectedLog.operation}
                  </div>
                  <button className="btn-icon" onClick={() => setSelectedLog(null)} title="Close (Esc)">×</button>
                </div>
                
                <div className="log-detail-meta">
                  <div><strong>Time:</strong> {new Date(selectedLog.timestamp).toLocaleString()}</div>
                  {selectedLog.duration && <div><strong>Duration:</strong> {(selectedLog.duration / 1000000).toFixed(2)}ms</div>}
                </div>

                {selectedLog.message && (
                  <>
                    <h4>Message</h4>
                    <pre>{selectedLog.message}</pre>
                  </>
                )}

                {selectedLog.error && (
                  <>
                    <h4>Error</h4>
                    <pre className="text-danger">{selectedLog.error}</pre>
                  </>
                )}

                {selectedLog.data !== undefined && selectedLog.data !== null && (
                  <>
                    <h4>Data</h4>
                    <CodeBlock>{JSON.stringify(selectedLog.data, null, 2)}</CodeBlock>
                  </>
                )}
              </div>
            </div>
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
                  <div className="card-actions">
                    <CopyButton text={JSON.stringify({ input: selectedCall.input, output: selectedCall.output }, null, 2)} label="Copy" />
                    <button className="btn-icon" onClick={() => setSelectedCall(null)} title="Close (Esc)">×</button>
                  </div>
                </div>
                <h4>Input</h4>
                <CodeBlock>{JSON.stringify(selectedCall.input, null, 2)}</CodeBlock>
                {selectedCall.output !== undefined && selectedCall.output !== null && (
                  <>
                    <h4>Output</h4>
                    <CodeBlock>{JSON.stringify(selectedCall.output, null, 2)}</CodeBlock>
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

      {view === 'generic' && (
        <div className="generic-server-panel">
          <div className="card">
            <div className="card-header">
              <div className="card-title">Generic Connector Server</div>
            </div>
            <div className="card-body">
              <p className="text-muted" style={{ marginBottom: '16px' }}>
                Expose a connector via the generic connector REST API for remote integration testing.
                Staging services can install a "generic connector" pointing to this workbench endpoint.
              </p>

              <div className="form-group">
                <label>Connector</label>
                <select 
                  value={selectedGenericConnector}
                  onChange={(e) => setSelectedGenericConnector(e.target.value)}
                >
                  <option value="">-- Select a connector --</option>
                  {connectors?.filter(c => c.installed).map(c => (
                    <option key={c.id} value={c.id}>
                      {c.provider} - {c.name || c.id}
                    </option>
                  ))}
                </select>
                {connectors && !connectors.some(c => c.installed) && (
                  <p className="text-muted" style={{ marginTop: '8px', fontSize: '12px' }}>
                    No installed connectors. Install a connector first.
                  </p>
                )}
              </div>

              <div className="form-group">
                <label>API Key (optional)</label>
                <input 
                  type="text"
                  value={genericApiKey}
                  onChange={(e) => setGenericApiKey(e.target.value)}
                  placeholder="Leave empty for no authentication"
                />
                <p className="text-muted" style={{ marginTop: '4px', fontSize: '12px' }}>
                  If set, requests must include <code>Authorization: Bearer &lt;key&gt;</code> or <code>X-API-Key</code> header
                </p>
              </div>

              <div className="form-actions" style={{ marginTop: '16px' }}>
                <button 
                  className="btn-primary"
                  onClick={() => onSetGenericConnector?.(selectedGenericConnector, genericApiKey)}
                  disabled={!selectedGenericConnector}
                >
                  {genericServerStatus?.enabled ? 'Update Configuration' : 'Enable Generic Server'}
                </button>
                {genericServerStatus?.enabled && (
                  <button 
                    className="btn-secondary"
                    onClick={() => {
                      onSetGenericConnector?.('', '');
                      setSelectedGenericConnector('');
                      setGenericApiKey('');
                    }}
                  >
                    Disable
                  </button>
                )}
              </div>

              {genericServerStatus?.enabled && (
                <div className="generic-status" style={{ marginTop: '24px', padding: '16px', background: 'var(--panel-bg)', borderRadius: '8px' }}>
                  <h4 style={{ marginBottom: '12px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <span className="status-dot active" style={{ width: '8px', height: '8px', borderRadius: '50%', background: 'var(--success)' }} />
                    Server Active
                  </h4>
                  <div className="status-grid" style={{ display: 'grid', gap: '8px' }}>
                    <div className="status-row" style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <span className="text-muted">Endpoint:</span>
                      <span style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <code>{genericServerStatus.endpoint}</code>
                        <CopyButton text={genericServerStatus.endpoint} label="Copy" />
                      </span>
                    </div>
                    <div className="status-row" style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <span className="text-muted">Provider:</span>
                      <span className="provider-badge">{genericServerStatus.connector_provider}</span>
                    </div>
                    <div className="status-row" style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <span className="text-muted">Authentication:</span>
                      <span>{genericServerStatus.has_api_key ? 'API Key Required' : 'None'}</span>
                    </div>
                  </div>

                  <div style={{ marginTop: '16px' }}>
                    <h5 style={{ marginBottom: '8px' }}>Available Endpoints:</h5>
                    <ul style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
                      <li><code>GET {genericServerStatus.endpoint}/accounts</code> - List accounts</li>
                      <li><code>GET {genericServerStatus.endpoint}/accounts/:id/balances</code> - Get account balances</li>
                      <li><code>GET {genericServerStatus.endpoint}/beneficiaries</code> - List beneficiaries</li>
                      <li><code>GET {genericServerStatus.endpoint}/transactions</code> - List transactions</li>
                    </ul>
                  </div>

                  <div style={{ marginTop: '16px' }}>
                    <h5 style={{ marginBottom: '8px' }}>Generic Connector Config:</h5>
                    <p className="text-muted" style={{ fontSize: '12px', marginBottom: '8px' }}>
                      Use these settings when installing a generic connector on staging:
                    </p>
                    <CodeBlock language="json">{JSON.stringify({
                      endpoint: genericServerStatus.endpoint,
                      apiKey: genericServerStatus.has_api_key ? '<your-api-key>' : '',
                      pollingPeriod: '30s'
                    }, null, 2)}</CodeBlock>
                  </div>
                </div>
              )}
            </div>
          </div>
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
  connectorId: string;
  captureId: string;
  onClose: () => void;
  onSaved: () => void;
}

function SaveSnapshotModal({ connectorId, captureId, onClose, onSaved }: SaveSnapshotModalProps) {
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
      await connectorApi(connectorId).createSnapshotFromCapture(captureId, {
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
  const { connectorId } = useConnector();
  const [info, setInfo] = useState<ConnectorInfo | null>(null);
  const [files, setFiles] = useState<FileNode[]>([]);
  const [selectedFile, setSelectedFile] = useState<SourceFile | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [view, setView] = useState<'overview' | 'files' | 'search'>('overview');
  const [loading, setLoading] = useState(false);
  const [expandedDirs, setExpandedDirs] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (connectorId) {
      loadInfo();
    }
  }, [connectorId]);

  const loadInfo = async () => {
    if (!connectorId) return;
    try {
      const api = connectorApi(connectorId);
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
    if (!connectorId) return;
    setLoading(true);
    try {
      const file = await connectorApi(connectorId).getFile(path);
      setSelectedFile(file);
    } catch (e) {
      console.error('Failed to load file:', e);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = async () => {
    if (!searchQuery.trim() || !connectorId) return;
    setLoading(true);
    try {
      const res = await connectorApi(connectorId).searchCode(searchQuery);
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
  const { connectorId } = useConnector();
  const [taskData, setTaskData] = useState<TaskTreeSummary | null>(null);
  const [executions, setExecutions] = useState<TaskExecution[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedTask, setSelectedTask] = useState<TaskNodeSummary | null>(null);
  const [selectedExecution, setSelectedExecution] = useState<TaskExecution | null>(null);

  const refresh = useCallback(async () => {
    if (!connectorId) return;
    try {
      const api = connectorApi(connectorId);
      const [tasks, execs] = await Promise.all([
        api.getTasks(),
        api.getTaskExecutions(30),
      ]);
      setTaskData(tasks);
      setExecutions(execs.executions || []);
    } catch (e) {
      console.error('Failed to load tasks:', e);
    }
  }, [connectorId]);

  useEffect(() => {
    refresh();
    const interval = setInterval(refresh, 1000);
    return () => clearInterval(interval);
  }, [refresh]);

  const handleStep = async () => {
    if (!connectorId) return;
    setLoading(true);
    try {
      await connectorApi(connectorId).stepTask();
    } finally {
      setLoading(false);
    }
  };

  const toggleStepMode = async () => {
    if (!taskData || !connectorId) return;
    setLoading(true);
    try {
      await connectorApi(connectorId).setStepMode(!taskData.step_mode);
      refresh();
    } finally {
      setLoading(false);
    }
  };

  const handleReset = async () => {
    if (!connectorId) return;
    setLoading(true);
    try {
      await connectorApi(connectorId).resetTasks();
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

    const isSelected = selectedTask?.id === node.id;

    return (
      <div key={node.id} className="task-node" style={{ marginLeft: depth * 20 }}>
        <div 
          className={`task-node-header task-${node.status} ${isSelected ? 'selected' : ''}`}
          onClick={() => { setSelectedTask(node); setSelectedExecution(null); }}
        >
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
          <Tooltip text="Pause execution between each task step">
            <button 
              className={`btn-small ${taskData?.step_mode ? 'btn-primary' : 'btn-secondary'}`}
              onClick={toggleStepMode}
              disabled={loading}
            >
              {taskData?.step_mode ? '|| Step Mode On' : '> Step Mode Off'}
            </button>
          </Tooltip>
          {taskData?.step_mode && (
            <Tooltip text="Execute the next task step">
              <button 
                className="btn-primary btn-small"
                onClick={handleStep}
                disabled={loading || !taskData?.is_running}
              >
                {'>>'} Next Step
              </button>
            </Tooltip>
          )}
          <Tooltip text="Reset all tasks and execution history">
            <button 
              className="btn-secondary btn-small"
              onClick={handleReset}
              disabled={loading}
            >
              Reset
            </button>
          </Tooltip>
        </div>
        {taskData && (
          <div className="tasks-stats">
            {taskData.is_running && (
              <ProgressBar 
                current={taskData.stats.success_count} 
                total={taskData.stats.total_executions || taskData.stats.success_count + 1} 
                label="Progress"
              />
            )}
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
                <div 
                  key={exec.id} 
                  className={`execution-item exec-${exec.status} ${selectedExecution?.id === exec.id ? 'selected' : ''}`}
                  onClick={() => { setSelectedExecution(exec); setSelectedTask(null); }}
                >
                  <div className="execution-header">
                    <span className={`execution-status status-${exec.status}`}>
                      {exec.status === 'completed' ? '✓' : exec.status === 'failed' ? '✗' : '○'}
                    </span>
                    <span className="execution-page">Page {exec.page_number}</span>
                    <span className="execution-items">{exec.items_count} items</span>
                    <span className="execution-duration">{(exec.duration / 1000000).toFixed(0)}ms</span>
                    {exec.has_more && <span className="text-muted">has_more: true</span>}
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

        {/* Detail Panel */}
        {(selectedTask || selectedExecution) && (
          <div className="tasks-detail-panel">
            <div className="card">
              <div className="card-header">
                <div className="card-title">
                  {selectedTask ? 'Task Details' : 'Execution Details'}
                </div>
                <div className="card-actions">
                  <CopyButton 
                    text={JSON.stringify(selectedTask || selectedExecution, null, 2)} 
                    label="Copy JSON" 
                  />
                  <button 
                    className="btn-icon" 
                    onClick={() => { setSelectedTask(null); setSelectedExecution(null); }} 
                    title="Close (Esc)"
                  >×</button>
                </div>
              </div>
              {selectedTask && (
                <div className="detail-content">
                  <div className="detail-row">
                    <span className="detail-label">ID</span>
                    <span className="detail-value mono">{selectedTask.id}</span>
                  </div>
                  <div className="detail-row">
                    <span className="detail-label">Type</span>
                    <span className="detail-value">{formatTaskType(selectedTask.type)}</span>
                  </div>
                  {selectedTask.name && (
                    <div className="detail-row">
                      <span className="detail-label">Name</span>
                      <span className="detail-value">{selectedTask.name}</span>
                    </div>
                  )}
                  <div className="detail-row">
                    <span className="detail-label">Status</span>
                    <span className={`detail-value status-${selectedTask.status}`}>{selectedTask.status}</span>
                  </div>
                  {selectedTask.duration && (
                    <div className="detail-row">
                      <span className="detail-label">Duration</span>
                      <span className="detail-value">{selectedTask.duration}</span>
                    </div>
                  )}
                  <div className="detail-row">
                    <span className="detail-label">Items</span>
                    <span className="detail-value">{selectedTask.items_count}</span>
                  </div>
                  {selectedTask.child_count > 0 && (
                    <div className="detail-row">
                      <span className="detail-label">Child Tasks</span>
                      <span className="detail-value">{selectedTask.child_count}</span>
                    </div>
                  )}
                  {selectedTask.last_exec_time && (
                    <div className="detail-row">
                      <span className="detail-label">Last Execution</span>
                      <span className="detail-value">{selectedTask.last_exec_time}</span>
                    </div>
                  )}
                  {selectedTask.error && (
                    <div className="detail-row detail-error">
                      <span className="detail-label">Error</span>
                      <span className="detail-value">{selectedTask.error}</span>
                    </div>
                  )}
                  {selectedTask.from_payload !== undefined && selectedTask.from_payload !== null && (
                    <div className="detail-payload">
                      <div className="detail-payload-header">Trigger Payload</div>
                      <CodeBlock>{JSON.stringify(selectedTask.from_payload, null, 2)}</CodeBlock>
                    </div>
                  )}
                </div>
              )}
              {selectedExecution && (
                <div className="detail-content">
                  <div className="detail-row">
                    <span className="detail-label">ID</span>
                    <span className="detail-value mono">{selectedExecution.id}</span>
                  </div>
                  <div className="detail-row">
                    <span className="detail-label">Status</span>
                    <span className={`detail-value status-${selectedExecution.status}`}>{selectedExecution.status}</span>
                  </div>
                  <div className="detail-row">
                    <span className="detail-label">Page</span>
                    <span className="detail-value">{selectedExecution.page_number}</span>
                  </div>
                  <div className="detail-row">
                    <span className="detail-label">Items</span>
                    <span className="detail-value">{selectedExecution.items_count}</span>
                  </div>
                  <div className="detail-row">
                    <span className="detail-label">Has More</span>
                    <span className="detail-value">{selectedExecution.has_more ? 'Yes' : 'No'}</span>
                  </div>
                  <div className="detail-row">
                    <span className="detail-label">Duration</span>
                    <span className="detail-value">{(selectedExecution.duration / 1000000).toFixed(2)}ms</span>
                  </div>
                  <div className="detail-row">
                    <span className="detail-label">Started</span>
                    <span className="detail-value">{new Date(selectedExecution.started_at).toLocaleString()}</span>
                  </div>
                  {selectedExecution.completed_at && (
                    <div className="detail-row">
                      <span className="detail-label">Completed</span>
                      <span className="detail-value">{new Date(selectedExecution.completed_at).toLocaleString()}</span>
                    </div>
                  )}
                  {selectedExecution.error && (
                    <div className="detail-row detail-error">
                      <span className="detail-label">Error</span>
                      <span className="detail-value">{selectedExecution.error}</span>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function formatPluginType(pluginType: number | string): string {
  const types: Record<number, string> = {
    0: 'PSP',
    1: 'Open Banking',
    2: 'Both',
  };
  if (typeof pluginType === 'number') {
    return types[pluginType] || `Type ${pluginType}`;
  }
  return String(pluginType);
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
  const { connectorId } = useConnector();
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [stats, setStats] = useState<SnapshotStats | null>(null);
  const [selectedSnapshot, setSelectedSnapshot] = useState<Snapshot | null>(null);
  const [showTestPreview, setShowTestPreview] = useState(false);
  const [testPreview, setTestPreview] = useState<GenerateResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [confirmAction, setConfirmAction] = useState<{ title: string; message: string; onConfirm: () => void } | null>(null);

  const refresh = useCallback(async () => {
    if (!connectorId) return;
    try {
      const api = connectorApi(connectorId);
      const [snapshotData, statsData] = await Promise.all([
        api.listSnapshots(),
        api.getSnapshotStats(),
      ]);
      setSnapshots(snapshotData.snapshots || []);
      setStats(statsData);
    } catch (e) {
      console.error('Failed to load snapshots:', e);
    }
  }, [connectorId]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const handleDelete = (id: string) => {
    if (!connectorId) return;
    const snapshot = snapshots.find(s => s.id === id);
    setConfirmAction({
      title: 'Delete Snapshot',
      message: `Delete snapshot "${snapshot?.name || id}"? This cannot be undone.`,
      onConfirm: async () => {
        setConfirmAction(null);
        setLoading(true);
        try {
          await connectorApi(connectorId).deleteSnapshot(id);
          refresh();
          if (selectedSnapshot?.id === id) {
            setSelectedSnapshot(null);
          }
        } finally {
          setLoading(false);
        }
      }
    });
  };

  const handlePreviewTests = async () => {
    if (!connectorId) return;
    setLoading(true);
    try {
      const preview = await connectorApi(connectorId).previewTests();
      setTestPreview(preview);
      setShowTestPreview(true);
    } catch (e: unknown) {
      alert('Failed to generate preview: ' + (e instanceof Error ? e.message : String(e)));
    } finally {
      setLoading(false);
    }
  };

  const handleGenerateTests = async () => {
    if (!connectorId) return;
    const outputDir = prompt('Output directory (leave empty to preview only):');
    setLoading(true);
    try {
      const result = await connectorApi(connectorId).generateTests(outputDir || undefined);
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

  const handleClearAll = () => {
    if (!connectorId) return;
    setConfirmAction({
      title: 'Clear All Snapshots',
      message: `Delete all ${snapshots.length} snapshots? This cannot be undone.`,
      onConfirm: async () => {
        setConfirmAction(null);
        setLoading(true);
        try {
          await connectorApi(connectorId).clearSnapshots();
          refresh();
          setSelectedSnapshot(null);
        } finally {
          setLoading(false);
        }
      }
    });
  };

  return (
    <div className="snapshots-tab">
      <div className="snapshots-header">
        <div className="snapshots-controls">
          <Tooltip text="Preview generated test code without saving">
            <button 
              className="btn-secondary btn-small" 
              onClick={handlePreviewTests}
              disabled={loading || snapshots.length === 0}
            >
              Preview Tests
            </button>
          </Tooltip>
          <Tooltip text="Generate and save test files to disk">
            <button 
              className="btn-primary btn-small" 
              onClick={handleGenerateTests}
              disabled={loading || snapshots.length === 0}
            >
              Generate Tests
            </button>
          </Tooltip>
          <Tooltip text="Delete all saved snapshots">
            <button 
              className="btn-danger btn-small" 
              onClick={handleClearAll}
              disabled={loading || snapshots.length === 0}
            >
              Clear All
            </button>
          </Tooltip>
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
                <p className="text-muted">Go to Debug → HTTP Traffic and click "Save Snapshot" on a request</p>
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
                      ×
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
                  <CodeBlock>{formatBody(selectedSnapshot.request.body)}</CodeBlock>
                </>
              )}

              <h4>Response ({selectedSnapshot.response.status_code})</h4>
              {selectedSnapshot.response.body && (
                <CodeBlock>{formatBody(selectedSnapshot.response.body)}</CodeBlock>
              )}
            </div>
          </div>
        )}
      </div>

      {confirmAction && (
        <ConfirmModal
          title={confirmAction.title}
          message={confirmAction.message}
          variant="danger"
          confirmText="Delete"
          onConfirm={confirmAction.onConfirm}
          onCancel={() => setConfirmAction(null)}
        />
      )}
    </div>
  );
}

// Analysis Tab - Schema Inference & Baseline Comparison
function AnalysisTab() {
  const { connectorId } = useConnector();
  const [view, setView] = useState<'schemas' | 'baselines'>('schemas');
  const [schemas, setSchemas] = useState<InferredSchema[]>([]);
  const [schemaBaselines, setSchemaBaselines] = useState<InferredSchema[]>([]);
  const [selectedSchema, setSelectedSchema] = useState<InferredSchema | null>(null);
  const [schemaDiff, setSchemaDiff] = useState<SchemaDiff | null>(null);
  
  const [baselines, setBaselines] = useState<DataBaseline[]>([]);
  const [selectedBaseline, setSelectedBaseline] = useState<DataBaseline | null>(null);
  const [baselineDiff, setBaselineDiff] = useState<BaselineDiff | null>(null);
  
  const [loading, setLoading] = useState(false);
  const [confirmAction, setConfirmAction] = useState<{ title: string; message: string; onConfirm: () => void } | null>(null);

  const refreshSchemas = useCallback(async () => {
    if (!connectorId) return;
    try {
      const api = connectorApi(connectorId);
      const [schemaData, baselineData] = await Promise.all([
        api.listSchemas(),
        api.listSchemaBaselines(),
      ]);
      setSchemas(schemaData.schemas || []);
      setSchemaBaselines(baselineData.baselines || []);
    } catch (e) {
      console.error('Failed to load schemas:', e);
    }
  }, [connectorId]);

  const refreshBaselines = useCallback(async () => {
    if (!connectorId) return;
    try {
      const data = await connectorApi(connectorId).listBaselines();
      setBaselines(data.baselines || []);
    } catch (e) {
      console.error('Failed to load baselines:', e);
    }
  }, [connectorId]);

  useEffect(() => {
    refreshSchemas();
    refreshBaselines();
  }, [refreshSchemas, refreshBaselines]);

  const handleSaveSchemaBaselines = async () => {
    if (!connectorId) return;
    setLoading(true);
    try {
      await connectorApi(connectorId).saveAllSchemaBaselines();
      await refreshSchemas();
      alert('Schema baselines saved!');
    } finally {
      setLoading(false);
    }
  };

  const handleCompareSchema = async (operation: string) => {
    if (!connectorId) return;
    setLoading(true);
    try {
      const diff = await connectorApi(connectorId).compareSchema(operation);
      setSchemaDiff(diff);
    } catch (e: unknown) {
      alert('Error: ' + (e instanceof Error ? e.message : String(e)));
    } finally {
      setLoading(false);
    }
  };

  const handleSaveBaseline = async () => {
    if (!connectorId) return;
    const name = prompt('Baseline name (optional):');
    setLoading(true);
    try {
      await connectorApi(connectorId).saveBaseline(name || undefined);
      await refreshBaselines();
    } finally {
      setLoading(false);
    }
  };

  const handleCompareBaseline = async (id: string) => {
    if (!connectorId) return;
    setLoading(true);
    try {
      const diff = await connectorApi(connectorId).compareBaseline(id);
      setBaselineDiff(diff);
    } catch (e: unknown) {
      alert('Error: ' + (e instanceof Error ? e.message : String(e)));
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteBaseline = (id: string) => {
    if (!connectorId) return;
    const baseline = baselines.find(b => b.id === id);
    setConfirmAction({
      title: 'Delete Baseline',
      message: `Delete baseline "${baseline?.name || id}"? This cannot be undone.`,
      onConfirm: async () => {
        setConfirmAction(null);
        setLoading(true);
        try {
          await connectorApi(connectorId).deleteBaseline(id);
          await refreshBaselines();
          if (selectedBaseline?.id === id) {
            setSelectedBaseline(null);
            setBaselineDiff(null);
          }
        } finally {
          setLoading(false);
        }
      }
    });
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
            <Tooltip text="Save current schemas as reference baselines for future comparison">
              <button 
                className="btn-primary btn-small" 
                onClick={handleSaveSchemaBaselines}
                disabled={loading || schemas.length === 0}
              >
                + Save All as Baselines
              </button>
            </Tooltip>
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
            <Tooltip text="Save current data state as a baseline for future comparison">
              <button 
                className="btn-primary btn-small" 
                onClick={handleSaveBaseline}
                disabled={loading}
              >
                + Save Current as Baseline
              </button>
            </Tooltip>
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
                          title="Delete"
                        >
                          ×
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

      {confirmAction && (
        <ConfirmModal
          title={confirmAction.title}
          message={confirmAction.message}
          variant="danger"
          confirmText="Delete"
          onConfirm={confirmAction.onConfirm}
          onCancel={() => setConfirmAction(null)}
        />
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
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [filter, setFilter] = useState('');
  const [hideNull, setHideNull] = useState(false);

  const stateEntries = Object.entries(states);
  const filteredEntries = stateEntries.filter(([key, value]) => {
    const matchesFilter = key.toLowerCase().includes(filter.toLowerCase());
    const matchesNull = hideNull ? value !== null : true;
    return matchesFilter && matchesNull;
  });

  const selectedValue = selectedKey ? states[selectedKey] : null;
  const nullCount = stateEntries.filter(([, v]) => v === null).length;

  return (
    <div className="state-tab">
      <div className="tabs">
        <button className={`tab ${view === 'states' ? 'active' : ''}`} onClick={() => setView('states')}>
          Fetch States ({stateEntries.length})
        </button>
        <button className={`tab ${view === 'tree' ? 'active' : ''}`} onClick={() => setView('tree')}>
          Tasks Tree
        </button>
      </div>

      {view === 'states' && (
        <div className="state-content">
          <div className="state-list-panel">
            <div className="state-list-header">
              <input
                type="text"
                placeholder="Filter states..."
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                className="state-filter"
              />
              {nullCount > 0 && (
                <label className="state-hide-null">
                  <input
                    type="checkbox"
                    checked={hideNull}
                    onChange={(e) => setHideNull(e.target.checked)}
                  />
                  Hide null ({nullCount})
                </label>
              )}
            </div>
            <div className="state-list">
              {filteredEntries.length === 0 ? (
                <div className="empty-state">
                  <p className="text-muted">
                    {stateEntries.length === 0 ? 'No state saved yet' : 'No matching states'}
                  </p>
                </div>
              ) : (
                filteredEntries.map(([key, value]) => (
                  <div
                    key={key}
                    className={`state-item ${selectedKey === key ? 'selected' : ''} ${value === null ? 'is-null' : ''}`}
                    onClick={() => setSelectedKey(key)}
                  >
                    <span className="state-key">{key}</span>
                    {value === null && <span className="state-null-badge">null</span>}
                  </div>
                ))
              )}
            </div>
          </div>
          <div className="state-detail-panel">
            {selectedKey ? (
              <div className="card">
                <div className="card-header">
                  <div className="card-title mono">{selectedKey}</div>
                  <CopyButton text={JSON.stringify(selectedValue, null, 2)} label="Copy" />
                </div>
                <CodeBlock>{JSON.stringify(selectedValue, null, 2)}</CodeBlock>
              </div>
            ) : (
              <div className="empty-state">
                <p className="text-muted">Select a state to view details</p>
              </div>
            )}
          </div>
        </div>
      )}

      {view === 'tree' && (
        <div className="tasks-tree">
          {tasksTree !== null && tasksTree !== undefined ? (
            <div className="card">
              <div className="card-header">
                <div className="card-title">Tasks Tree</div>
                <CopyButton text={JSON.stringify(tasksTree, null, 2)} label="Copy" />
              </div>
              <CodeBlock>{JSON.stringify(tasksTree, null, 2)}</CodeBlock>
            </div>
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
