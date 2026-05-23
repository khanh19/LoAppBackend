BEGIN;

SET search_path = public;

-- Future: user_place_rankings (Beli head-to-head) and user_place_lists (want-to-try / been-to).

CREATE TABLE IF NOT EXISTS public.cities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    country_code CHAR(2) NOT NULL,
    latitude NUMERIC(9, 6),
    longitude NUMERIC(9, 6),
    cover_image_url TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS cities_active_sort_idx
    ON public.cities (is_active, sort_order);

CREATE TABLE IF NOT EXISTS public.purpose_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    icon_name TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS purpose_categories_active_sort_idx
    ON public.purpose_categories (is_active, sort_order);

CREATE TABLE IF NOT EXISTS public.vibes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS vibes_active_sort_idx
    ON public.vibes (is_active, sort_order);

CREATE TABLE IF NOT EXISTS public.places (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    city_id UUID NOT NULL REFERENCES public.cities(id) ON DELETE RESTRICT,
    google_place_id TEXT UNIQUE,
    source TEXT NOT NULL,
    name TEXT NOT NULL,
    slug TEXT,
    neighborhood TEXT,
    address TEXT,
    latitude NUMERIC(9, 6),
    longitude NUMERIC(9, 6),
    price_level SMALLINT,
    rating_cached NUMERIC(2, 1),
    cover_image_url TEXT,
    tags TEXT[] NOT NULL DEFAULT '{}',
    last_synced_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT places_source_valid CHECK (source IN ('curated', 'google')),
    CONSTRAINT places_google_id_required CHECK (
        source = 'curated' OR google_place_id IS NOT NULL
    ),
    CONSTRAINT places_price_level_valid CHECK (
        price_level IS NULL OR price_level BETWEEN 1 AND 4
    ),
    UNIQUE (city_id, slug)
);

CREATE INDEX IF NOT EXISTS places_city_active_idx
    ON public.places (city_id, is_active);

CREATE INDEX IF NOT EXISTS places_source_idx
    ON public.places (source)
    WHERE is_active = true;

CREATE TABLE IF NOT EXISTS public.place_categories (
    place_id UUID NOT NULL REFERENCES public.places(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES public.purpose_categories(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (place_id, category_id)
);

CREATE TABLE IF NOT EXISTS public.place_vibes (
    place_id UUID NOT NULL REFERENCES public.places(id) ON DELETE CASCADE,
    vibe_id UUID NOT NULL REFERENCES public.vibes(id) ON DELETE RESTRICT,
    strength NUMERIC(4, 3) NOT NULL DEFAULT 1.000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (place_id, vibe_id),
    CONSTRAINT place_vibes_strength_valid CHECK (strength > 0 AND strength <= 1)
);

CREATE TABLE IF NOT EXISTS public.user_explore_cities (
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    city_id UUID NOT NULL REFERENCES public.cities(id) ON DELETE RESTRICT,
    priority INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, city_id),
    CONSTRAINT user_explore_cities_priority_valid CHECK (priority > 0)
);

CREATE TABLE IF NOT EXISTS public.user_purpose_categories (
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES public.purpose_categories(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, category_id)
);

CREATE TABLE IF NOT EXISTS public.user_vibes (
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    vibe_id UUID NOT NULL REFERENCES public.vibes(id) ON DELETE RESTRICT,
    weight NUMERIC(4, 3) NOT NULL DEFAULT 1.000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, vibe_id),
    CONSTRAINT user_vibes_weight_valid CHECK (weight > 0 AND weight <= 1)
);

CREATE TABLE IF NOT EXISTS public.user_liked_places (
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    place_id UUID NOT NULL REFERENCES public.places(id) ON DELETE CASCADE,
    source TEXT NOT NULL DEFAULT 'onboarding',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, place_id)
);

CREATE TABLE IF NOT EXISTS public.onboarding_steps (
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    step_number SMALLINT NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    PRIMARY KEY (user_id, step_number),
    CONSTRAINT onboarding_steps_number_valid CHECK (step_number BETWEEN 1 AND 4)
);

DROP TRIGGER IF EXISTS places_set_updated_at ON public.places;
CREATE TRIGGER places_set_updated_at
BEFORE UPDATE ON public.places
FOR EACH ROW
EXECUTE FUNCTION public.set_updated_at();

COMMIT;
