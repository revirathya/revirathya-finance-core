-- Schema (New)
CREATE SCHEMA app;

-- Tables (New)
-- a. users: Users information
-- b. user_telegram_contexts: Users context/state in Telegram conversation
-- c. categories: Expenses categories
-- d. expenses: Records of expenses
CREATE TABLE app.users (
  id UUID DEFAULT GEN_RANDOM_UUID(),
  first_name VARCHAR(100) NOT NULL,
  last_name VARCHAR(100),
  email VARCHAR(100) NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (id)
);

CREATE TABLE app.user_telegram_contexts (
  id UUID DEFAULT GEN_RANDOM_UUID(),
  user_id UUID UNIQUE NOT NULL,
  telegram_chat_id VARCHAR(100) UNIQUE,
  conversation VARCHAR(100),
  state VARCHAR(100),
  interaction_chat_id VARCHAR(100),
  interaction_message_id VARCHAR(100),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (id),
  FOREIGN KEY (user_id) REFERENCES app.users (id) ON DELETE CASCADE
);

CREATE INDEX interactions ON app.user_telegram_contexts (interaction_chat_id, interaction_message_id); -- noqa: PG01

CREATE TABLE app.categories (
  id UUID DEFAULT GEN_RANDOM_UUID(),
  category_name VARCHAR(100) NOT NULL UNIQUE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (id)
);

CREATE TABLE app.expenses (
  id UUID DEFAULT GEN_RANDOM_UUID(),
  user_id UUID NOT NULL,
  category_id UUID NOT NULL,
  transaction_dt DATE NOT NULL,
  description TEXT,
  amount NUMERIC(14, 2) NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (id),
  FOREIGN KEY (user_id) REFERENCES app.users (id) ON DELETE CASCADE,
  FOREIGN KEY (category_id) REFERENCES app.categories (id) ON DELETE RESTRICT
);

CREATE INDEX user_transaction_dt ON app.expenses (user_id, transaction_dt DESC); -- noqa: PG01
CREATE INDEX category_id ON app.expenses (category_id); -- noqa: PG01

-- Function (New)
-- a. UPDATE_TS_NOW: update updated_at column with NOW()
CREATE FUNCTION app.update_ts_now()
RETURNS TRIGGER AS $$
  BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
  END;
$$ LANGUAGE plpgsql;

-- Trigger (New)
-- a. set_updated_at on users
-- b. set_updated_at on user_telegram_contexts
-- c. set_updated_at on categories
-- d. set_updated_at on expenses

CREATE TRIGGER set_updated_at
BEFORE UPDATE ON app.users
FOR EACH ROW
EXECUTE FUNCTION app.update_ts_now();

CREATE TRIGGER set_updated_at
BEFORE UPDATE ON app.user_telegram_contexts
FOR EACH ROW
EXECUTE FUNCTION app.update_ts_now();

CREATE TRIGGER set_updated_at
BEFORE UPDATE ON app.categories
FOR EACH ROW
EXECUTE FUNCTION app.update_ts_now();

CREATE TRIGGER set_updated_at
BEFORE UPDATE ON app.expenses
FOR EACH ROW
EXECUTE FUNCTION app.update_ts_now();
