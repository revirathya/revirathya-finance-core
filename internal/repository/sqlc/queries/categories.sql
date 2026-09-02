-- name: ListCategories :many
SELECT
  c.id,
  c.category_name
FROM
  app.categories AS c
ORDER BY c.category_name;

-- name: GetCategoryByID :one
SELECT
  c.id,
  c.category_name
FROM
  app.categories AS c
WHERE
  c.id = sqlc.arg('id');

-- name: CreateCategory :one
INSERT INTO app.categories (
  category_name
)
VALUES (
  sqlc.arg('category_name')
)
RETURNING *;

-- name: UpdateCategory :one
UPDATE app.categories
SET
  category_name = COALESCE(sqlc.arg('category_name'), category_name)
WHERE
  id = sqlc.arg('id')
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM app.categories
WHERE
  id = sqlc.arg('id');
