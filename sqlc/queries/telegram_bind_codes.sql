-- name: DeleteBindCodes :exec
DELETE FROM app.telegram_bind_codes
WHERE
  user_id = sqlc.arg('user_id');

-- name: CreateBindCode :exec
INSERT INTO app.telegram_bind_codes (
  user_id,
  code_hash,
  expires_at
)
VALUES (
  sqlc.arg('user_id'),
  sqlc.arg('code_hash'),
  sqlc.arg('expires_at')
);

-- name: ConsumeBindCode :one
UPDATE app.telegram_bind_codes
SET
  used_at = NOW()
WHERE
  code_hash = sqlc.arg('code_hash')
  AND used_at IS NULL
  AND expires_at > NOW()
RETURNING user_id;
