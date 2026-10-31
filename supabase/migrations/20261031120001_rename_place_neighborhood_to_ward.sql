-- Vietnam's 2025 administrative reform removed the district level, so place
-- labels move from district to ward.

ALTER TABLE public.places
    RENAME COLUMN neighborhood TO ward;

-- Existing values were parsed from administrative_area_level_2, which is the
-- old district. Clear them so they get refreshed from Google lazily.
UPDATE public.places
SET ward       = NULL,
    updated_at = now()
WHERE ward IS NOT NULL
  AND (
    ward ILIKE '%district%'
    OR ward ILIKE 'quận%'
    OR ward ILIKE 'quan %'
    OR ward ILIKE 'huyện%'
    OR ward ILIKE 'huyen %'
    OR ward ILIKE 'thị xã%'
    OR ward ILIKE 'thi xa%'
  );
