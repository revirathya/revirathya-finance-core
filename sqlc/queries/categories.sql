-- name: ListCategories :many
SELECT
  c.id,
  c.user_id,
  c.category_name
FROM
  app.categories AS c
WHERE
  c.user_id = sqlc.arg('user_id')
  AND (sqlc.narg('after_name')::TEXT IS NULL OR c.category_name > sqlc.narg('after_name')::TEXT)
ORDER BY c.category_name
LIMIT sqlc.arg('row_limit');

-- name: GetCategoryByID :one
SELECT
  c.id,
  c.user_id,
  c.category_name
FROM
  app.categories AS c
WHERE
  c.id = sqlc.arg('id')
  AND c.user_id = sqlc.arg('user_id');

-- name: CreateCategory :one
INSERT INTO app.categories (
  user_id,
  category_name
)
VALUES (
  sqlc.arg('user_id'),
  sqlc.arg('category_name')
)
RETURNING *;

-- name: UpdateCategory :one
UPDATE app.categories
SET
  category_name = COALESCE(sqlc.narg('category_name')::VARCHAR, category_name)
WHERE
  id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
RETURNING *;

-- name: DeleteCategory :execrows
DELETE FROM app.categories
WHERE
  id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id');
