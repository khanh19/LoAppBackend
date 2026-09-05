-- Google Photo Media photoUri values are short-lived and must not be persisted.
-- Keep places.photo_names as a refreshable reference and places.google_place_id
-- as the durable identifier used to obtain new references.
UPDATE public.places
SET cover_image_url = NULL
WHERE cover_image_url ILIKE '%googleusercontent.com%';

UPDATE public.place_list_entries
SET image_url = NULL
WHERE image_url ILIKE '%googleusercontent.com%';

COMMENT ON COLUMN public.places.photo_names IS
  'Refreshable Google Places photo resource names. A name may expire; refresh it from Place Details using google_place_id.';

COMMENT ON COLUMN public.places.cover_image_url IS
  'Non-Google persistent image URL only. Google Photo Media photoUri values must not be stored here.';
