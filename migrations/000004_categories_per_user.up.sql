-- Add col:user_id to categories
ALTER TABLE app.categories ADD COLUMN user_id UUID NOT NULL;
ALTER TABLE app.categories ADD CONSTRAINT categories_user_id_fkey FOREIGN KEY (user_id) REFERENCES app.users(id) ON DELETE CASCADE;

-- Remove cons:category_name_key to categories
ALTER TABLE app.categories DROP CONSTRAINT categories_category_name_key;

-- Add cons:unique_user_category,unique_id_user to categories
ALTER TABLE app.categories ADD CONSTRAINT categories_unique_user_category UNIQUE (user_id, category_name);
ALTER TABLE app.categories ADD CONSTRAINT categories_unique_id_user UNIQUE (id, user_id);

-- Alter expenses FK to categories: id -> (id, user_id)
ALTER TABLE app.expenses DROP CONSTRAINT expenses_category_id_fkey;
ALTER TABLE app.expenses ADD CONSTRAINT expenses_categories_fkey FOREIGN KEY (category_id, user_id) REFERENCES app.categories(id, user_id) ON DELETE RESTRICT;
