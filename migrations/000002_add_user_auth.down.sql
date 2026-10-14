-- Remove col:google_sub from users
ALTER TABLE app.users DROP CONSTRAINT users_google_sub_key;
ALTER TABLE app.users DROP COLUMN google_sub;

-- Remove users cons:email_key UNIQUE
ALTER TABLE app.users DROP CONSTRAINT users_email_key;