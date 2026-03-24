-- 007_add_token_usage.sql
-- Adds token usage columns extracted from LLM API responses.
-- input_tokens:       prompt/input token count
-- output_tokens:      completion/output token count
-- cache_read_tokens:  Anthropic cache_read_input_tokens
-- cache_write_tokens: Anthropic cache_creation_input_tokens

ALTER TABLE audit.audit_events ADD COLUMN IF NOT EXISTS input_tokens UInt32 DEFAULT 0;
ALTER TABLE audit.audit_events ADD COLUMN IF NOT EXISTS output_tokens UInt32 DEFAULT 0;
ALTER TABLE audit.audit_events ADD COLUMN IF NOT EXISTS cache_read_tokens UInt32 DEFAULT 0;
ALTER TABLE audit.audit_events ADD COLUMN IF NOT EXISTS cache_write_tokens UInt32 DEFAULT 0;
