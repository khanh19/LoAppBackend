BEGIN;

SET search_path = public;

ALTER TABLE public.place_lists
ADD COLUMN IF NOT EXISTS creator_user_id UUID REFERENCES public.users(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS subtitle TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS list_type TEXT NOT NULL DEFAULT 'curated',
ADD COLUMN IF NOT EXISTS visibility TEXT NOT NULL DEFAULT 'public',
ADD COLUMN IF NOT EXISTS saves_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE public.place_lists
DROP CONSTRAINT IF EXISTS place_lists_list_type_valid;

ALTER TABLE public.place_lists
ADD CONSTRAINT place_lists_list_type_valid CHECK (
    list_type IN ('curated', 'user_plan')
);

ALTER TABLE public.place_lists
DROP CONSTRAINT IF EXISTS place_lists_visibility_valid;

ALTER TABLE public.place_lists
ADD CONSTRAINT place_lists_visibility_valid CHECK (
    visibility IN ('public', 'private')
);

ALTER TABLE public.place_lists
DROP CONSTRAINT IF EXISTS place_lists_saves_count_valid;

ALTER TABLE public.place_lists
ADD CONSTRAINT place_lists_saves_count_valid CHECK (saves_count >= 0);

CREATE INDEX IF NOT EXISTS place_lists_creator_idx
    ON public.place_lists (creator_user_id)
    WHERE creator_user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS place_lists_type_active_sort_idx
    ON public.place_lists (list_type, is_active, sort_order);

ALTER TABLE public.place_list_entries
ADD COLUMN IF NOT EXISTS time_label TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS activity_type TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS image_url TEXT;

COMMIT;
