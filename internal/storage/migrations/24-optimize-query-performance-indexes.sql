-- Migration 24: Optimize Query Performance Indexes
-- This migration adds comprehensive indexes to improve query performance across the application.
-- All indexes use CREATE INDEX CONCURRENTLY for zero-downtime deployment.

-- ============================================
-- JSONB GIN Indexes
-- ============================================
-- Dramatically improve metadata filtering performance (100x+ faster)

CREATE INDEX CONCURRENTLY idx_payments_metadata_gin
    ON payments USING gin (metadata);

CREATE INDEX CONCURRENTLY idx_accounts_metadata_gin
    ON accounts USING gin (metadata);

CREATE INDEX CONCURRENTLY idx_payment_adjustments_metadata_gin
    ON payment_adjustments USING gin (metadata);

CREATE INDEX CONCURRENTLY idx_payment_initiations_metadata_gin
    ON payment_initiations USING gin (metadata);

CREATE INDEX CONCURRENTLY idx_payment_initiation_adjustments_metadata_gin
    ON payment_initiation_adjustments USING gin (metadata);

CREATE INDEX CONCURRENTLY idx_payment_initiation_reversals_metadata_gin
    ON payment_initiation_reversals USING gin (metadata);

CREATE INDEX CONCURRENTLY idx_bank_accounts_metadata_gin
    ON bank_accounts USING gin (metadata);

-- ============================================
-- Payment Adjustment Optimizations
-- ============================================
-- Composite index for LATERAL JOIN in PaymentsList (payments.go:432-439)
-- Covers: WHERE payment_id = X ORDER BY created_at DESC, sort_id DESC LIMIT 1

CREATE INDEX CONCURRENTLY idx_payment_adjustments_payment_created_sort
    ON payment_adjustments (payment_id, created_at DESC, sort_id DESC);

-- ============================================
-- Reference Lookup Optimizations
-- ============================================
-- Composite index for PaymentsGetByReference (payments.go:254-264)

CREATE INDEX CONCURRENTLY idx_payments_connector_reference
    ON payments (connector_id, reference);

-- ============================================
-- Balance Query Optimizations
-- ============================================
-- Composite index for balance time-range queries (balances.go:173-191)

CREATE INDEX CONCURRENTLY idx_balances_account_asset_time_range
    ON balances (account_id, asset, last_updated_at, created_at);

-- ============================================
-- Connector Deletion Optimizations
-- ============================================
-- Composite indexes for CASCADE DELETE performance when deleting connectors (connectors.go:169-175)

CREATE INDEX CONCURRENTLY idx_payments_connector_created_sort
    ON payments (connector_id, created_at DESC, sort_id DESC);

CREATE INDEX CONCURRENTLY idx_accounts_connector_created_sort
    ON accounts (connector_id, created_at DESC, sort_id DESC);

CREATE INDEX CONCURRENTLY idx_balances_connector_account_asset
    ON balances (connector_id, account_id, asset);

CREATE INDEX CONCURRENTLY idx_payment_initiations_connector_created_sort
    ON payment_initiations (connector_id, created_at DESC, sort_id DESC);

CREATE INDEX CONCURRENTLY idx_payment_initiation_adjustments_pi_created_sort
    ON payment_initiation_adjustments (payment_initiation_id, created_at DESC, sort_id DESC);

-- ============================================
-- Partial Indexes
-- ============================================
-- Partial index for active (non-deleted) connectors

CREATE INDEX CONCURRENTLY idx_connectors_active_id_name
    ON connectors (id, name, created_at)
    WHERE scheduled_for_deletion = false;

-- ============================================
-- Payment Initiation Related Optimizations
-- ============================================
-- Covering index for payment initiation related payments lookup

CREATE INDEX CONCURRENTLY idx_payment_initiation_related_payments_both_ids
    ON payment_initiation_related_payments (payment_initiation_id, payment_id, created_at DESC);

-- ============================================
-- Additional Composite Indexes
-- ============================================
-- Composite index for pool account lookups

CREATE INDEX CONCURRENTLY idx_pool_accounts_pool_account_connector
    ON pool_accounts (pool_id, account_id, connector_id);

-- Composite index for workflow instances by connector and schedule

CREATE INDEX CONCURRENTLY idx_workflows_instances_connector_schedule
    ON workflows_instances (connector_id, schedule_id, created_at DESC);

-- Composite index for tasks by connector and status

CREATE INDEX CONCURRENTLY idx_tasks_connector_status_created
    ON tasks (connector_id, status, created_at DESC)
    WHERE connector_id IS NOT NULL;

-- ============================================
-- PSU and Open Banking Indexes
-- ============================================
-- Composite indexes for PSU and open banking connection queries

CREATE INDEX CONCURRENTLY idx_payments_psu_connector_obc
    ON payments (psu_id, connector_id, open_banking_connection_id)
    WHERE psu_id IS NOT NULL;

CREATE INDEX CONCURRENTLY idx_accounts_psu_connector_obc
    ON accounts (psu_id, connector_id, open_banking_connection_id)
    WHERE psu_id IS NOT NULL;

CREATE INDEX CONCURRENTLY idx_balances_psu_connector_obc
    ON balances (psu_id, connector_id, open_banking_connection_id)
    WHERE psu_id IS NOT NULL;
