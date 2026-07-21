-- name: GetPlaceByID :one
SELECT
  p.id::text AS id,
  p.city_id::text AS city_id,
  c.slug AS city_slug,
  c.name AS city_name,
  p.google_place_id,
  p.source,
  p.name,
  p.slug,
  p.neighborhood,
  p.address,
  p.latitude,
  p.longitude,
  p.price_level,
  p.rating_cached,
  p.user_rating_count,
  p.phone,
  p.website,
  p.hours_json,
  p.photo_names,
  p.business_status,
  p.cover_image_url,
  p.tags,
  p.last_synced_at,
  p.is_active
FROM places p
JOIN cities c ON c.id = p.city_id
WHERE p.id = sqlc.arg(id)::uuid
  AND p.is_active = true;

-- name: GetPlaceByGooglePlaceID :one
SELECT
  p.id::text AS id,
  p.city_id::text AS city_id,
  c.slug AS city_slug,
  c.name AS city_name,
  p.google_place_id,
  p.source,
  p.name,
  p.slug,
  p.neighborhood,
  p.address,
  p.latitude,
  p.longitude,
  p.price_level,
  p.rating_cached,
  p.user_rating_count,
  p.phone,
  p.website,
  p.hours_json,
  p.photo_names,
  p.business_status,
  p.cover_image_url,
  p.tags,
  p.last_synced_at,
  p.is_active
FROM places p
JOIN cities c ON c.id = p.city_id
WHERE p.google_place_id = sqlc.arg(google_place_id)
  AND p.is_active = true;

-- name: UpsertPlaceFromGoogle :one
INSERT INTO places (
  city_id,
  google_place_id,
  source,
  name,
  neighborhood,
  address,
  latitude,
  longitude,
  price_level,
  rating_cached,
  user_rating_count,
  phone,
  website,
  hours_json,
  photo_names,
  business_status,
  cover_image_url,
  tags,
  last_synced_at,
  is_active
)
VALUES (
  sqlc.arg(city_id)::uuid,
  sqlc.arg(google_place_id),
  'google',
  sqlc.arg(name),
  sqlc.narg(neighborhood),
  sqlc.narg(address),
  sqlc.narg(latitude),
  sqlc.narg(longitude),
  sqlc.narg(price_level),
  sqlc.narg(rating_cached),
  sqlc.narg(user_rating_count),
  sqlc.narg(phone),
  sqlc.narg(website),
  sqlc.narg(hours_json),
  COALESCE(sqlc.narg(photo_names), '{}'::text[]),
  sqlc.narg(business_status),
  sqlc.narg(cover_image_url),
  COALESCE(sqlc.arg(tags), '{}'::text[]),
  now(),
  true
)
ON CONFLICT (google_place_id) DO UPDATE
SET
  name = EXCLUDED.name,
  neighborhood = COALESCE(EXCLUDED.neighborhood, places.neighborhood),
  address = COALESCE(EXCLUDED.address, places.address),
  latitude = COALESCE(EXCLUDED.latitude, places.latitude),
  longitude = COALESCE(EXCLUDED.longitude, places.longitude),
  price_level = COALESCE(EXCLUDED.price_level, places.price_level),
  rating_cached = COALESCE(EXCLUDED.rating_cached, places.rating_cached),
  user_rating_count = COALESCE(EXCLUDED.user_rating_count, places.user_rating_count),
  phone = COALESCE(EXCLUDED.phone, places.phone),
  website = COALESCE(EXCLUDED.website, places.website),
  hours_json = COALESCE(EXCLUDED.hours_json, places.hours_json),
  photo_names = CASE
    WHEN cardinality(EXCLUDED.photo_names) > 0 THEN EXCLUDED.photo_names
    ELSE places.photo_names
  END,
  business_status = COALESCE(EXCLUDED.business_status, places.business_status),
  cover_image_url = COALESCE(EXCLUDED.cover_image_url, places.cover_image_url),
  tags = CASE
    WHEN cardinality(EXCLUDED.tags) > 0 THEN EXCLUDED.tags
    ELSE places.tags
  END,
  last_synced_at = now(),
  updated_at = now()
RETURNING id::text AS id;
