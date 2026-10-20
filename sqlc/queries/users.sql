-- name: GetUserByID :one
SELECT
  usr.id,
  usr.first_name,
  usr.last_name,
  usr.email,
  utel.telegram_chat_id
FROM
  app.users AS usr
LEFT JOIN
  app.user_telegram_contexts AS utel
  ON
    usr.id = utel.user_id
WHERE
  usr.id = sqlc.arg('id');

-- name: GetUserByGoogleSub :one
SELECT
  usr.id,
  usr.first_name,
  usr.last_name,
  usr.email,
  utel.telegram_chat_id
FROM
  app.users AS usr
LEFT JOIN
  app.user_telegram_contexts AS utel
  ON
    usr.id = utel.user_id
WHERE
  usr.google_sub = sqlc.arg('google_sub');

-- name: CreateUser :one
INSERT INTO app.users (
  first_name,
  last_name,
  email,
  google_sub
)
VALUES (
  sqlc.arg('first_name'),
  sqlc.narg('last_name'),
  sqlc.arg('email'),
  sqlc.arg('google_sub')
)
RETURNING id;

-- name: UpdateUser :one
UPDATE app.users
SET
  first_name = COALESCE(sqlc.narg('first_name')::VARCHAR, first_name),
  last_name = (CASE
    WHEN sqlc.narg('last_name')::VARCHAR IS NULL THEN last_name
    ELSE NULLIF(sqlc.narg('last_name')::VARCHAR, '')
  END)
WHERE
  id = sqlc.arg('id')
RETURNING id;