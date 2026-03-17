-- Migration 004: add resource_group column for SAP AI Core support.
ALTER TABLE audit_events
    ADD COLUMN IF NOT EXISTS resource_group LowCardinality(String) DEFAULT '';
