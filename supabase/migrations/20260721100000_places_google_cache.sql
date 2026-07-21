-- Extended Google Places cache fields on places
ALTER TABLE public.places
  ADD COLUMN IF NOT EXISTS user_rating_count INTEGER,
  ADD COLUMN IF NOT EXISTS phone TEXT,
  ADD COLUMN IF NOT EXISTS website TEXT,
  ADD COLUMN IF NOT EXISTS hours_json JSONB,
  ADD COLUMN IF NOT EXISTS photo_names TEXT[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS business_status TEXT;

CREATE INDEX IF NOT EXISTS places_google_place_id_active_idx
  ON public.places (google_place_id)
  WHERE is_active = true AND google_place_id IS NOT NULL;
