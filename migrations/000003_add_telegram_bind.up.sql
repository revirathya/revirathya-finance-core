-- Add table:telegram_bind_codes
CREATE TABLE app.telegram_bind_codes (
  id UUID DEFAULT GEN_RANDOM_UUID(),
  user_id UUID NOT NULL UNIQUE,
  code_hash VARCHAR(64) NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (id),
  FOREIGN KEY (user_id) REFERENCES app.users (id) ON DELETE CASCADE
);

-- Add trigger:set_updated_at on telegram_bind_codes
CREATE TRIGGER set_updated_at
BEFORE UPDATE ON app.telegram_bind_codes
FOR EACH ROW
EXECUTE FUNCTION app.update_ts_now();