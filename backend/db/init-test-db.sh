#!/bin/bash
set -e

# Run migrations in dependency order.
# 007_marketplace.sql references search_presets (from 008) and backtest_jobs
# (not yet defined), so we must run 008 before 007 and create the missing table.

MIGRATIONS_DIR="/migrations"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
  -- 001: core schema (users, pipelines, cases, etc.)
  \i ${MIGRATIONS_DIR}/001_initial.sql

  -- 005: portfolio tables
  \i ${MIGRATIONS_DIR}/005_portfolio.sql

  -- 008: search_presets (must come before 007)
  \i ${MIGRATIONS_DIR}/008_search_presets.sql

  -- stub: backtest_jobs referenced by 007 but not yet migrated
  CREATE TABLE IF NOT EXISTS backtest_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );

  -- 007: marketplace tables
  \i ${MIGRATIONS_DIR}/007_marketplace.sql

  -- 003: pipeline builder (agent blocks, monitor blocks, pipeline jobs)
  \i ${MIGRATIONS_DIR}/003_pipeline.sql

  -- 009: case enhancements
  \i ${MIGRATIONS_DIR}/009_case_enhancements.sql

  -- 009: watchlist
  \i ${MIGRATIONS_DIR}/009_watchlist.sql

  -- 010: monitoring engine (add case_id, last_executed_at, dsl_polling_enabled)
  \i ${MIGRATIONS_DIR}/010_monitoring.sql
EOSQL

echo "All migrations applied successfully."
