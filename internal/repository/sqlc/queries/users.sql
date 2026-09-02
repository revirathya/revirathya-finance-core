-- name: GetUserByID :one
SELECT
  usr.id,
  usr.first_name,
  usr.last_name,
  utel.telegram_chat_id
FROM
  app.users AS usr
INNER JOIN
  app.user_telegram_contexts AS utel
  ON
    usr.id = utel.user_id
WHERE
  usr.id = sqlc.arg('id');
