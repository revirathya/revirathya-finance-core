-- Alter expenses FK to categories: (id, user_id) -> id
ALTER TABLE app.expenses ADD CONSTRAINT expenses_category_id_fkey FOREIGN KEY (category_id) REFERENCES app.categories(id) ON DELETE RESTRICT;
ALTER TABLE app.expenses DROP CONSTRAINT expenses_categories_fkey;

-- Remove cons:unique_user_category,unique_id_user to categories
ALTER TABLE app.categories DROP CONSTRAINT categories_unique_user_category;
ALTER TABLE app.categories DROP CONSTRAINT categories_unique_id_user;

-- Add cons:category_name_key to categories
ALTER TABLE app.categories Add CONSTRAINT categories_category_name_key UNIQUE (category_name);

-- Remove col:user_id to categories
ALTER TABLE app.categories DROP CONSTRAINT categories_user_id_fkey;
ALTER TABLE app.categories DROP COLUMN user_id;

