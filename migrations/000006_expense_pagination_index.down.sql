-- Drop index:user_dt_id on expenses
DROP INDEX app.expenses_user_dt_id;

-- Add index:transaction_dt on expenses
CREATE INDEX user_transaction_dt
ON app.expenses (
  user_id,
  transaction_dt DESC
);
