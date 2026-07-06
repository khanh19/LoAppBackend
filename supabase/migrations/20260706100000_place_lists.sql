BEGIN;

SET search_path = public;

CREATE TABLE IF NOT EXISTS public.place_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT '',
    area TEXT NOT NULL DEFAULT '',
    occasions TEXT[] NOT NULL DEFAULT '{}',
    city_id UUID NOT NULL REFERENCES public.cities(id) ON DELETE RESTRICT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    source_hash TEXT,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS place_lists_active_sort_idx
    ON public.place_lists (is_active, sort_order);

CREATE TABLE IF NOT EXISTS public.place_list_entries (
    list_id UUID NOT NULL REFERENCES public.place_lists(id) ON DELETE CASCADE,
    place_id UUID NOT NULL REFERENCES public.places(id) ON DELETE RESTRICT,
    seed_name TEXT NOT NULL,
    rank INTEGER NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (list_id, seed_name),
    CONSTRAINT place_list_entries_rank_valid CHECK (rank > 0),
    UNIQUE (list_id, place_id)
);

CREATE INDEX IF NOT EXISTS place_list_entries_list_active_rank_idx
    ON public.place_list_entries (list_id, is_active, rank);

CREATE TABLE IF NOT EXISTS public.list_sync_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'running',
    entries_added INTEGER NOT NULL DEFAULT 0,
    entries_skipped INTEGER NOT NULL DEFAULT 0,
    entries_failed INTEGER NOT NULL DEFAULT 0,
    details JSONB NOT NULL DEFAULT '[]'::jsonb,
    CONSTRAINT list_sync_runs_status_valid CHECK (
        status IN ('running', 'completed', 'failed')
    )
);

CREATE INDEX IF NOT EXISTS list_sync_runs_started_at_idx
    ON public.list_sync_runs (started_at DESC);

DROP TRIGGER IF EXISTS place_lists_set_updated_at ON public.place_lists;
CREATE TRIGGER place_lists_set_updated_at
BEFORE UPDATE ON public.place_lists
FOR EACH ROW
EXECUTE FUNCTION public.set_updated_at();

DROP TRIGGER IF EXISTS place_list_entries_set_updated_at ON public.place_list_entries;
CREATE TRIGGER place_list_entries_set_updated_at
BEFORE UPDATE ON public.place_list_entries
FOR EACH ROW
EXECUTE FUNCTION public.set_updated_at();

COMMIT;
