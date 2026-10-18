BEGIN;

SET search_path = public;

-- User-owned place playlists (Spotify-style collections on the List tab).
CREATE TABLE IF NOT EXISTS public.user_place_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_place_lists_title_nonempty CHECK (btrim(title) <> '')
);

CREATE INDEX IF NOT EXISTS user_place_lists_owner_updated_idx
    ON public.user_place_lists (owner_user_id, updated_at DESC);

DROP TRIGGER IF EXISTS user_place_lists_set_updated_at ON public.user_place_lists;
CREATE TRIGGER user_place_lists_set_updated_at
BEFORE UPDATE ON public.user_place_lists
FOR EACH ROW
EXECUTE FUNCTION public.set_updated_at();

CREATE TABLE IF NOT EXISTS public.user_place_list_items (
    list_id UUID NOT NULL REFERENCES public.user_place_lists(id) ON DELETE CASCADE,
    place_id UUID NOT NULL REFERENCES public.places(id) ON DELETE CASCADE,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (list_id, place_id),
    CONSTRAINT user_place_list_items_position_valid CHECK (position >= 0)
);

CREATE INDEX IF NOT EXISTS user_place_list_items_list_position_idx
    ON public.user_place_list_items (list_id, position);

CREATE INDEX IF NOT EXISTS user_place_list_items_place_idx
    ON public.user_place_list_items (place_id);

-- Individual place bookmarks (feeds the system "Bookmarked" collection).
CREATE TABLE IF NOT EXISTS public.place_bookmarks (
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    place_id UUID NOT NULL REFERENCES public.places(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, place_id)
);

CREATE INDEX IF NOT EXISTS place_bookmarks_user_created_idx
    ON public.place_bookmarks (user_id, created_at DESC);

-- Plan bookmarks (saved user_plan place_lists).
CREATE TABLE IF NOT EXISTS public.plan_bookmarks (
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    list_id UUID NOT NULL REFERENCES public.place_lists(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, list_id)
);

CREATE INDEX IF NOT EXISTS plan_bookmarks_user_created_idx
    ON public.plan_bookmarks (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS plan_bookmarks_list_idx
    ON public.plan_bookmarks (list_id);

COMMIT;
