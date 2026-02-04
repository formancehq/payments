package workbench

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/formancehq/payments/internal/models"
)

// MemoryStorage provides an in-memory implementation of storage for the workbench.
// It stores only the data needed for connector development and testing.
type MemoryStorage struct {
	mu sync.RWMutex

	// Core data
	accounts  map[string]models.PSPAccount
	payments  map[string]models.PSPPayment
	balances  map[string]models.PSPBalance
	others    map[string][]models.PSPOther
	
	// External accounts (beneficiaries, etc.)
	externalAccounts map[string]models.PSPAccount

	// State management
	states map[string]json.RawMessage

	// Workflow tree from install
	tasksTree *models.ConnectorTasksTree

	// Webhook configs
	webhookConfigs []models.PSPWebhookConfig

	// Statistics
	stats StorageStats
}

// StorageStats tracks storage statistics.
type StorageStats struct {
	AccountsCount         int
	PaymentsCount         int
	BalancesCount         int
	ExternalAccountsCount int
	LastUpdated           time.Time
}

// NewMemoryStorage creates a new in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		accounts:         make(map[string]models.PSPAccount),
		payments:         make(map[string]models.PSPPayment),
		balances:         make(map[string]models.PSPBalance),
		others:           make(map[string][]models.PSPOther),
		externalAccounts: make(map[string]models.PSPAccount),
		states:           make(map[string]json.RawMessage),
	}
}

// Clear clears all stored data.
func (s *MemoryStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.accounts = make(map[string]models.PSPAccount)
	s.payments = make(map[string]models.PSPPayment)
	s.balances = make(map[string]models.PSPBalance)
	s.others = make(map[string][]models.PSPOther)
	s.externalAccounts = make(map[string]models.PSPAccount)
	s.states = make(map[string]json.RawMessage)
	s.tasksTree = nil
	s.webhookConfigs = nil
	s.stats = StorageStats{}
}

// === Accounts ===

// StoreAccounts stores accounts.
func (s *MemoryStorage) StoreAccounts(accounts []models.PSPAccount) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, acc := range accounts {
		s.accounts[acc.Reference] = acc
	}
	s.stats.AccountsCount = len(s.accounts)
	s.stats.LastUpdated = time.Now()
}

// GetAccounts returns all accounts.
func (s *MemoryStorage) GetAccounts() []models.PSPAccount {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accounts := make([]models.PSPAccount, 0, len(s.accounts))
	for _, acc := range s.accounts {
		accounts = append(accounts, acc)
	}
	return accounts
}

// GetAccount returns a specific account by reference.
func (s *MemoryStorage) GetAccount(reference string) (models.PSPAccount, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	acc, ok := s.accounts[reference]
	return acc, ok
}

// === External Accounts ===

// StoreExternalAccounts stores external accounts.
func (s *MemoryStorage) StoreExternalAccounts(accounts []models.PSPAccount) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, acc := range accounts {
		s.externalAccounts[acc.Reference] = acc
	}
	s.stats.ExternalAccountsCount = len(s.externalAccounts)
	s.stats.LastUpdated = time.Now()
}

// GetExternalAccounts returns all external accounts.
func (s *MemoryStorage) GetExternalAccounts() []models.PSPAccount {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accounts := make([]models.PSPAccount, 0, len(s.externalAccounts))
	for _, acc := range s.externalAccounts {
		accounts = append(accounts, acc)
	}
	return accounts
}

// === Payments ===

// StorePayments stores payments.
func (s *MemoryStorage) StorePayments(payments []models.PSPPayment) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range payments {
		s.payments[p.Reference] = p
	}
	s.stats.PaymentsCount = len(s.payments)
	s.stats.LastUpdated = time.Now()
}

// GetPayments returns all payments.
func (s *MemoryStorage) GetPayments() []models.PSPPayment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	payments := make([]models.PSPPayment, 0, len(s.payments))
	for _, p := range s.payments {
		payments = append(payments, p)
	}
	return payments
}

// GetPayment returns a specific payment by reference.
func (s *MemoryStorage) GetPayment(reference string) (models.PSPPayment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.payments[reference]
	return p, ok
}

// === Balances ===

// StoreBalances stores balances.
func (s *MemoryStorage) StoreBalances(balances []models.PSPBalance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, b := range balances {
		key := b.AccountReference + "/" + b.Asset
		s.balances[key] = b
	}
	s.stats.BalancesCount = len(s.balances)
	s.stats.LastUpdated = time.Now()
}

// GetBalances returns all balances.
func (s *MemoryStorage) GetBalances() []models.PSPBalance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	balances := make([]models.PSPBalance, 0, len(s.balances))
	for _, b := range s.balances {
		balances = append(balances, b)
	}
	return balances
}

