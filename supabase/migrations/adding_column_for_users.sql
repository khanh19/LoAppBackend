ALTER TABLE users
ADD COLUMN first_name TEXT,
ADD COLUMN last_name TEXT,
ADD COLUMN username CITEXT UNIQUE,
ADD COLUMN phone_country_code TEXT,
ADD COLUMN phone_number TEXT,
ADD COLUMN date_of_birth DATE,
ADD COLUMN profile_picture_url TEXT;

ALTER TABLE users
ADD CONSTRAINT users_username_length_check
CHECK (
    username IS NULL 
    OR length(username::text) BETWEEN 3 AND 20
);

ALTER TABLE users
ADD CONSTRAINT users_username_format_check
CHECK (
    username IS NULL
    OR username::text ~ '^[a-zA-Z0-9_\.]+$'
);

ALTER TABLE users
ADD CONSTRAINT users_phone_country_code_check
CHECK (
    phone_country_code IS NULL
    OR phone_country_code ~ '^\+[0-9]{1,4}$'
);

ALTER TABLE users
ADD CONSTRAINT users_dob_not_future_check
CHECK (
    date_of_birth IS NULL
    OR date_of_birth <= CURRENT_DATE
);


/** Dropping above columns and adding new table */
ALTER TABLE users
DROP CONSTRAINT IF EXISTS users_username_length_check,
DROP CONSTRAINT IF EXISTS users_username_format_check,
DROP CONSTRAINT IF EXISTS users_phone_country_code_check,
DROP CONSTRAINT IF EXISTS users_dob_not_future_check,
DROP CONSTRAINT IF EXISTS users_username_key;

ALTER TABLE users
DROP COLUMN IF EXISTS first_name,
DROP COLUMN IF EXISTS last_name,
DROP COLUMN IF EXISTS username,
DROP COLUMN IF EXISTS phone_country_code,
DROP COLUMN IF EXISTS phone_number,
DROP COLUMN IF EXISTS date_of_birth,
DROP COLUMN IF EXISTS profile_picture_url;