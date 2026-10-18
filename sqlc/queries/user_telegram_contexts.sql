-- name: GetUserTelegramContext :one
SELECT
  utel.id,
  utel.user_id,
  utel.telegram_chat_id,
  utel.conversation,
  utel.state,
  utel.interaction_chat_id,
  utel.interaction_message_id
FROM
  app.user_telegram_contexts AS utel
WHERE
  utel.user_id = sqlc.arg('user_id');

-- name: GetUserTelegramContextByTelegramID :one
SELECT
  utel.id,
  utel.user_id,
  utel.telegram_chat_id,
  utel.conversation,
  utel.state,
  utel.interaction_chat_id,
  utel.interaction_message_id
FROM
  app.user_telegram_contexts AS utel
WHERE
  utel.telegram_chat_id = sqlc.arg('telegram_chat_id');

-- name: UpdateUserTelegramContext :one
UPDATE app.user_telegram_contexts
SET
  conversation = COALESCE(sqlc.narg('conversation'), conversation),
  state = COALESCE(sqlc.narg('state'), state),
  interaction_chat_id = COALESCE(sqlc.narg('interaction_chat_id'), interaction_chat_id),
  interaction_message_id = COALESCE(sqlc.narg('interaction_message_id'), interaction_message_id)
WHERE
  user_id = sqlc.arg('user_id')
RETURNING *;

-- name: UpsertTelegramBinding :one
INSERT INTO app.user_telegram_contexts (
  user_id,
  telegram_chat_id
)
VALUES (
  sqlc.arg('user_id'),
  sqlc.arg('telegram_chat_id')
)
ON CONFLICT (user_id)
DO UPDATE SET
  telegram_chat_id = sqlc.arg('telegram_chat_id'),
  conversation = NULL,
  state = NULL,
  interaction_chat_id = NULL,
  interaction_message_id = NULL
RETURNING user_id;

-- name: UnbindTelegramBinding :exec
UPDATE app.user_telegram_contexts
SET
  telegram_chat_id = NULL,
  conversation = NULL,
  state = NULL,
  interaction_chat_id = NULL,
  interaction_message_id = NULL
WHERE
  user_id = sqlc.arg('user_id');
