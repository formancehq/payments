# RFC: Migration Temporal vers PostgreSQL Job Queue

**Status**: Draft
**Auteur**: Équipe Payments
**Date**: 2026-01-17
**Version**: 1.0

---

## Table des Matières

1. [Résumé Exécutif](#1-résumé-exécutif)
2. [Contexte et Motivation](#2-contexte-et-motivation)
3. [Architecture Actuelle](#3-architecture-actuelle)
4. [Architecture Cible](#4-architecture-cible)
5. [Spécifications Techniques](#5-spécifications-techniques)
6. [Plan de Migration](#6-plan-de-migration)
7. [Stratégie de Tests](#7-stratégie-de-tests)
8. [Risques et Mitigations](#8-risques-et-mitigations)
9. [Critères d'Acceptation](#9-critères-dacceptation)
10. [Estimation et Planning](#10-estimation-et-planning)

---

## 1. Résumé Exécutif

### Objectif

Remplacer Temporal par une solution de job queue native PostgreSQL pour réduire la complexité opérationnelle et les coûts d'infrastructure du projet Payments, tout en maintenant les garanties de fiabilité et de scalabilité.

### Décision

Implémenter une architecture **Multi-Instance + PostgreSQL SKIP LOCKED** dès le départ pour permettre le scaling horizontal sans dépendance externe supplémentaire.

### Bénéfices Attendus

| Métrique | Avant | Après | Gain |
|----------|-------|-------|------|
| Pods d'infrastructure | 3-5 (Temporal) | 0 | -100% |
| Bases de données | 2 (Payments + Temporal) | 1 (Payments) | -50% |
| Dépendances externes | Temporal Server | Aucune | Simplicité |
| Onboarding contributeurs | Complexe | Simple | Adoption OSS |
| Coût cloud mensuel estimé | ~$200-400 | ~$50-100 | -75% |

---

## 2. Contexte et Motivation

### 2.1 Problématique Actuelle

Payments utilise **Temporal** comme moteur d'orchestration pour gérer les workflows asynchrones (polling, création de paiements, webhooks, etc.). Bien que Temporal soit un excellent outil, son utilisation dans le contexte d'un projet **Open Source avec contraintes de coûts** pose plusieurs problèmes :

1. **Coût d'infrastructure élevé**
   - Temporal Server nécessite 3-5 pods minimum
   - Base de données dédiée pour l'historique des workflows
   - Ressources mémoire/CPU significatives

2. **Barrière à l'adoption OSS**
   - Les utilisateurs doivent déployer Temporal avant de tester Payments
   - Documentation et expertise Temporal requises
   - Complexité de la stack de développement local

3. **Surqualification**
   - Payments n'utilise pas les fonctionnalités avancées de Temporal (Signals, Queries, Versioning)
   - Les workflows sont relativement simples (pas de saga complexes)
   - Les besoins réels : retry, scheduling, state persistence

### 2.2 Analyse de l'Utilisation Actuelle de Temporal

#### Workflows Identifiés (31 au total)

| Catégorie | Workflows | Complexité |
|-----------|-----------|------------|
| Fetch/Polling | FetchAccounts, FetchBalances, FetchPayments, FetchExternalAccounts, FetchOthers | Moyenne |
| Paiements | CreateTransfer, CreatePayout, ReverseTransfer, ReversePayout, PollTransfer, PollPayout | Faible |
| Connecteurs | InstallConnector, UninstallConnector, ResetConnector | Haute |
| Webhooks | HandleWebhooks, CreateWebhooks, StoreWebhookTranslation | Moyenne |
| Open Banking | CompleteUserLink, DeletePSU, DeletePSUConnector, DeleteConnection | Faible-Moyenne |
| Scheduling | TerminateSchedules, TerminateWorkflows, UpdateSchedulePollingPeriod | Moyenne |
| Outbox | OutboxPublisher, OutboxCleanup | Faible |

#### Activities Identifiées (162+ fichiers)

- **Storage** (~60) : Opérations CRUD en base de données
- **Plugin** (~30) : Appels aux providers externes (Stripe, Wise, etc.)
- **Events** (~30) : Publication d'événements
- **Temporal Schedule** (~5) : Gestion des schedules (à supprimer)

#### Patterns Temporal Utilisés

| Pattern | Utilisé | Remplaçable |
|---------|---------|-------------|
| Infinite retry + exponential backoff | ✅ Oui | ✅ Librairie Go |
| Continue-as-new | ✅ Oui | ✅ Pagination DB |
| Child workflows (ABANDON) | ✅ Oui | ✅ Goroutines + errgroup |
| Workflow.Sleep | ✅ Oui | ✅ time.Sleep / ticker |
| Search Attributes | ✅ Oui | ✅ Colonnes DB |
| Schedules | ✅ Oui | ✅ gocron / in-process |
| Signals | ❌ Non | N/A |
| Queries | ❌ Non | N/A |
| Workflow Versioning | ❌ Non | N/A |

**Conclusion** : Aucune fonctionnalité avancée de Temporal n'est utilisée. La migration est techniquement faisable.

---

## 3. Architecture Actuelle

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              API Layer                                   │
│                         (chi router, v3 API)                            │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────────────┐
│                               Engine                                     │
│                    (engine.go - 1667 lignes)                            │
│  - InstallConnector, UninstallConnector, ResetConnector                 │
│  - CreateTransfer, CreatePayout, ReverseTransfer, ReversePayout         │
│  - HandleWebhook, ForwardBankAccount                                    │
│  - PSU operations (Forward, Delete, Link)                               │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │    Temporal Client      │
                    │  ExecuteWorkflow(...)   │
                    └────────────┬────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────────────┐
│                          Temporal Server                                 │
│  - Workflow orchestration                                               │
│  - Task queue management                                                │
│  - Retry & timeout handling                                             │
│  - Schedule management                                                  │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                  │
     ┌────────▼────────┐ ┌──────▼──────┐ ┌────────▼────────┐
     │    Worker 1     │ │   Worker 2  │ │    Worker N     │
     │  (72 workflows) │ │             │ │                 │
     │  (162 activities)│ │             │ │                 │
     └────────┬────────┘ └──────┬──────┘ └────────┬────────┘
              │                 │                  │
              └─────────────────┼──────────────────┘
                                │
                    ┌───────────▼───────────┐
                    │      PostgreSQL       │
                    │   (Payments Data)     │
                    └───────────────────────┘
```

### Dépendances Actuelles

```
go.temporal.io/sdk v1.38.0
go.temporal.io/api v1.54.0
```

### Fichiers Impactés

```
internal/connectors/engine/
├── engine.go                 # Point d'entrée (1667 lignes)
├── workers.go                # Worker pool Temporal
├── workflow/                 # 72 fichiers
│   ├── context.go           # Retry policies
│   ├── install_connector.go
│   ├── uninstall_connector.go
│   ├── fetch_accounts.go
│   ├── create_transfer.go
│   └── ...
└── activities/               # 162+ fichiers
    ├── plugin_*.go          # Appels plugins
    ├── storage_*.go         # Opérations DB
    └── events_*.go          # Publication events
```

---

## 4. Architecture Cible

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              API Layer                                   │
│                         (chi router, v3 API)                            │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────────────┐
│                          Engine (simplifié)                              │
│  - Opérations synchrones directes                                       │
│  - Enqueue jobs asynchrones via JobQueue                                │
└───────────────┬─────────────────────────────────┬───────────────────────┘
                │                                 │
    ┌───────────▼───────────┐         ┌──────────▼──────────┐
    │      Job Queue        │         │      Scheduler      │
    │   (PostgreSQL)        │         │    (In-Process)     │
    │                       │         │                     │
    │ - jobs table          │         │ - gocron            │
    │ - FOR UPDATE          │         │ - Polling periods   │
    │   SKIP LOCKED         │         │ - Outbox publisher  │
    └───────────┬───────────┘         └──────────┬──────────┘
                │                                 │
                └─────────────┬───────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────────────┐
│                         Worker Pool                                      │
│  - N goroutines par instance                                            │
│  - Handlers par type de job                                             │
│  - State machine pour workflows complexes                               │
└─────────────────────────────┬───────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────────────┐
│                          PostgreSQL                                      │
│  - Tables existantes (payments, accounts, etc.)                         │
│  - Nouvelle table: jobs                                                 │
│  - Nouvelle table: schedules                                            │
└─────────────────────────────────────────────────────────────────────────┘
```

### Scaling Horizontal

```
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│   Instance 1     │  │   Instance 2     │  │   Instance 3     │
│   (20 workers)   │  │   (20 workers)   │  │   (20 workers)   │
│                  │  │                  │  │                  │
│ Pod: payments-0  │  │ Pod: payments-1  │  │ Pod: payments-2  │
└────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘
         │                     │                     │
         └─────────────────────┼─────────────────────┘
                               │
                               │ FOR UPDATE SKIP LOCKED
                               │ (Coordination native PostgreSQL)
                               │
                    ┌──────────▼──────────┐
                    │     PostgreSQL      │
                    │    (jobs table)     │
                    └─────────────────────┘
```

---

## 5. Spécifications Techniques

### 5.1 Schema Base de Données

#### Table `jobs`

```sql
CREATE TABLE jobs (
    -- Identité
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type VARCHAR(100) NOT NULL,
    connector_id TEXT,

    -- Payload
    payload JSONB NOT NULL,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    -- Valeurs: pending, processing, completed, failed, cancelled

    -- Locking (multi-instance)
    locked_by VARCHAR(100),
    locked_at TIMESTAMPTZ,

    -- Retry management
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 10,
    next_retry_at TIMESTAMPTZ,
    last_error TEXT,

    -- State machine (workflows multi-étapes)
    workflow_state JSONB,
    current_step VARCHAR(100),

    -- Tracking
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Idempotency
    idempotency_key VARCHAR(255),

    -- Constraints
    CONSTRAINT jobs_idempotency_key_unique UNIQUE (idempotency_key)
);

-- Index pour le worker (performance critique)
CREATE INDEX idx_jobs_pending
    ON jobs (status, next_retry_at, created_at)
    WHERE status IN ('pending', 'failed');

-- Index pour le cleanup
CREATE INDEX idx_jobs_completed
    ON jobs (completed_at)
    WHERE status = 'completed';

-- Index pour filtering par connector
CREATE INDEX idx_jobs_connector
    ON jobs (connector_id, status);

-- Index pour orphan detection
CREATE INDEX idx_jobs_locked
    ON jobs (locked_at)
    WHERE status = 'processing';
```

#### Table `schedules`

```sql
CREATE TABLE schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    connector_id TEXT,

    -- Schedule config
    job_type VARCHAR(100) NOT NULL,
    job_payload JSONB NOT NULL DEFAULT '{}',
    interval_seconds INT NOT NULL,

    -- Status
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,

    -- Tracking
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT schedules_name_connector_unique UNIQUE (name, connector_id)
);

CREATE INDEX idx_schedules_next_run
    ON schedules (next_run_at)
    WHERE enabled = true;
```

### 5.2 Interface JobQueue

```go
package jobqueue

import (
    "context"
    "time"
)

// Job représente un job dans la queue
type Job struct {
    ID             uuid.UUID
    JobType        string
    ConnectorID    *string
    Payload        json.RawMessage
    Status         JobStatus
    LockedBy       *string
    LockedAt       *time.Time
    RetryCount     int
    MaxRetries     int
    NextRetryAt    *time.Time
    LastError      *string
    WorkflowState  json.RawMessage
    CurrentStep    *string
    CreatedAt      time.Time
    StartedAt      *time.Time
    CompletedAt    *time.Time
    IdempotencyKey *string
}

type JobStatus string

const (
    JobStatusPending    JobStatus = "pending"
    JobStatusProcessing JobStatus = "processing"
    JobStatusCompleted  JobStatus = "completed"
    JobStatusFailed     JobStatus = "failed"
    JobStatusCancelled  JobStatus = "cancelled"
)

// JobHandler interface pour les handlers de jobs
type JobHandler interface {
    // Handle exécute le job. Retourne une erreur pour retry.
    Handle(ctx context.Context, job Job) error

    // Type retourne le type de job géré
    Type() string

    // MaxRetries retourne le nombre max de retries (0 = défaut)
    MaxRetries() int

    // Timeout retourne le timeout d'exécution
    Timeout() time.Duration
}

// Queue interface principale
type Queue interface {
    // Enqueue ajoute un job à la queue
    Enqueue(ctx context.Context, jobType string, payload any, opts ...EnqueueOption) (*Job, error)

    // RegisterHandler enregistre un handler pour un type de job
    RegisterHandler(handler JobHandler)

    // Start démarre les workers
    Start(ctx context.Context) error

    // Stop arrête gracieusement les workers
    Stop(ctx context.Context) error

    // Stats retourne les statistiques de la queue
    Stats(ctx context.Context) (*QueueStats, error)
}

// EnqueueOption options pour l'enqueue
type EnqueueOption func(*enqueueOptions)

func WithConnectorID(id string) EnqueueOption
func WithIdempotencyKey(key string) EnqueueOption
func WithMaxRetries(n int) EnqueueOption
func WithDelay(d time.Duration) EnqueueOption
func WithWorkflowState(state any) EnqueueOption
```

### 5.3 Interface Scheduler

```go
package scheduler

import (
    "context"
    "time"
)

// Schedule représente une tâche planifiée
type Schedule struct {
    ID              uuid.UUID
    Name            string
    ConnectorID     *string
    JobType         string
    JobPayload      json.RawMessage
    IntervalSeconds int
    Enabled         bool
    LastRunAt       *time.Time
    NextRunAt       *time.Time
}

// Scheduler interface
type Scheduler interface {
    // Create crée un nouveau schedule
    Create(ctx context.Context, schedule Schedule) error

    // Update met à jour un schedule existant
    Update(ctx context.Context, id uuid.UUID, interval time.Duration) error

    // Delete supprime un schedule
    Delete(ctx context.Context, id uuid.UUID) error

    // DeleteByConnector supprime tous les schedules d'un connector
    DeleteByConnector(ctx context.Context, connectorID string) error

    // List liste les schedules (optionnel: par connector)
    List(ctx context.Context, connectorID *string) ([]Schedule, error)

    // Start démarre le scheduler
    Start(ctx context.Context) error

    // Stop arrête le scheduler
    Stop(ctx context.Context) error
}
```

### 5.4 State Machine pour Workflows Complexes

```go
package workflow

// WorkflowState état générique pour workflows multi-étapes
type WorkflowState struct {
    CurrentPhase   string            `json:"current_phase"`
    CompletedSteps []string          `json:"completed_steps"`
    FailedSteps    map[string]string `json:"failed_steps"` // step -> error
    Data           json.RawMessage   `json:"data"`         // Données spécifiques
    StartedAt      time.Time         `json:"started_at"`
    UpdatedAt      time.Time         `json:"updated_at"`
}

// Phase définit une phase d'un workflow
type Phase struct {
    Name     string
    Steps    []string
    Parallel bool // true = étapes en parallèle, false = séquentiel
}

// UninstallConnectorWorkflow phases pour l'uninstall
var UninstallConnectorPhases = []Phase{
    {
        Name:     "terminate",
        Steps:    []string{"terminate_schedules", "terminate_active_jobs"},
        Parallel: true,
    },
    {
        Name: "delete_data",
        Steps: []string{
            "delete_events_sent",
            "delete_instances",
            "delete_tasks",
            "delete_bank_accounts_related",
            "delete_accounts",
            "delete_payments",
            "delete_balances",
            "delete_states",
            "delete_webhooks_configs",
            "delete_webhooks",
        },
        Parallel: true,
    },
    {
        Name:     "cleanup",
        Steps:    []string{"plugin_uninstall", "delete_connector_entry"},
        Parallel: false,
    },
}
```

### 5.5 Handlers à Implémenter

| Handler | Remplace Workflow | Complexité |
|---------|-------------------|------------|
| `CreateTransferHandler` | `create_transfer.go` | Faible |
| `CreatePayoutHandler` | `create_payout.go` | Faible |
| `ReverseTransferHandler` | `reverse_transfer.go` | Faible |
| `ReversePayoutHandler` | `reverse_payout.go` | Faible |
| `PollTransferHandler` | `poll_transfer.go` | Faible |
| `PollPayoutHandler` | `poll_payout.go` | Faible |
| `CreateBankAccountHandler` | `create_bank_account.go` | Faible |
| `FetchAccountsHandler` | `fetch_accounts.go` | Moyenne |
| `FetchBalancesHandler` | `fetch_balances.go` | Moyenne |
| `FetchPaymentsHandler` | `fetch_payments.go` | Moyenne |
| `FetchExternalAccountsHandler` | `fetch_external_accounts.go` | Moyenne |
| `FetchOthersHandler` | `fetch_others.go` | Moyenne |
| `InstallConnectorHandler` | `install_connector.go` | Haute |
| `UninstallConnectorHandler` | `uninstall_connector.go` | **Très Haute** |
| `ResetConnectorHandler` | `reset_connector.go` | Haute |
| `HandleWebhooksHandler` | `handle_webhooks.go` | Moyenne |
| `CreateWebhooksHandler` | `create_webhooks.go` | Moyenne |
| `CompleteUserLinkHandler` | `complete_user_link.go` | Faible |
| `DeletePSUHandler` | `delete_psu.go` | Moyenne |
| `DeletePSUConnectorHandler` | `delete_psu_connector.go` | Faible |
| `DeleteConnectionHandler` | `delete_connection.go` | Moyenne |
| `OutboxPublisherHandler` | `outbox_publisher.go` | Faible |
| `OutboxCleanupHandler` | `outbox_cleanup.go` | Faible |

---

## 6. Plan de Migration

### 6.1 Vue d'Ensemble des Phases

```
Phase 0          Phase 1           Phase 2           Phase 3
Préparation      Infrastructure    Migration         Migration
                 de Base           Workflows         Workflows
                                   Simples           Complexes
    │                │                 │                 │
    ▼                ▼                 ▼                 ▼
┌────────┐      ┌────────┐        ┌────────┐       ┌────────┐
│ Tests  │      │JobQueue│        │Handlers│       │ State  │
│ Baseline│ ──▶ │Scheduler│  ──▶  │ Simples│  ──▶  │Machine │
│ Feature│      │ Tables │        │ Fetch  │       │Install │
│ Flags  │      │        │        │        │       │Uninstall│
└────────┘      └────────┘        └────────┘       └────────┘
                                                        │
    ┌───────────────────────────────────────────────────┘
    ▼
Phase 4           Phase 5           Phase 6
Migration         Cleanup &         Déploiement
Webhooks &        Optimisation      Production
Open Banking
    │                │                 │
    ▼                ▼                 ▼
┌────────┐      ┌────────┐        ┌────────┐
│Webhooks│      │ Remove │        │ Rollout│
│  PSU   │ ──▶  │Temporal│  ──▶   │ Graduel│
│Handlers│      │  Code  │        │        │
└────────┘      └────────┘        └────────┘
```

### 6.2 Phase 0 : Préparation (1 semaine)

**Objectifs** :
- Établir une baseline de tests
- Mettre en place l'infrastructure de feature flags
- Documenter le comportement actuel

**Tâches** :

- [ ] Audit complet des tests existants
  - [ ] Lister tous les tests workflow (`*_test.go` dans `workflow/`)
  - [ ] Lister tous les tests activities (`*_test.go` dans `activities/`)
  - [ ] Identifier les tests d'intégration
  - [ ] Mesurer la couverture actuelle

- [ ] Créer des tests de non-régression end-to-end
  - [ ] Test: Installation d'un connecteur (Stripe mock)
  - [ ] Test: Polling complet (accounts, balances, payments)
  - [ ] Test: Création de transfert end-to-end
  - [ ] Test: Webhook reception et traitement
  - [ ] Test: Uninstall complet avec cleanup

- [ ] Implémenter le système de feature flags
  ```go
  type FeatureFlags struct {
      UseNewJobQueue bool `env:"FF_USE_NEW_JOB_QUEUE" default:"false"`
  }
  ```

- [ ] Documentation du comportement actuel
  - [ ] Diagrammes de séquence pour chaque workflow
  - [ ] Documenter les retry policies
  - [ ] Documenter les timeouts

**Livrables** :
- Suite de tests E2E complète
- Feature flag `FF_USE_NEW_JOB_QUEUE`
- Documentation des workflows actuels

### 6.3 Phase 1 : Infrastructure de Base (2-3 semaines)

**Objectifs** :
- Créer les nouvelles tables PostgreSQL
- Implémenter la JobQueue de base
- Implémenter le Scheduler

**Tâches** :

- [ ] Migrations base de données
  - [ ] Migration: Créer table `jobs`
  - [ ] Migration: Créer table `schedules`
  - [ ] Migration: Ajouter index de performance
  - [ ] Tests des migrations (up/down)

- [ ] Implémenter `jobqueue` package
  - [ ] `Queue` struct et constructeur
  - [ ] `Enqueue()` avec options
  - [ ] `fetchJob()` avec SKIP LOCKED
  - [ ] `workerLoop()` avec graceful shutdown
  - [ ] `processJob()` avec timeout et recovery
  - [ ] `retryOrFail()` avec exponential backoff
  - [ ] `cleanupOrphanedJobs()` pour jobs bloqués
  - [ ] Tests unitaires (>90% couverture)

- [ ] Implémenter `scheduler` package
  - [ ] `Scheduler` struct avec gocron
  - [ ] `Create()`, `Update()`, `Delete()`
  - [ ] `Start()` / `Stop()` lifecycle
  - [ ] Intégration avec JobQueue
  - [ ] Tests unitaires (>90% couverture)

- [ ] Tests d'intégration infrastructure
  - [ ] Test: Multi-instance avec SKIP LOCKED
  - [ ] Test: Retry avec backoff
  - [ ] Test: Orphan job recovery
  - [ ] Test: Schedule trigger
  - [ ] Benchmark: throughput jobs/sec

**Livrables** :
- Package `internal/jobqueue/`
- Package `internal/scheduler/`
- Migrations SQL
- Tests avec >90% couverture

### 6.4 Phase 2 : Migration Workflows Simples (2-3 semaines)

**Objectifs** :
- Migrer les workflows à une seule étape
- Migrer les workflows de fetch/polling
- Validation avec feature flag

**Tâches** :

- [ ] Handlers opérations de paiement
  - [ ] `CreateTransferHandler`
  - [ ] `CreatePayoutHandler`
  - [ ] `ReverseTransferHandler`
  - [ ] `ReversePayoutHandler`
  - [ ] `PollTransferHandler`
  - [ ] `PollPayoutHandler`
  - [ ] `CreateBankAccountHandler`
  - [ ] Tests unitaires pour chaque handler

- [ ] Handlers fetch/polling
  - [ ] `FetchAccountsHandler` (avec pagination state)
  - [ ] `FetchBalancesHandler`
  - [ ] `FetchPaymentsHandler`
  - [ ] `FetchExternalAccountsHandler`
  - [ ] `FetchOthersHandler`
  - [ ] Tests unitaires avec mock plugins

- [ ] Intégration dans Engine
  - [ ] Modifier `engine.go` pour utiliser JobQueue (si feature flag)
  - [ ] Mapping ancien → nouveau pour chaque opération
  - [ ] Tests d'intégration

- [ ] Handlers outbox
  - [ ] `OutboxPublisherHandler`
  - [ ] `OutboxCleanupHandler`
  - [ ] Migration des schedules système

**Livrables** :
- 15 handlers implémentés et testés
- Engine modifié avec dual-path (Temporal/JobQueue)
- Tests de non-régression passants

### 6.5 Phase 3 : Migration Workflows Complexes (3-4 semaines)

**Objectifs** :
- Implémenter la state machine
- Migrer InstallConnector, UninstallConnector, ResetConnector

**Tâches** :

- [ ] Implémenter le framework state machine
  - [ ] `WorkflowState` struct
  - [ ] `Phase` et `Step` definitions
  - [ ] `executePhase()` (parallel/sequential)
  - [ ] `checkpoint()` pour persistence
  - [ ] `resume()` pour reprise après crash
  - [ ] Tests unitaires state machine

- [ ] `InstallConnectorHandler`
  - [ ] Phase 1: Plugin install
  - [ ] Phase 2: Create schedules
  - [ ] Phase 3: Initial fetch trigger
  - [ ] Tests avec différents providers

- [ ] `UninstallConnectorHandler` (le plus critique)
  - [ ] Phase 1: Terminate schedules & active jobs
  - [ ] Phase 2: Delete all data (parallel)
  - [ ] Phase 3: Plugin uninstall & cleanup
  - [ ] Tests de crash recovery
  - [ ] Tests de cleanup complet

- [ ] `ResetConnectorHandler`
  - [ ] Orchestration Uninstall → Install
  - [ ] Préservation de la config
  - [ ] Tests end-to-end

**Livrables** :
- Framework state machine
- 3 handlers complexes
- Tests de crash recovery
- Tests de performance (temps d'uninstall)

### 6.6 Phase 4 : Migration Webhooks & Open Banking (2 semaines)

**Objectifs** :
- Migrer les handlers webhooks
- Migrer les handlers PSU/Open Banking

**Tâches** :

- [ ] Handlers webhooks
  - [ ] `HandleWebhooksHandler`
  - [ ] `CreateWebhooksHandler`
  - [ ] `StoreWebhookTranslationHandler`
  - [ ] Tests avec différents types de webhooks

- [ ] Handlers Open Banking
  - [ ] `CompleteUserLinkHandler`
  - [ ] `DeletePSUHandler`
  - [ ] `DeletePSUConnectorHandler`
  - [ ] `DeleteConnectionHandler`
  - [ ] `FetchOpenBankingDataHandler`
  - [ ] Tests avec mock providers

**Livrables** :
- Tous les handlers restants
- Couverture complète des fonctionnalités

### 6.7 Phase 5 : Cleanup & Optimisation (1-2 semaines)

**Objectifs** :
- Supprimer le code Temporal
- Optimiser les performances
- Finaliser la documentation

**Tâches** :

- [ ] Suppression code Temporal
  - [ ] Supprimer `internal/connectors/engine/workflow/`
  - [ ] Supprimer `internal/connectors/engine/activities/`
  - [ ] Supprimer `engine/workers.go` (version Temporal)
  - [ ] Supprimer dépendances `go.mod`
  - [ ] Nettoyer les imports

- [ ] Optimisations
  - [ ] Benchmark final throughput
  - [ ] Optimisation requêtes SQL si nécessaire
  - [ ] Tuning worker count
  - [ ] Tuning batch sizes

- [ ] Documentation
  - [ ] Mise à jour README
  - [ ] Guide de migration pour utilisateurs existants
  - [ ] Documentation API interne
  - [ ] Runbook opérationnel

- [ ] Feature flag cleanup
  - [ ] Retirer le feature flag
  - [ ] Supprimer le code dual-path

**Livrables** :
- Codebase sans Temporal
- Documentation complète
- Benchmarks de performance

### 6.8 Phase 6 : Déploiement Production (1-2 semaines)

**Objectifs** :
- Rollout graduel en production
- Monitoring et validation

**Tâches** :

- [ ] Préparation
  - [ ] Plan de rollback documenté
  - [ ] Alertes monitoring configurées
  - [ ] Runbook incidents

- [ ] Rollout graduel
  - [ ] Étape 1: Environnement staging complet
  - [ ] Étape 2: 10% du trafic production
  - [ ] Étape 3: 50% du trafic production
  - [ ] Étape 4: 100% du trafic production

- [ ] Validation post-déploiement
  - [ ] Vérifier métriques de performance
  - [ ] Vérifier taux d'erreur
  - [ ] Vérifier temps de traitement jobs
  - [ ] Validation utilisateurs

**Livrables** :
- Déploiement production stable
- Métriques de validation
- Documentation incidents (si applicable)

---

## 7. Stratégie de Tests

### 7.1 Pyramide de Tests

```
                    ┌───────────┐
                    │    E2E    │  ~10 tests
                    │  (Slow)   │
                    ├───────────┤
                    │Integration│  ~50 tests
                    │  (Medium) │
                    ├───────────┤
                    │   Unit    │  ~200 tests
                    │  (Fast)   │
                    └───────────┘
```

### 7.2 Tests Unitaires (Obligatoires)

**Couverture minimale requise : 90%**

#### JobQueue Tests

```go
// internal/jobqueue/queue_test.go

func TestQueue_Enqueue(t *testing.T) {
    // Test enqueue basic
    // Test enqueue with idempotency key (duplicate rejected)
    // Test enqueue with delay
    // Test enqueue with custom max retries
}

func TestQueue_FetchJob(t *testing.T) {
    // Test fetch returns oldest pending job
    // Test fetch skips locked jobs (SKIP LOCKED)
    // Test fetch returns nil when queue empty
    // Test fetch respects next_retry_at
}

func TestQueue_ProcessJob(t *testing.T) {
    // Test successful processing -> completed
    // Test handler error -> retry
    // Test max retries exceeded -> failed
    // Test handler panic -> recovered and failed
    // Test context timeout -> retry
}

func TestQueue_RetryBackoff(t *testing.T) {
    // Test exponential backoff: 1s, 2s, 4s, 8s...
    // Test max backoff cap (5 minutes)
}

func TestQueue_OrphanedJobs(t *testing.T) {
    // Test jobs locked > 15min are released
    // Test released jobs are retried
}

func TestQueue_MultiInstance(t *testing.T) {
    // Test two instances don't process same job
    // Test jobs distributed across instances
}
```

#### Handler Tests

```go
// internal/handlers/create_transfer_test.go

func TestCreateTransferHandler_Success(t *testing.T) {
    // Mock storage, mock plugin
    // Verify plugin called with correct params
    // Verify payment stored
    // Verify task updated to success
}

func TestCreateTransferHandler_PluginError(t *testing.T) {
    // Test retryable error -> return error
    // Test non-retryable error -> return wrapped error
}

func TestCreateTransferHandler_StorageError(t *testing.T) {
    // Test storage error -> return error for retry
}
```

#### State Machine Tests

```go
// internal/workflow/state_machine_test.go

func TestStateMachine_ExecutePhase_Parallel(t *testing.T) {
    // Test all steps executed in parallel
    // Test partial failure (some steps fail)
    // Test all steps fail
}

func TestStateMachine_ExecutePhase_Sequential(t *testing.T) {
    // Test steps executed in order
    // Test stops on first failure
}

func TestStateMachine_Checkpoint(t *testing.T) {
    // Test state saved after each step
    // Test resume from checkpoint
}

func TestStateMachine_Resume(t *testing.T) {
    // Test skips completed steps
    // Test retries failed steps
}
```

### 7.3 Tests d'Intégration (Obligatoires)

**Utilisent une vraie base PostgreSQL (testcontainers)**

```go
// internal/jobqueue/integration_test.go

func TestIntegration_JobQueue_FullCycle(t *testing.T) {
    // Setup: PostgreSQL container
    // Enqueue job
    // Start worker
    // Wait for completion
    // Verify job status = completed
}

func TestIntegration_JobQueue_MultiInstance(t *testing.T) {
    // Setup: PostgreSQL container
    // Enqueue 100 jobs
    // Start 3 worker instances
    // Verify all jobs processed exactly once
    // Verify distribution across instances
}

func TestIntegration_Scheduler_TriggerJob(t *testing.T) {
    // Setup: PostgreSQL container
    // Create schedule (interval: 1 second)
    // Start scheduler
    // Wait 3 seconds
    // Verify 3 jobs created
}

func TestIntegration_UninstallConnector_FullCleanup(t *testing.T) {
    // Setup: PostgreSQL with test data
    // Create connector with accounts, payments, etc.
    // Run UninstallConnectorHandler
    // Verify ALL related data deleted
    // Verify connector entry deleted
}
```

### 7.4 Tests End-to-End (Obligatoires)

**Tests avec l'API HTTP complète**

```go
// tests/e2e/connector_lifecycle_test.go

func TestE2E_ConnectorLifecycle(t *testing.T) {
    // 1. POST /v3/connectors - Install Stripe (mock)
    // 2. Verify connector appears in GET /v3/connectors
    // 3. Wait for initial fetch (accounts appear)
    // 4. POST /v3/payment-initiations - Create transfer
    // 5. Verify payment created
    // 6. DELETE /v3/connectors/{id} - Uninstall
    // 7. Verify all data cleaned up
}

func TestE2E_WebhookProcessing(t *testing.T) {
    // 1. Install connector
    // 2. POST /v3/connectors/{id}/webhooks - Simulate webhook
    // 3. Verify payment/account updated
}

func TestE2E_CrashRecovery(t *testing.T) {
    // 1. Start uninstall
    // 2. Kill process mid-execution
    // 3. Restart process
    // 4. Verify uninstall completes
    // 5. Verify no data corruption
}
```

### 7.5 Tests de Performance (Recommandés)

```go
// tests/benchmark/jobqueue_bench_test.go

func BenchmarkQueue_Enqueue(b *testing.B) {
    // Target: > 10,000 enqueues/sec
}

func BenchmarkQueue_ProcessJob(b *testing.B) {
    // Target: > 1,000 jobs/sec (single instance)
}

func BenchmarkQueue_MultiInstance(b *testing.B) {
    // Target: Linear scaling with instances
}
```

### 7.6 Tests de Non-Régression

**Comparaison Temporal vs Nouvelle Implémentation**

```go
// tests/regression/compare_test.go

func TestRegression_CreateTransfer_SameResult(t *testing.T) {
    // Run with Temporal
    resultTemporal := runWithTemporal(createTransferInput)

    // Run with JobQueue
    resultJobQueue := runWithJobQueue(createTransferInput)

    // Compare results
    assert.Equal(t, resultTemporal.PaymentID, resultJobQueue.PaymentID)
    assert.Equal(t, resultTemporal.Status, resultJobQueue.Status)
}
```

### 7.7 Matrice de Tests par Phase

| Phase | Unit Tests | Integration Tests | E2E Tests | Couverture Min |
|-------|------------|-------------------|-----------|----------------|
| Phase 1 | JobQueue, Scheduler | Multi-instance | - | 90% |
| Phase 2 | Tous handlers simples | Fetch workflows | Connector install | 85% |
| Phase 3 | State machine | Uninstall complet | Lifecycle complet | 90% |
| Phase 4 | Handlers webhooks/PSU | Webhook flow | Webhook E2E | 85% |
| Phase 5 | - | Regression suite | Full regression | 90% |
| Phase 6 | - | - | Smoke tests prod | - |

### 7.8 CI/CD Pipeline

```yaml
# .github/workflows/test.yml

name: Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run unit tests
        run: go test -v -race -coverprofile=coverage.out ./...
      - name: Check coverage
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$COVERAGE < 85" | bc -l) )); then
            echo "Coverage $COVERAGE% is below 85%"
            exit 1
          fi

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v4
      - name: Run integration tests
        run: go test -v -tags=integration ./...
        env:
          DATABASE_URL: postgres://postgres:test@localhost:5432/postgres

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Start services
        run: docker-compose -f docker-compose.test.yml up -d
      - name: Run E2E tests
        run: go test -v -tags=e2e ./tests/e2e/...
      - name: Stop services
        run: docker-compose -f docker-compose.test.yml down
```

---

## 8. Risques et Mitigations

### 8.1 Risques Techniques

| Risque | Probabilité | Impact | Mitigation |
|--------|-------------|--------|------------|
| **Bugs state machine** | Haute | Élevé | Tests exhaustifs, feature flags, rollback rapide |
| **Perte de jobs en transit** | Moyenne | Élevé | Transactions DB, checkpoints fréquents |
| **Race conditions multi-instance** | Moyenne | Moyen | FOR UPDATE SKIP LOCKED, tests concurrence |
| **Performance dégradée** | Faible | Moyen | Benchmarks avant/après, monitoring |
| **Crash pendant uninstall** | Moyenne | Élevé | State machine avec resume, tests crash recovery |
| **Orphan data après migration** | Faible | Moyen | Scripts de vérification, cleanup jobs |

### 8.2 Risques Projet

| Risque | Probabilité | Impact | Mitigation |
|--------|-------------|--------|------------|
| **Dépassement délais** | Moyenne | Moyen | Buffer 20%, phases indépendantes |
| **Régression non détectée** | Faible | Élevé | Suite de tests exhaustive, feature flags |
| **Incompatibilité API** | Faible | Élevé | Tests de contrat API, versioning |

### 8.3 Plan de Rollback

#### Rollback Phase 1-5 (Avant suppression Temporal)

```bash
# Le feature flag permet un rollback instantané
export FF_USE_NEW_JOB_QUEUE=false
# Redéployer
kubectl rollout restart deployment/payments
```

#### Rollback Phase 6 (Après suppression Temporal)

```bash
# Revert vers la dernière version avec Temporal
git revert HEAD~N  # N = nombre de commits phase 5-6
# Redéployer Temporal infrastructure
helm install temporal temporal/temporal
# Redéployer Payments
kubectl apply -f k8s/payments-with-temporal.yaml
```

**Temps de rollback estimé** :
- Avec feature flag : < 5 minutes
- Sans feature flag : 30-60 minutes

---

## 9. Critères d'Acceptation

### 9.1 Critères Fonctionnels

- [ ] **Tous les providers existants fonctionnent** (17 providers)
- [ ] **Toutes les opérations API fonctionnent** identiquement
- [ ] **Polling des connecteurs fonctionne** avec les mêmes intervalles
- [ ] **Webhooks sont traités** correctement
- [ ] **Retry automatique** fonctionne avec exponential backoff
- [ ] **Idempotence** préservée pour toutes les opérations
- [ ] **Multi-instance** fonctionne sans duplicate processing

### 9.2 Critères de Performance

- [ ] **Throughput** : ≥ 500 jobs/sec (single instance, 20 workers)
- [ ] **Latence P99** : < 100ms pour enqueue
- [ ] **Latence P99** : < 5s pour job simple (CreateTransfer)
- [ ] **Scaling** : Linéaire avec le nombre d'instances
- [ ] **Recovery** : < 1 minute pour reprendre après crash

### 9.3 Critères de Qualité

- [ ] **Couverture tests** : ≥ 85% global, ≥ 90% pour jobqueue et state machine
- [ ] **Zéro régression** : Tous les tests E2E existants passent
- [ ] **Documentation** : README, API docs, runbook mis à jour
- [ ] **Pas de dette technique** : Code Temporal entièrement supprimé

### 9.4 Critères Opérationnels

- [ ] **Monitoring** : Métriques jobs (pending, processing, failed, completed)
- [ ] **Alerting** : Alertes sur queue depth, error rate, processing time
- [ ] **Logs** : Logs structurés pour debug
- [ ] **Graceful shutdown** : Jobs en cours terminés proprement

---

## 10. Estimation et Planning

### 10.1 Estimation par Phase

| Phase | Durée | Effort Dev | Effort Test | Total |
|-------|-------|------------|-------------|-------|
| Phase 0: Préparation | 1 sem | 3 j | 2 j | 1 sem |
| Phase 1: Infrastructure | 2-3 sem | 8 j | 4 j | 2.5 sem |
| Phase 2: Workflows simples | 2-3 sem | 8 j | 5 j | 2.5 sem |
| Phase 3: Workflows complexes | 3-4 sem | 12 j | 6 j | 3.5 sem |
| Phase 4: Webhooks/PSU | 2 sem | 6 j | 4 j | 2 sem |
| Phase 5: Cleanup | 1-2 sem | 4 j | 3 j | 1.5 sem |
| Phase 6: Déploiement | 1-2 sem | 2 j | 5 j | 1.5 sem |
| **Total** | **14-19 sem** | **43 j** | **29 j** | **~15 sem** |

### 10.2 Ressources Recommandées

- **2 développeurs backend** (full-time)
- **1 développeur QA** (50%)
- **1 DevOps** (20% - infra et déploiement)

### 10.3 Planning Suggéré

```
Semaine  1    2    3    4    5    6    7    8    9   10   11   12   13   14   15
        ├────┼────┼────┼────┼────┼────┼────┼────┼────┼────┼────┼────┼────┼────┤
Phase 0 │████│    │    │    │    │    │    │    │    │    │    │    │    │    │
Phase 1 │    │████│████│████│    │    │    │    │    │    │    │    │    │    │
Phase 2 │    │    │    │████│████│████│    │    │    │    │    │    │    │    │
Phase 3 │    │    │    │    │    │████│████│████│████│    │    │    │    │    │
Phase 4 │    │    │    │    │    │    │    │    │████│████│    │    │    │    │
Phase 5 │    │    │    │    │    │    │    │    │    │████│████│    │    │    │
Phase 6 │    │    │    │    │    │    │    │    │    │    │████│████│████│    │
Buffer  │    │    │    │    │    │    │    │    │    │    │    │    │████│████│
```

### 10.4 Jalons Clés

| Jalon | Date (semaine) | Critère de Succès |
|-------|----------------|-------------------|
| **M1: Infrastructure Ready** | S4 | JobQueue + Scheduler fonctionnels, tests passants |
| **M2: Simple Workflows** | S7 | 15 handlers migrés, feature flag activable |
| **M3: Complex Workflows** | S10 | Install/Uninstall fonctionnels, crash recovery testé |
| **M4: Feature Complete** | S12 | Tous handlers migrés, Temporal supprimable |
| **M5: Production Ready** | S15 | Déploiement production stable |

---

## Annexes

### A. Glossaire

| Terme | Définition |
|-------|------------|
| **Job** | Unité de travail asynchrone dans la queue |
| **Handler** | Fonction qui traite un type de job spécifique |
| **State Machine** | Pattern pour gérer les workflows multi-étapes |
| **SKIP LOCKED** | Clause PostgreSQL pour éviter les deadlocks |
| **Checkpoint** | Sauvegarde de l'état intermédiaire d'un workflow |
| **Orphan Job** | Job bloqué par une instance morte |

### B. Références

- [PostgreSQL Advisory Locks](https://www.postgresql.org/docs/current/explicit-locking.html)
- [FOR UPDATE SKIP LOCKED](https://www.postgresql.org/docs/current/sql-select.html#SQL-FOR-UPDATE-SHARE)
- [gocron - Go job scheduling](https://github.com/go-co-op/gocron)
- [Temporal Documentation](https://docs.temporal.io/) (référence actuelle)

### C. Liens Utiles

- Repository: `github.com/formancehq/payments`
- Issue Tracker: `github.com/formancehq/payments/issues`
- Documentation: `docs.formance.com/payments`

---

**Document maintenu par**: Équipe Payments
**Dernière mise à jour**: 2026-01-17
**Prochaine revue**: Avant début Phase 1
