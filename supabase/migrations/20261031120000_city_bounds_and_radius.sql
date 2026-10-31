-- City geography: center, bounding box and the Google search radius used for
-- autocomplete bias. Drives dynamic city selection instead of a hard-coded HCMC.

ALTER TABLE public.cities
    ADD COLUMN IF NOT EXISTS sw_lat NUMERIC(9, 6),
    ADD COLUMN IF NOT EXISTS sw_lng NUMERIC(9, 6),
    ADD COLUMN IF NOT EXISTS ne_lat NUMERIC(9, 6),
    ADD COLUMN IF NOT EXISTS ne_lng NUMERIC(9, 6),
    ADD COLUMN IF NOT EXISTS search_radius_m INTEGER NOT NULL DEFAULT 15000;

ALTER TABLE public.cities
    DROP CONSTRAINT IF EXISTS cities_bounds_valid;

ALTER TABLE public.cities
    ADD CONSTRAINT cities_bounds_valid CHECK (
        (sw_lat IS NULL AND sw_lng IS NULL AND ne_lat IS NULL AND ne_lng IS NULL)
        OR (sw_lat < ne_lat AND sw_lng < ne_lng)
    );

UPDATE public.cities
SET latitude        = 21.028500,
    longitude       = 105.854200,
    sw_lat          = 20.830000,
    sw_lng          = 105.600000,
    ne_lat          = 21.230000,
    ne_lng          = 106.060000,
    search_radius_m = 25000,
    is_active       = true
WHERE slug = 'hanoi';

UPDATE public.cities
SET latitude        = 16.054400,
    longitude       = 108.202200,
    sw_lat          = 15.930000,
    sw_lng          = 108.080000,
    ne_lat          = 16.180000,
    ne_lng          = 108.350000,
    search_radius_m = 15000,
    is_active       = true
WHERE slug = 'danang';

UPDATE public.cities
SET latitude        = 10.776900,
    longitude       = 106.700900,
    sw_lat          = 10.570000,
    sw_lng          = 106.480000,
    ne_lat          = 10.980000,
    ne_lng          = 106.900000,
    search_radius_m = 25000,
    is_active       = true
WHERE slug = 'hcmc';

-- Cities without a bounding box can never be matched by coordinates, so keep
-- them out of the active set until they are configured.
UPDATE public.cities
SET is_active = false
WHERE sw_lat IS NULL;
