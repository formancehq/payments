package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type pluginEntry struct {
	Plugin      models.Plugin
	ConnectorID models.ConnectorID
	Provider    string
	Name        string

	StateCache map[string]cachedState
}

type cachedState struct {
	State     json.RawMessage `json:"state"`
	HasMore   bool            `json:"hasMore"`
	PageSize  int             `json:"pageSize"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

var (
	pluginsMu   sync.RWMutex
	pluginsByID map[string]pluginEntry = make(map[string]pluginEntry)
)

func newRouter(debug bool) *chi.Mux {
	r := chi.NewRouter()

	// Always respond JSON
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, req)
		})
	})

	r.Get("/providers", func(w http.ResponseWriter, _ *http.Request) {
		configs := registry.GetConfigs(debug)
		providers := make([]string, 0, len(configs))
		for k := range configs {
			providers = append(providers, k)
		}
		sort.Strings(providers)

		_ = json.NewEncoder(w).Encode(providers)
	})

	r.Get("/providers/{provider}/config-schema", func(w http.ResponseWriter, req *http.Request) {
		provider := chi.URLParam(req, "provider")
		if provider == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "provider is required"})
			return
		}

		conf, err := registry.GetConfig(provider)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "provider not found", "details": err.Error()})
			return
		}

		_ = json.NewEncoder(w).Encode(conf)
	})

	r.Post("/connectors", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Provider    string          `json:"provider"`
			Name        string          `json:"name"`
			ConnectorID string          `json:"connectorId,omitempty"`
			Config      json.RawMessage `json:"config"`
		}

		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "invalid JSON body", "details": err.Error()})
			return
		}

		if body.Provider == "" || body.Name == "" || len(body.Config) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "provider, name and config are required"})
			return
		}

		var cid models.ConnectorID
		if strings.TrimSpace(body.ConnectorID) != "" {
			parsed, err := models.ConnectorIDFromString(body.ConnectorID)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "invalid connectorId", "details": err.Error()})
				return
			}
			cid = parsed
		} else {
			cid = models.ConnectorID{Reference: uuid.Must(uuid.NewUUID()), Provider: strings.ToLower(body.Provider)}
		}

		logger := logging.NewDefaultLogger(os.Stdout, true, false, false)

		plugin, err := registry.GetPlugin(cid, logger, body.Provider, body.Name, body.Config)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "failed to instantiate plugin", "details": err.Error()})
			return
		}

		pluginsMu.Lock()
		pluginsByID[cid.String()] = pluginEntry{
			Plugin:      plugin,
			ConnectorID: cid,
			Provider:    body.Provider,
			Name:        body.Name,
			StateCache:  make(map[string]cachedState),
		}
		pluginsMu.Unlock()

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"connectorId": cid.String(),
			"provider":    body.Provider,
			"name":        body.Name,
		})
	})

	// removed singular /plugin endpoint; use /plugins and /plugins/{connectorId}

	r.Get("/connectors", func(w http.ResponseWriter, req *http.Request) {
		pluginsMu.RLock()
		list := make([]map[string]string, 0, len(pluginsByID))
		for _, e := range pluginsByID {
			list = append(list, map[string]string{
				"connectorId": e.ConnectorID.String(),
				"provider":    e.Provider,
				"name":        e.Name,
			})
		}
		pluginsMu.RUnlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"plugins": list})
	})

	r.Get("/connectors/{connectorId}", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "connectorId")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "connectorId is required"})
			return
		}
		pluginsMu.RLock()
		e, ok := pluginsByID[id]
		pluginsMu.RUnlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "plugin not found"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"connectorId": e.ConnectorID.String(),
			"provider":    e.Provider,
			"name":        e.Name,
		})
	})

	r.Get("/connectors/{connectorId}/state", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "connectorId")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "connectorId is required"})
			return
		}
		pluginsMu.RLock()
		e, ok := pluginsByID[id]
		pluginsMu.RUnlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "plugin not found"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fetch": map[string]cachedState{
				"accounts":          e.StateCache["fetch/accounts"],
				"external-accounts": e.StateCache["fetch/external-accounts"],
				"balances":          e.StateCache["fetch/balances"],
				"payments":          e.StateCache["fetch/payments"],
			},
		})
	})

	r.Post("/fetch/accounts", func(w http.ResponseWriter, req *http.Request) {
		// Only works when exactly one plugin is instantiated
		pluginsMu.RLock()
		var p models.Plugin
		if len(pluginsByID) == 1 {
			for _, e := range pluginsByID {
				p = e.Plugin
			}
		}
		pluginsMu.RUnlock()

		if p == nil {
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "requires exactly one plugin; use /connectors/{connectorId}/fetch/accounts"})
			return
		}

		var body models.FetchNextAccountsRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "invalid JSON body", "details": err.Error()})
			return
		}

		resp, err := p.FetchNextAccounts(req.Context(), body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "fetch accounts failed", "details": err.Error()})
			return
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	r.Post("/fetch/balances", func(w http.ResponseWriter, req *http.Request) {
		// Only works when exactly one plugin is instantiated
		pluginsMu.RLock()
		var p models.Plugin
		if len(pluginsByID) == 1 {
			for _, e := range pluginsByID {
				p = e.Plugin
			}
		}
		pluginsMu.RUnlock()

		if p == nil {
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "requires exactly one plugin; use /connectors/{connectorId}/fetch/balances"})
			return
		}

		var body struct {
			// Either provide account (as returned by /fetch/accounts) or fromPayload directly
			Account     json.RawMessage `json:"account"`
			FromPayload json.RawMessage `json:"fromPayload"`
			State       json.RawMessage `json:"state"`
			PageSize    int             `json:"pageSize"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "invalid JSON body", "details": err.Error()})
			return
		}

		var fromPayload json.RawMessage
		if len(body.FromPayload) != 0 {
			fromPayload = body.FromPayload
		} else if len(body.Account) != 0 {
			// Pass the account JSON straight through as the fromPayload expected by the plugin
			fromPayload = body.Account
		} else {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "account or fromPayload is required"})
			return
		}

		resp, err := p.FetchNextBalances(req.Context(), models.FetchNextBalancesRequest{
			FromPayload: fromPayload,
			State:       body.State,
			PageSize:    body.PageSize,
		})
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "fetch balances failed", "details": err.Error()})
			return
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	r.Post("/connectors/{connectorId}/fetch/accounts", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "connectorId")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "connectorId is required"})
			return
		}
		pluginsMu.RLock()
		e, ok := pluginsByID[id]
		pluginsMu.RUnlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "plugin not found"})
			return
		}

		var body struct {
			models.FetchNextAccountsRequest
			UseCachedState bool `json:"useCachedState"`
			Reset          bool `json:"reset"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "invalid JSON body", "details": err.Error()})
			return
		}

		if body.Reset {
			pluginsMu.Lock()
			entry := pluginsByID[id]
			delete(entry.StateCache, "fetch/accounts")
			pluginsByID[id] = entry
			pluginsMu.Unlock()
		}

		if body.State == nil && body.UseCachedState {
			pluginsMu.RLock()
			entry := pluginsByID[id]
			c, ok := entry.StateCache["fetch/accounts"]
			pluginsMu.RUnlock()
			if ok && len(c.State) != 0 {
				body.State = c.State
			}
		}

		resp, err := e.Plugin.FetchNextAccounts(req.Context(), body.FetchNextAccountsRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "fetch accounts failed", "details": err.Error()})
			return
		}

		pluginsMu.Lock()
		entry := pluginsByID[id]
		entry.StateCache["fetch/accounts"] = cachedState{
			State:     resp.NewState,
			HasMore:   resp.HasMore,
			PageSize:  body.PageSize,
			UpdatedAt: time.Now(),
		}
		pluginsMu.Unlock()
		_ = json.NewEncoder(w).Encode(resp)
	})

	r.Post("/connectors/{connectorId}/fetch/balances", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "connectorId")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "connectorId is required"})
			return
		}
		pluginsMu.RLock()
		e, ok := pluginsByID[id]
		pluginsMu.RUnlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "plugin not found"})
			return
		}

		var body struct {
			Account        json.RawMessage `json:"account"`
			FromPayload    json.RawMessage `json:"fromPayload"`
			State          json.RawMessage `json:"state"`
			PageSize       int             `json:"pageSize"`
			UseCachedState bool            `json:"useCachedState"`
			Reset          bool            `json:"reset"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "invalid JSON body", "details": err.Error()})
			return
		}
		var fromPayload json.RawMessage
		if len(body.FromPayload) != 0 {
			fromPayload = body.FromPayload
		} else if len(body.Account) != 0 {
			fromPayload = body.Account
		} else {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "account or fromPayload is required"})
			return
		}

		if body.Reset {
			pluginsMu.Lock()
			entry := pluginsByID[id]
			delete(entry.StateCache, "fetch/balances")
			pluginsMu.Unlock()
		}

		if body.State == nil && body.UseCachedState {
			pluginsMu.RLock()
			entry := pluginsByID[id]
			c, ok := entry.StateCache["fetch/balances"]
			pluginsMu.RUnlock()
			if ok && len(c.State) != 0 {
				body.State = c.State
			}
		}
		resp, err := e.Plugin.FetchNextBalances(req.Context(), models.FetchNextBalancesRequest{
			FromPayload: fromPayload,
			State:       body.State,
			PageSize:    body.PageSize,
		})
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "fetch balances failed", "details": err.Error()})
			return
		}
		pluginsMu.Lock()
		entry := pluginsByID[id]
		entry.StateCache["fetch/balances"] = cachedState{
			State:     resp.NewState,
			HasMore:   resp.HasMore,
			PageSize:  body.PageSize,
			UpdatedAt: time.Now(),
		}
		pluginsMu.Unlock()
		_ = json.NewEncoder(w).Encode(resp)
	})

	r.Post("/connectors/{connectorId}/fetch/external-accounts", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "connectorId")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "connectorId is required"})
			return
		}
		pluginsMu.RLock()
		e, ok := pluginsByID[id]
		pluginsMu.RUnlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "plugin not found"})
			return
		}

		var body struct {
			models.FetchNextExternalAccountsRequest
			UseCachedState bool `json:"useCachedState"`
			Reset          bool `json:"reset"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "invalid JSON body", "details": err.Error()})
			return
		}

		if body.Reset {
			pluginsMu.Lock()
			entry := pluginsByID[id]
			delete(entry.StateCache, "fetch/external-accounts")
			pluginsMu.Unlock()
		}

		if body.State == nil && body.UseCachedState {
			pluginsMu.RLock()
			entry := pluginsByID[id]
			c, ok := entry.StateCache["fetch/external-accounts"]
			pluginsMu.RUnlock()
			if ok && len(c.State) != 0 {
				body.State = c.State
			}
		}

		resp, err := e.Plugin.FetchNextExternalAccounts(req.Context(), body.FetchNextExternalAccountsRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "fetch external accounts failed", "details": err.Error()})
			return
		}

		pluginsMu.Lock()
		entry := pluginsByID[id]
		entry.StateCache["fetch/external-accounts"] = cachedState{
			State:     resp.NewState,
			HasMore:   resp.HasMore,
			PageSize:  body.PageSize,
			UpdatedAt: time.Now(),
		}
		pluginsMu.Unlock()
		_ = json.NewEncoder(w).Encode(resp)
	})

	r.Post("/connectors/{connectorId}/fetch/payments", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "connectorId")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "connectorId is required"})
			return
		}
		pluginsMu.RLock()
		e, ok := pluginsByID[id]
		pluginsMu.RUnlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "plugin not found"})
			return
		}

		var body struct {
			models.FetchNextPaymentsRequest
			UseCachedState bool `json:"useCachedState"`
			Reset          bool `json:"reset"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "invalid JSON body", "details": err.Error()})
			return
		}

		if body.Reset {
			pluginsMu.Lock()
			entry := pluginsByID[id]
			delete(entry.StateCache, "fetch/payments")
			pluginsMu.Unlock()
		}

		if body.State == nil && body.UseCachedState {
			pluginsMu.RLock()
			entry := pluginsByID[id]
			c, ok := entry.StateCache["fetch/payments"]
			pluginsMu.RUnlock()
			if ok && len(c.State) != 0 {
				body.State = c.State
			}
		}

		resp, err := e.Plugin.FetchNextPayments(req.Context(), body.FetchNextPaymentsRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "fetch payments failed", "details": err.Error()})
			return
		}

		pluginsMu.Lock()
		entry := pluginsByID[id]
		entry.StateCache["fetch/payments"] = cachedState{
			State:     resp.NewState,
			HasMore:   resp.HasMore,
			PageSize:  body.PageSize,
			UpdatedAt: time.Now(),
		}
		pluginsMu.Unlock()
		_ = json.NewEncoder(w).Encode(resp)
	})

	return r
}
