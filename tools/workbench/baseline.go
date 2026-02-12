package workbench

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/formancehq/payments/internal/models"
)

// Baseline represents a saved snapshot of connector output for comparison.
type Baseline struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Provider  string    `json:"provider"`

	// Captured data
	Accounts         []models.PSPAccount  `json:"accounts"`
	Payments         []models.PSPPayment  `json:"payments"`
	Balances         []models.PSPBalance  `json:"balances"`
	ExternalAccounts []models.PSPAccount  `json:"external_accounts"`

	// Metadata
	AccountCount         int `json:"account_count"`
	PaymentCount         int `json:"payment_count"`
	BalanceCount         int `json:"balance_count"`
	ExternalAccountCount int `json:"external_account_count"`
}

// BaselineDiff represents differences between baseline and current data.
type BaselineDiff struct {
	Timestamp   time.Time `json:"timestamp"`
	BaselineID  string    `json:"baseline_id"`
	HasChanges  bool      `json:"has_changes"`

	Accounts    DataDiff `json:"accounts"`
	Payments    DataDiff `json:"payments"`
	Balances    DataDiff `json:"balances"`
	ExternalAccounts DataDiff `json:"external_accounts"`

	Summary string `json:"summary"`
}

// DataDiff represents differences in a specific data type.
type DataDiff struct {
	BaselineCount int           `json:"baseline_count"`
	CurrentCount  int           `json:"current_count"`
	Added         []string      `json:"added,omitempty"`   // References of added items
	Removed       []string      `json:"removed,omitempty"` // References of removed items
	Modified      []ItemDiff    `json:"modified,omitempty"`
}

// ItemDiff represents differences in a single item.
type ItemDiff struct {
	Reference string   `json:"reference"`
	Changes   []Change `json:"changes"`
}

// Change represents a single field change.
type Change struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

// BaselineManager manages baselines and comparisons.
type BaselineManager struct {
	mu sync.RWMutex

	// Saved baselines
	baselines map[string]*Baseline

	// Reference to storage for current data
	storage *MemoryStorage

	provider string
}

// NewBaselineManager creates a new baseline manager.
func NewBaselineManager(provider string, storage *MemoryStorage) *BaselineManager {
	return &BaselineManager{
		baselines: make(map[string]*Baseline),
		storage:   storage,
		provider:  provider,
	}
}

// SaveBaseline saves current data as a baseline.
func (m *BaselineManager) SaveBaseline(name string) (*Baseline, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	accounts := m.storage.GetAccounts()
	payments := m.storage.GetPayments()
	balances := m.storage.GetBalances()
	externalAccounts := m.storage.GetExternalAccounts()

	baseline := &Baseline{
		ID:                   fmt.Sprintf("baseline-%d", time.Now().UnixNano()),
		Name:                 name,
		CreatedAt:            time.Now(),
		Provider:             m.provider,
		Accounts:             accounts,
		Payments:             payments,
		Balances:             balances,
		ExternalAccounts:     externalAccounts,
		AccountCount:         len(accounts),
		PaymentCount:         len(payments),
		BalanceCount:         len(balances),
		ExternalAccountCount: len(externalAccounts),
	}

	m.baselines[baseline.ID] = baseline
	return baseline, nil
}

// GetBaseline returns a baseline by ID.
func (m *BaselineManager) GetBaseline(id string) *Baseline {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.baselines[id]
}

