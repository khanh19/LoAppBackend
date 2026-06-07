BEGIN;

SET search_path = public;

-- ---------------------------------------------------------------------------
-- Event sources (scrape providers)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS public.event_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    base_url TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO public.event_sources (slug, name, base_url)
VALUES (
    'vietnamnightlife',
    'Vietnam Nightlife',
    'https://vietnamnightlife.com'
)
ON CONFLICT (slug) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Scraped external events (replaced wholesale per import)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS public.external_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES public.event_sources(id) ON DELETE CASCADE,
    city_id UUID REFERENCES public.cities(id) ON DELETE SET NULL,
    place_id UUID REFERENCES public.places(id) ON DELETE SET NULL,

    event_name TEXT NOT NULL,
    event_url TEXT NOT NULL,
    venue_name TEXT,
    venue_url TEXT,
    schedule_text TEXT,
    music_text TEXT,
    description TEXT,
    poster_image_url TEXT,

    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT external_events_source_event_url_unique UNIQUE (source_id, event_url)
);

CREATE INDEX IF NOT EXISTS external_events_city_imported_idx
    ON public.external_events (city_id, imported_at DESC);

CREATE INDEX IF NOT EXISTS external_events_source_idx
    ON public.external_events (source_id);

CREATE INDEX IF NOT EXISTS external_events_venue_name_idx
    ON public.external_events (venue_name)
    WHERE venue_name IS NOT NULL;

DROP TRIGGER IF EXISTS external_events_set_updated_at ON public.external_events;
CREATE TRIGGER external_events_set_updated_at
BEFORE UPDATE ON public.external_events
FOR EACH ROW
EXECUTE FUNCTION public.set_updated_at();

-- ---------------------------------------------------------------------------
-- RLS: service-role / direct Postgres imports only for now
-- ---------------------------------------------------------------------------

ALTER TABLE public.event_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.external_events ENABLE ROW LEVEL SECURITY;

-- ---------------------------------------------------------------------------
-- Stored procedure: replace all events for a source in one transaction
-- ---------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION public.replace_external_events(
    p_source_slug TEXT,
    p_events JSONB
)
RETURNS TABLE (
    source_slug TEXT,
    deleted_count BIGINT,
    inserted_count BIGINT,
    imported_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_source_id UUID;
    v_imported_at TIMESTAMPTZ := now();
    v_deleted BIGINT := 0;
    v_inserted BIGINT := 0;
BEGIN
    IF p_events IS NULL OR jsonb_typeof(p_events) <> 'array' THEN
        RAISE EXCEPTION 'p_events must be a JSON array';
    END IF;

    SELECT id INTO v_source_id
    FROM public.event_sources
    WHERE slug = p_source_slug AND is_active = true;

    IF v_source_id IS NULL THEN
        RAISE EXCEPTION 'Unknown or inactive event source: %', p_source_slug;
    END IF;

    DELETE FROM public.external_events
    WHERE source_id = v_source_id;

    GET DIAGNOSTICS v_deleted = ROW_COUNT;

    INSERT INTO public.external_events (
        source_id,
        city_id,
        place_id,
        event_name,
        event_url,
        venue_name,
        venue_url,
        schedule_text,
        music_text,
        description,
        poster_image_url,
        imported_at
    )
    SELECT
        v_source_id,
        c.id,
        NULL::uuid,
        NULLIF(trim(e->>'event_name'), ''),
        NULLIF(trim(e->>'event_url'), ''),
        NULLIF(trim(e->>'venue_name'), ''),
        NULLIF(trim(e->>'venue_url'), ''),
        NULLIF(trim(e->>'schedule'), ''),
        NULLIF(trim(e->>'music'), ''),
        NULLIF(trim(e->>'description'), ''),
        NULLIF(trim(e->>'image_url'), ''),
        v_imported_at
    FROM jsonb_array_elements(p_events) AS e
    LEFT JOIN public.cities c
        ON c.slug = NULLIF(trim(e->>'city'), '')
        AND c.is_active = true
    WHERE NULLIF(trim(e->>'event_name'), '') IS NOT NULL
      AND NULLIF(trim(e->>'event_url'), '') IS NOT NULL;

    GET DIAGNOSTICS v_inserted = ROW_COUNT;

    RETURN QUERY
    SELECT p_source_slug, v_deleted, v_inserted, v_imported_at;
END;
$$;

REVOKE ALL ON FUNCTION public.replace_external_events(TEXT, JSONB) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.replace_external_events(TEXT, JSONB) TO postgres;
GRANT EXECUTE ON FUNCTION public.replace_external_events(TEXT, JSONB) TO service_role;

COMMIT;
