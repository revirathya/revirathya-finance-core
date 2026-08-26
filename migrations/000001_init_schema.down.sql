-- Trigger
DROP TRIGGER set_updated_at ON app.expenses CASCADE;
DROP TRIGGER set_updated_at ON app.categories CASCADE;
DROP TRIGGER set_updated_at ON app.user_telegram_contexts CASCADE;
DROP TRIGGER set_updated_at ON app.users CASCADE;

-- Function
DROP FUNCTION app.update_ts_now;

-- Tables
DROP TABLE app.expenses;
DROP TABLE app.categories;
DROP TABLE app.user_telegram_contexts;
DROP TABLE app.users;

-- Schema
DROP SCHEMA app;