// ListBaselines returns all baselines.
func (m *BaselineManager) ListBaselines() []*Baseline {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Baseline
	for _, b := range m.baselines {
		result = append(result, b)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

// DeleteBaseline deletes a baseline.
func (m *BaselineManager) DeleteBaseline(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.baselines[id]; !exists {
		return fmt.Errorf("baseline not found: %s", id)
	}
	delete(m.baselines, id)
	return nil
}

// CompareWithCurrent compares a baseline with current data.
func (m *BaselineManager) CompareWithCurrent(baselineID string) (*BaselineDiff, error) {
	m.mu.RLock()
	baseline, exists := m.baselines[baselineID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("baseline not found: %s", baselineID)
	}

	currentAccounts := m.storage.GetAccounts()
	currentPayments := m.storage.GetPayments()
	currentBalances := m.storage.GetBalances()
	currentExtAccounts := m.storage.GetExternalAccounts()

	diff := &BaselineDiff{
		Timestamp:  time.Now(),
		BaselineID: baselineID,
		Accounts:   m.compareAccounts(baseline.Accounts, currentAccounts),
		Payments:   m.comparePayments(baseline.Payments, currentPayments),
		Balances:   m.compareBalances(baseline.Balances, currentBalances),
		ExternalAccounts: m.compareAccounts(baseline.ExternalAccounts, currentExtAccounts),
	}

	diff.HasChanges = diff.Accounts.hasChanges() || diff.Payments.hasChanges() ||
		diff.Balances.hasChanges() || diff.ExternalAccounts.hasChanges()

	diff.Summary = m.generateSummary(diff)
	return diff, nil
}

func (m *BaselineManager) compareAccounts(baseline, current []models.PSPAccount) DataDiff {
	diff := DataDiff{
		BaselineCount: len(baseline),
		CurrentCount:  len(current),
	}

	baselineMap := make(map[string]models.PSPAccount)
	for _, acc := range baseline {
		baselineMap[acc.Reference] = acc
	}

	currentMap := make(map[string]models.PSPAccount)
	for _, acc := range current {
		currentMap[acc.Reference] = acc
	}

	// Find added and modified
	for ref, curr := range currentMap {
		if base, exists := baselineMap[ref]; exists {
			// Check for modifications
			changes := m.compareAccount(base, curr)
			if len(changes) > 0 {
				diff.Modified = append(diff.Modified, ItemDiff{
					Reference: ref,
					Changes:   changes,
				})
			}
		} else {
			diff.Added = append(diff.Added, ref)
		}
	}

	// Find removed
	for ref := range baselineMap {
		if _, exists := currentMap[ref]; !exists {
			diff.Removed = append(diff.Removed, ref)
		}
	}

	sort.Strings(diff.Added)
	sort.Strings(diff.Removed)
	return diff
}

func (m *BaselineManager) compareAccount(base, curr models.PSPAccount) []Change {
	var changes []Change

	// Compare DefaultAsset by value, not pointer address
	if (base.DefaultAsset == nil) != (curr.DefaultAsset == nil) ||
		(base.DefaultAsset != nil && curr.DefaultAsset != nil && *base.DefaultAsset != *curr.DefaultAsset) {
		var oldVal, newVal interface{}
		if base.DefaultAsset != nil {
			oldVal = *base.DefaultAsset
		}
		if curr.DefaultAsset != nil {
			newVal = *curr.DefaultAsset
		}
		changes = append(changes, Change{Field: "default_asset", OldValue: oldVal, NewValue: newVal})
	}
	if (base.Name == nil) != (curr.Name == nil) || (base.Name != nil && curr.Name != nil && *base.Name != *curr.Name) {
		var oldVal, newVal interface{}
		if base.Name != nil {
			oldVal = *base.Name
		}
		if curr.Name != nil {
			newVal = *curr.Name
		}
		changes = append(changes, Change{Field: "name", OldValue: oldVal, NewValue: newVal})
	}

	return changes
}

func (m *BaselineManager) comparePayments(baseline, current []models.PSPPayment) DataDiff {
	diff := DataDiff{
		BaselineCount: len(baseline),
		CurrentCount:  len(current),
	}

	baselineMap := make(map[string]models.PSPPayment)
	for _, p := range baseline {
		baselineMap[p.Reference] = p
	}

	currentMap := make(map[string]models.PSPPayment)
	for _, p := range current {
		currentMap[p.Reference] = p
	}

	// Find added and modified
	for ref, curr := range currentMap {
		if base, exists := baselineMap[ref]; exists {
			changes := m.comparePayment(base, curr)
			if len(changes) > 0 {
				diff.Modified = append(diff.Modified, ItemDiff{
					Reference: ref,
					Changes:   changes,
				})
			}
		} else {
			diff.Added = append(diff.Added, ref)
		}
	}

	// Find removed
	for ref := range baselineMap {
		if _, exists := currentMap[ref]; !exists {
			diff.Removed = append(diff.Removed, ref)
		}
	}

	sort.Strings(diff.Added)
	sort.Strings(diff.Removed)
	return diff
}

func (m *BaselineManager) comparePayment(base, curr models.PSPPayment) []Change {
	var changes []Change

	if base.Amount.Cmp(curr.Amount) != 0 {
		changes = append(changes, Change{Field: "amount", OldValue: base.Amount.String(), NewValue: curr.Amount.String()})
	}
	if base.Asset != curr.Asset {
		changes = append(changes, Change{Field: "asset", OldValue: base.Asset, NewValue: curr.Asset})
	}
	if base.Status != curr.Status {
		changes = append(changes, Change{Field: "status", OldValue: base.Status, NewValue: curr.Status})
	}
	if base.Type != curr.Type {
		changes = append(changes, Change{Field: "type", OldValue: base.Type, NewValue: curr.Type})
	}

	return changes
}

func (m *BaselineManager) compareBalances(baseline, current []models.PSPBalance) DataDiff {
	diff := DataDiff{
		BaselineCount: len(baseline),
		CurrentCount:  len(current),
	}

	// Create key for balance (account + asset)
	balanceKey := func(b models.PSPBalance) string {
		return b.AccountReference + ":" + b.Asset
	}

	baselineMap := make(map[string]models.PSPBalance)
	for _, b := range baseline {
		baselineMap[balanceKey(b)] = b
	}

	currentMap := make(map[string]models.PSPBalance)
	for _, b := range current {
		currentMap[balanceKey(b)] = b
	}

	// Find added and modified
	for key, curr := range currentMap {
		if base, exists := baselineMap[key]; exists {
			changes := m.compareBalance(base, curr)
			if len(changes) > 0 {
				diff.Modified = append(diff.Modified, ItemDiff{
					Reference: key,
					Changes:   changes,
				})
			}
		} else {
			diff.Added = append(diff.Added, key)
		}
	}

	// Find removed
	for key := range baselineMap {
		if _, exists := currentMap[key]; !exists {
			diff.Removed = append(diff.Removed, key)
		}
	}

	sort.Strings(diff.Added)
	sort.Strings(diff.Removed)
	return diff
}

func (m *BaselineManager) compareBalance(base, curr models.PSPBalance) []Change {
	var changes []Change

	if base.Amount.Cmp(curr.Amount) != 0 {
		changes = append(changes, Change{Field: "amount", OldValue: base.Amount.String(), NewValue: curr.Amount.String()})
	}

	return changes
}

func (d DataDiff) hasChanges() bool {
	return len(d.Added) > 0 || len(d.Removed) > 0 || len(d.Modified) > 0
}

func (m *BaselineManager) generateSummary(diff *BaselineDiff) string {
	var parts []string

	if diff.Accounts.hasChanges() {
		parts = append(parts, fmt.Sprintf("Accounts: +%d/-%d/~%d",
			len(diff.Accounts.Added), len(diff.Accounts.Removed), len(diff.Accounts.Modified)))
	}
	if diff.Payments.hasChanges() {
		parts = append(parts, fmt.Sprintf("Payments: +%d/-%d/~%d",
			len(diff.Payments.Added), len(diff.Payments.Removed), len(diff.Payments.Modified)))
	}
	if diff.Balances.hasChanges() {
		parts = append(parts, fmt.Sprintf("Balances: +%d/-%d/~%d",
			len(diff.Balances.Added), len(diff.Balances.Removed), len(diff.Balances.Modified)))
	}
	if diff.ExternalAccounts.hasChanges() {
		parts = append(parts, fmt.Sprintf("ExtAccounts: +%d/-%d/~%d",
			len(diff.ExternalAccounts.Added), len(diff.ExternalAccounts.Removed), len(diff.ExternalAccounts.Modified)))
	}

	if len(parts) == 0 {
		return "No changes detected"
	}
	return join(parts, " | ")
}

func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// Clear clears all baselines.
func (m *BaselineManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.baselines = make(map[string]*Baseline)
}

// Export exports a baseline as JSON.
func (m *BaselineManager) Export(baselineID string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	baseline, exists := m.baselines[baselineID]
	if !exists {
		return nil, fmt.Errorf("baseline not found: %s", baselineID)
	}

	return json.MarshalIndent(baseline, "", "  ")
}

// Import imports a baseline from JSON.
func (m *BaselineManager) Import(data []byte) (*Baseline, error) {
	var baseline Baseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, err
	}

	// Generate new ID to avoid conflicts
	baseline.ID = fmt.Sprintf("imported-%d", time.Now().UnixNano())

	m.mu.Lock()
	m.baselines[baseline.ID] = &baseline
	m.mu.Unlock()

	return &baseline, nil
}
