-- name: CreateExpense :one
INSERT INTO app.expenses (
  user_id,
  category_id,
  transaction_dt,
  description,
  amount
)
VALUES (
  sqlc.arg('user_id'),
  sqlc.arg('category_id'),
  sqlc.arg('transaction_dt'),
  COALESCE(sqlc.narg('description')::TEXT, ''),
  sqlc.arg('amount')
)
RETURNING *;

-- name: GetExpenseByID :one
SELECT
  exp.id,
  exp.user_id,
  exp.category_id,
  ctg.category_name,
  exp.transaction_dt,
  exp.description,
  exp.amount
FROM
  app.expenses AS exp
INNER JOIN app.categories AS ctg
  ON exp.category_id = ctg.id
WHERE
  exp.id = sqlc.arg('id')
  AND exp.user_id = sqlc.arg('user_id');

-- name: ListExpenses :many
SELECT
  exp.id,
  exp.user_id,
  exp.category_id,
  ctg.category_name,
  exp.transaction_dt,
  exp.description,
  exp.amount
FROM
  app.expenses AS exp
INNER JOIN app.categories AS ctg
  ON exp.category_id = ctg.id
WHERE
  exp.user_id = sqlc.arg('user_id')
  AND (sqlc.narg('transaction_dt')::DATE IS NULL OR exp.transaction_dt = sqlc.narg('transaction_dt')::DATE)
ORDER BY exp.transaction_dt DESC;

-- name: UpdateExpense :one
UPDATE app.expenses
SET
  category_id = COALESCE(sqlc.arg('category_id'), category_id),
  transaction_dt = COALESCE(sqlc.arg('transaction_dt'), transaction_dt),
  description = COALESCE(sqlc.narg('description'), description),
  amount = COALESCE(sqlc.narg('amount'), amount)
WHERE
  id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
RETURNING *;

-- name: DeleteExpense :execrows
DELETE FROM app.expenses
WHERE
  id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id');