// GetBalancesForAccount returns balances for a specific account.
func (s *MemoryStorage) GetBalancesForAccount(accountRef string) []models.PSPBalance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var balances []models.PSPBalance
	for _, b := range s.balances {
		if b.AccountReference == accountRef {
			balances = append(balances, b)
		}
	}
	return balances
}

// === Others ===

// StoreOthers stores other data by name.
func (s *MemoryStorage) StoreOthers(name string, others []models.PSPOther) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.others[name] = append(s.others[name], others...)
	s.stats.LastUpdated = time.Now()
}

// GetOthers returns other data by name.
func (s *MemoryStorage) GetOthers(name string) []models.PSPOther {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.others[name]
}

// GetAllOthers returns all other data.
func (s *MemoryStorage) GetAllOthers() map[string][]models.PSPOther {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string][]models.PSPOther)
	for k, v := range s.others {
		result[k] = v
	}
	return result
}

// === State Management ===

// SaveState saves state for a given key.
func (s *MemoryStorage) SaveState(key string, state json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[key] = state
}

// GetState returns state for a given key.
func (s *MemoryStorage) GetState(key string) json.RawMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.states[key]
}

// GetAllStates returns all states.
func (s *MemoryStorage) GetAllStates() map[string]json.RawMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]json.RawMessage)
	for k, v := range s.states {
		result[k] = v
	}
	return result
}

// ClearState clears state for a given key.
func (s *MemoryStorage) ClearState(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, key)
}

// ClearAllStates clears all states.
func (s *MemoryStorage) ClearAllStates() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states = make(map[string]json.RawMessage)
}

// === Tasks Tree ===

// SetTasksTree sets the connector tasks tree.
func (s *MemoryStorage) SetTasksTree(tree *models.ConnectorTasksTree) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasksTree = tree
}

// GetTasksTree returns the connector tasks tree.
func (s *MemoryStorage) GetTasksTree() *models.ConnectorTasksTree {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.tasksTree
}

// === Webhook Configs ===

// SetWebhookConfigs sets webhook configs.
func (s *MemoryStorage) SetWebhookConfigs(configs []models.PSPWebhookConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.webhookConfigs = configs
}

// GetWebhookConfigs returns webhook configs.
func (s *MemoryStorage) GetWebhookConfigs() []models.PSPWebhookConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.webhookConfigs
}

// === Statistics ===

// GetStats returns storage statistics.
func (s *MemoryStorage) GetStats() StorageStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stats
}

// === Export/Import ===

// StorageSnapshot represents a snapshot of all storage data.
type StorageSnapshot struct {
	Accounts         map[string]models.PSPAccount   `json:"accounts"`
	Payments         map[string]models.PSPPayment   `json:"payments"`
	Balances         map[string]models.PSPBalance   `json:"balances"`
	ExternalAccounts map[string]models.PSPAccount   `json:"external_accounts"`
	Others           map[string][]models.PSPOther   `json:"others"`
	States           map[string]json.RawMessage     `json:"states"`
	TasksTree        *models.ConnectorTasksTree     `json:"tasks_tree,omitempty"`
	WebhookConfigs   []models.PSPWebhookConfig      `json:"webhook_configs,omitempty"`
	ExportedAt       time.Time                      `json:"exported_at"`
}

// Export exports all storage data as a snapshot.
func (s *MemoryStorage) Export() StorageSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return StorageSnapshot{
		Accounts:         s.accounts,
		Payments:         s.payments,
		Balances:         s.balances,
		ExternalAccounts: s.externalAccounts,
		Others:           s.others,
		States:           s.states,
		TasksTree:        s.tasksTree,
		WebhookConfigs:   s.webhookConfigs,
		ExportedAt:       time.Now(),
	}
}

// Import imports storage data from a snapshot.
func (s *MemoryStorage) Import(snapshot StorageSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if snapshot.Accounts != nil {
		s.accounts = snapshot.Accounts
	}
	if snapshot.Payments != nil {
		s.payments = snapshot.Payments
	}
	if snapshot.Balances != nil {
		s.balances = snapshot.Balances
	}
	if snapshot.ExternalAccounts != nil {
		s.externalAccounts = snapshot.ExternalAccounts
	}
	if snapshot.Others != nil {
		s.others = snapshot.Others
	}
	if snapshot.States != nil {
		s.states = snapshot.States
	}
	s.tasksTree = snapshot.TasksTree
	s.webhookConfigs = snapshot.WebhookConfigs

	s.stats.AccountsCount = len(s.accounts)
	s.stats.PaymentsCount = len(s.payments)
	s.stats.BalancesCount = len(s.balances)
	s.stats.ExternalAccountsCount = len(s.externalAccounts)
	s.stats.LastUpdated = time.Now()
}
