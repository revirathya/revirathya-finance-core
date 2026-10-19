-- Drop nullable from col:description on expenses
ALTER TABLE app.expenses
  ALTER COLUMN description DROP NOT NULL,
  ALTER COLUMN description DROP DEFAULT;

-- Fill empty string col:description on expenses to NULL
UPDATE app.expenses
SET
  description = NULL
WHERE
  description = '';
