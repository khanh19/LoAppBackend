-- name: FindUserByIdentity :one
SELECT
  u.id::text AS id,
  u.primary_email,
  u.email_verified_at,
  u.onboarding_status::text AS onboarding_status,
  u.last_login_at,
  u.created_at,
  u.updated_at
FROM user_auth_identities i
JOIN users u ON u.id = i.user_id
WHERE i.provider = sqlc.arg(provider)::auth_provider
  AND i.provider_user_id = sqlc.arg(provider_user_id)
  AND u.deleted_at IS NULL;

-- name: FindUserByEmail :one
SELECT
  id::text AS id,
  primary_email,
  email_verified_at,
  onboarding_status::text AS onboarding_status,
  last_login_at,
  created_at,
  updated_at
FROM users
WHERE primary_email = sqlc.arg(primary_email)
  AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (primary_email, email_verified_at)
VALUES (sqlc.arg(primary_email), CASE WHEN sqlc.arg(email_verified)::boolean THEN now() ELSE NULL END)
RETURNING
  id::text AS id,
  primary_email,
  email_verified_at,
  onboarding_status::text AS onboarding_status,
  last_login_at,
  created_at,
  updated_at;

-- name: UpsertIdentity :exec
INSERT INTO user_auth_identities (
  user_id, provider, provider_user_id, email, email_verified, raw_profile, linked_at, last_used_at
)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.arg(provider)::auth_provider,
  sqlc.arg(provider_user_id),
  sqlc.arg(email),
  sqlc.arg(email_verified),
  sqlc.arg(raw_profile)::jsonb,
  now(),
  now()
)
ON CONFLICT (provider, provider_user_id)
DO UPDATE SET
  user_id = EXCLUDED.user_id,
  email = EXCLUDED.email,
  email_verified = EXCLUDED.email_verified,
  raw_profile = EXCLUDED.raw_profile,
  last_used_at = now();

-- name: UpdateLastLogin :one
UPDATE users
SET
  last_login_at = now(),
  email_verified_at = CASE
    WHEN sqlc.arg(email_verified)::boolean AND email_verified_at IS NULL THEN now()
    ELSE email_verified_at
  END,
  primary_email = COALESCE(primary_email, sqlc.arg(primary_email))
WHERE id = sqlc.arg(user_id)::uuid
RETURNING
  id::text AS id,
  primary_email,
  email_verified_at,
  onboarding_status::text AS onboarding_status,
  last_login_at,
  created_at,
  updated_at;

-- name: UpsertUserProfile :one
INSERT INTO user_profiles (
  user_id,
  first_name,
  last_name,
  username,
  phone_e164,
  phone_country_code,
  phone_national_number,
  date_of_birth,
  avatar_object_key,
  avatar_url,
  bio
)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.arg(first_name),
  sqlc.arg(last_name),
  sqlc.arg(username)::citext,
  sqlc.arg(phone_e164),
  sqlc.arg(phone_country_code),
  sqlc.arg(phone_national_number),
  sqlc.arg(date_of_birth)::date,
  sqlc.arg(avatar_object_key),
  sqlc.arg(avatar_url),
  sqlc.arg(bio)
)
ON CONFLICT (user_id) DO UPDATE SET
  first_name = EXCLUDED.first_name,
  last_name = EXCLUDED.last_name,
  username = EXCLUDED.username,
  phone_e164 = EXCLUDED.phone_e164,
  phone_country_code = EXCLUDED.phone_country_code,
  phone_national_number = EXCLUDED.phone_national_number,
  date_of_birth = EXCLUDED.date_of_birth,
  avatar_object_key = EXCLUDED.avatar_object_key,
  avatar_url = EXCLUDED.avatar_url,
  bio = EXCLUDED.bio,
  updated_at = now()
RETURNING
  user_id::text AS user_id,
  first_name,
  last_name,
  username::text AS username,
  phone_e164,
  phone_country_code,
  phone_national_number,
  date_of_birth,
  avatar_object_key,
  avatar_url,
  bio,
  created_at,
  updated_at;

-- name: GetUserProfile :one
SELECT
  user_id::text AS user_id,
  first_name,
  last_name,
  username::text AS username,
  phone_e164,
  phone_country_code,
  phone_national_number,
  date_of_birth,
  avatar_object_key,
  avatar_url,
  bio,
  created_at,
  updated_at
FROM user_profiles
WHERE user_id = sqlc.arg(user_id)::uuid;

-- name: IsUsernameTakenByAnotherUser :one
SELECT EXISTS (
  SELECT 1
  FROM user_profiles
  WHERE username = sqlc.arg(username)::citext
    AND user_id <> sqlc.arg(current_user_id)::uuid
) AS taken;
