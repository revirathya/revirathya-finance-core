-- Add col:request_id in expenses
ALTER TABLE app.expenses ADD COLUMN request_id UUID;

CREATE UNIQUE INDEX expenses_user_request_id
ON app.expenses (user_id, request_id)
WHERE request_id IS NOT NULL;
