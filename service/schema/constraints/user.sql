ALTER TABLE "user".session ADD CONSTRAINT session_refresh_hash CHECK (refresh_tkn_hash ~ '^[0-9a-f]{64}$');
