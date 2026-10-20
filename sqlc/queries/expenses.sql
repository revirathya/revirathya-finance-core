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
  AND (
    sqlc.narg('after_dt')::DATE IS NULL
    OR (exp.transaction_dt, exp.id) < (sqlc.narg('after_dt')::DATE, sqlc.narg('after_id')::UUID)
  )
ORDER BY
  exp.transaction_dt DESC,
  exp.id DESC
LIMIT sqlc.arg('row_limit');

-- name: UpdateExpense :one
UPDATE app.expenses
SET
  category_id = COALESCE(sqlc.narg('category_id')::UUID, category_id),
  transaction_dt = COALESCE(sqlc.narg('transaction_dt')::DATE, transaction_dt),
  description = COALESCE(sqlc.narg('description')::TEXT, description),
  amount = COALESCE(sqlc.narg('amount')::NUMERIC(14,2), amount)
WHERE
  id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
RETURNING *;

-- name: DeleteExpense :execrows
DELETE FROM app.expenses
WHERE
  id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id');
