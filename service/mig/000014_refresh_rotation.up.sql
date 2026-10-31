-- Existing sessions deliberately require login: plaintext refresh tokens are retired.
ALTER TABLE "user".session RENAME COLUMN refresh_tkn TO refresh_tkn_hash;
UPDATE "user".session SET blocked=true, refresh_tkn_hash=repeat('0',64);
ALTER TABLE "user".session ADD CONSTRAINT session_refresh_hash CHECK (refresh_tkn_hash ~ '^[0-9a-f]{64}$');
