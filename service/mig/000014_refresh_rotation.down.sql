-- Hashes cannot be converted back into credentials; downgrade also requires login.
ALTER TABLE "user".session DROP CONSTRAINT session_refresh_hash;
UPDATE "user".session SET blocked=true, refresh_tkn_hash='';
ALTER TABLE "user".session RENAME COLUMN refresh_tkn_hash TO refresh_tkn;
