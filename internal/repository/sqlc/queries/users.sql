-- name: GetUserByID :one
SELECT
  usr.id,
  usr.first_name,
  usr.last_name
FROM
  app.users AS usr
WHERE
  usr.id = sqlc.arg('id');
