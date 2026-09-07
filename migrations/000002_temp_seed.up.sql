-- Seed: Categories
INSERT INTO app.categories (
  category_name
)
VALUES
('Food - Daily'),
('Food - Snack & Coffee'),
('Groceries');

-- Seed: Users
INSERT INTO app.users (
  first_name,
  last_name,
  email
)
VALUES
('Leonardi', 'Fabianto', 'av.leonardif@gmail.com');

-- Seed: User Telegram Contexts
INSERT INTO app.user_telegram_contexts (
  user_id,
  telegram_chat_id,
  conversation,
  state
)
SELECT
  id AS user_id,
  '675439471' AS telegram_chat_id,
  'root' AS conversation,
  'init' AS state
FROM app.users;
