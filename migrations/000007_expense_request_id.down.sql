-- Remove col:request_id from expenses
DROP INDEX app.expenses_user_request_id;

ALTER TABLE app.expenses DROP COLUMN request_id;
