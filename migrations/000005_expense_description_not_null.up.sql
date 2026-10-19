-- Fill NULL col:description on expenses to empty string
UPDATE app.expenses
SET
  description = ''
WHERE
  description IS NULL;

-- Update col:description on expenses to not null
ALTER TABLE app.expenses
  ALTER COLUMN description SET DEFAULT '',
  ALTER COLUMN description SET NOT NULL;
