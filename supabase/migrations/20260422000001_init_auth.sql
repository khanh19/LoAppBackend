BEGIN;

SET search_path = public;

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    CREATE TYPE public.onboarding_status AS ENUM ('not_started', 'in_progress', 'completed');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

DO $$
BEGIN
    CREATE TYPE public.auth_provider AS ENUM ('email', 'google', 'apple', 'facebook', 'auth0');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

CREATE TABLE IF NOT EXISTS public.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    primary_email CITEXT UNIQUE,
    email_verified_at TIMESTAMPTZ,
    password_hash TEXT,
    onboarding_status public.onboarding_status NOT NULL DEFAULT 'not_started',
    onboarding_completed_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT users_email_or_identity_ready CHECK (
        primary_email IS NOT NULL OR password_hash IS NULL
    )
);

CREATE TABLE IF NOT EXISTS public.user_auth_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    provider public.auth_provider NOT NULL,
    provider_user_id TEXT NOT NULL,
    email CITEXT,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    raw_profile JSONB NOT NULL DEFAULT '{}'::jsonb,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    UNIQUE (provider, provider_user_id),
    UNIQUE (user_id, provider, provider_user_id)
);

CREATE INDEX IF NOT EXISTS users_primary_email_idx
    ON public.users (primary_email)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS user_auth_identities_user_idx
    ON public.user_auth_identities (user_id);

CREATE OR REPLACE FUNCTION public.set_updated_at() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS users_set_updated_at ON public.users;
CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON public.users
FOR EACH ROW
EXECUTE FUNCTION public.set_updated_at();

COMMIT;
