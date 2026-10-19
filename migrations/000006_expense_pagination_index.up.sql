-- Drop index:transaction_dt on expense
DROP INDEX app.user_transaction_dt;

-- Add index:user_dt_id on expense
CREATE INDEX expenses_user_dt_id
ON app.expenses (
  user_id,
  transaction_dt DESC,
  id DESC
);
