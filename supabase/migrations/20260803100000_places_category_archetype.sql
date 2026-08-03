-- Add canonical venue category, archetype, and raw Google types for ranking gates.

ALTER TABLE public.places
  ADD COLUMN IF NOT EXISTS venue_category TEXT,
  ADD COLUMN IF NOT EXISTS venue_archetype TEXT,
  ADD COLUMN IF NOT EXISTS google_types TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE public.places
  DROP CONSTRAINT IF EXISTS places_venue_category_valid;

ALTER TABLE public.places
  ADD CONSTRAINT places_venue_category_valid CHECK (
    venue_category IS NULL
    OR venue_category IN ('restaurant', 'cafe', 'bar', 'club', 'other')
  );

ALTER TABLE public.places
  DROP CONSTRAINT IF EXISTS places_venue_archetype_valid;

ALTER TABLE public.places
  ADD CONSTRAINT places_venue_archetype_valid CHECK (
    venue_archetype IS NULL
    OR venue_archetype IN (
      'street_food', 'casual', 'midrange', 'fine_dining',
      'cocktail_bar', 'craft_beer', 'pub', 'rooftop',
      'specialty_coffee', 'study_cafe', 'chain',
      'nightclub', 'live_music', 'lounge'
    )
  );

CREATE INDEX IF NOT EXISTS places_venue_category_active_idx
  ON public.places (venue_category)
  WHERE is_active = true AND venue_category IS NOT NULL;

CREATE INDEX IF NOT EXISTS places_venue_archetype_active_idx
  ON public.places (venue_archetype)
  WHERE is_active = true AND venue_archetype IS NOT NULL;

-- Backfill curated places from purpose_categories junction.
UPDATE public.places p
SET venue_category = CASE pc.slug
  WHEN 'restaurants' THEN 'restaurant'
  WHEN 'cafes' THEN 'cafe'
  WHEN 'bars' THEN 'bar'
  WHEN 'entertainment' THEN 'club'
  ELSE 'other'
END
FROM public.place_categories plc
JOIN public.purpose_categories pc ON pc.id = plc.category_id
WHERE plc.place_id = p.id
  AND p.venue_category IS NULL;

-- Hand-set archetypes for known curated seeds.
UPDATE public.places
SET venue_archetype = CASE slug
  WHEN 'pizzas-4ps' THEN 'midrange'
  WHEN 'a-choens-grill' THEN 'casual'
  WHEN 'banh-mi-huynh-hoa' THEN 'street_food'
  WHEN 'the-workshop' THEN 'specialty_coffee'
  WHEN 'noir-bar' THEN 'cocktail_bar'
  ELSE venue_archetype
END
WHERE source = 'curated'
  AND venue_archetype IS NULL;

-- Infer category from humanized tags / name keywords when still null.
UPDATE public.places
SET venue_category = CASE
  WHEN EXISTS (
    SELECT 1 FROM unnest(tags) t
    WHERE lower(t) LIKE '%cafe%'
       OR lower(t) LIKE '%coffee%'
       OR lower(t) LIKE '%bakery%'
  ) THEN 'cafe'
  WHEN EXISTS (
    SELECT 1 FROM unnest(tags) t
    WHERE lower(t) LIKE '%bar%'
       OR lower(t) LIKE '%pub%'
       OR lower(t) LIKE '%cocktail%'
       OR lower(t) LIKE '%wine%'
  ) THEN 'bar'
  WHEN EXISTS (
    SELECT 1 FROM unnest(tags) t
    WHERE lower(t) LIKE '%club%'
       OR lower(t) LIKE '%night%'
       OR lower(t) LIKE '%dance%'
  ) THEN 'club'
  WHEN EXISTS (
    SELECT 1 FROM unnest(tags) t
    WHERE lower(t) LIKE '%restaurant%'
       OR lower(t) LIKE '%food%'
       OR lower(t) LIKE '%meal%'
  ) THEN 'restaurant'
  ELSE venue_category
END
WHERE venue_category IS NULL;

-- Infer archetype from price_level + category when still null.
UPDATE public.places
SET venue_archetype = CASE
  WHEN venue_category = 'restaurant' AND price_level = 1 THEN 'street_food'
  WHEN venue_category = 'restaurant' AND price_level = 2 THEN 'casual'
  WHEN venue_category = 'restaurant' AND price_level = 3 THEN 'midrange'
  WHEN venue_category = 'restaurant' AND price_level = 4 THEN 'fine_dining'
  WHEN venue_category = 'cafe' AND price_level <= 2 THEN 'specialty_coffee'
  WHEN venue_category = 'cafe' THEN 'chain'
  WHEN venue_category = 'bar' AND price_level >= 3 THEN 'cocktail_bar'
  WHEN venue_category = 'bar' AND price_level = 1 THEN 'pub'
  WHEN venue_category = 'bar' THEN 'craft_beer'
  WHEN venue_category = 'club' THEN 'nightclub'
  ELSE venue_archetype
END
WHERE venue_archetype IS NULL
  AND venue_category IS NOT NULL;
