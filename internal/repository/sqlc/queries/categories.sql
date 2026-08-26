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
