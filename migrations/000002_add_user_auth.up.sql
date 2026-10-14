-- Add col:google_sub to users
ALTER TABLE app.users ADD COLUMN google_sub VARCHAR(255);
ALTER TABLE app.users ADD CONSTRAINT users_google_sub_key UNIQUE (google_sub);

-- Add users cons:email_key to UNIQUE
ALTER TABLE app.users ADD CONSTRAINT users_email_key UNIQUE (email);