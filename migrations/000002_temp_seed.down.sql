-- Remove: User Telegram Contexts
DELETE FROM app.user_telegram_contexts
WHERE telegram_chat_id = '675439471';

-- Remove: Users
DELETE FROM app.users
WHERE email = 'av.leonardif@gmail.com';

-- Remove: Categories
DELETE FROM app.categories
WHERE category_name IN (
  'Food - Daily',
  'Food - Snack & Coffee',
  'Groceries'
);
