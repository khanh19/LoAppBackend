-- Feed post time. Editing a plan republishes it without changing created_at,
-- and without treating a bookmark's updated_at bump as a new post.

ALTER TABLE public.place_lists
    ADD COLUMN posted_at TIMESTAMPTZ;

UPDATE public.place_lists
SET posted_at = created_at
WHERE posted_at IS NULL;

ALTER TABLE public.place_lists
    ALTER COLUMN posted_at SET DEFAULT now(),
    ALTER COLUMN posted_at SET NOT NULL;
