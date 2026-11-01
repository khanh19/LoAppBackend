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
  p.ward,
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
  p.venue_category,
  p.venue_archetype,
  p.google_types,
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
  p.ward,
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
  p.venue_category,
  p.venue_archetype,
  p.google_types,
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
  ward,
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
  google_types,
  venue_category,
  venue_archetype,
  last_synced_at,
  is_active
)
VALUES (
  sqlc.arg(city_id)::uuid,
  sqlc.arg(google_place_id),
  'google',
  sqlc.arg(name),
  sqlc.narg(ward),
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
  COALESCE(sqlc.arg(google_types), '{}'::text[]),
  sqlc.narg(venue_category),
  sqlc.narg(venue_archetype),
  now(),
  true
)
ON CONFLICT (google_place_id) DO UPDATE
SET
  name = EXCLUDED.name,
  ward = COALESCE(EXCLUDED.ward, places.ward),
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
  google_types = CASE
    WHEN cardinality(EXCLUDED.google_types) > 0 THEN EXCLUDED.google_types
    ELSE places.google_types
  END,
  venue_category = COALESCE(EXCLUDED.venue_category, places.venue_category),
  venue_archetype = COALESCE(EXCLUDED.venue_archetype, places.venue_archetype),
  last_synced_at = now(),
  updated_at = now()
RETURNING id::text AS id;

-- name: ListNearbyPlaces :many
-- Bounding-box prefilter, then haversine distance, then random order for the swipe deck.
SELECT
  p.id::text AS id,
  p.google_place_id,
  p.name,
  p.ward,
  p.address,
  p.latitude,
  p.longitude,
  p.price_level,
  p.cover_image_url,
  p.venue_category,
  c.slug AS city_slug,
  c.name AS city_name
FROM places p
JOIN cities c ON c.id = p.city_id
WHERE p.is_active = true
  AND p.latitude IS NOT NULL
  AND p.longitude IS NOT NULL
  AND p.latitude::float8 BETWEEN sqlc.arg(sw_lat)::float8 AND sqlc.arg(ne_lat)::float8
  AND p.longitude::float8 BETWEEN sqlc.arg(sw_lng)::float8 AND sqlc.arg(ne_lng)::float8
  AND (
    sqlc.narg(category)::text IS NULL
    OR (
      sqlc.narg(category)::text = 'pub'
      AND (
        'pub' = ANY (COALESCE(p.google_types, '{}'::text[]))
        OR p.venue_archetype = 'pub'
      )
    )
    OR (
      sqlc.narg(category)::text = 'bar'
      AND p.venue_category = 'bar'
      AND NOT ('pub' = ANY (COALESCE(p.google_types, '{}'::text[])))
      AND COALESCE(p.venue_archetype, '') <> 'pub'
    )
    OR (
      sqlc.narg(category)::text IS NOT NULL
      AND sqlc.narg(category)::text NOT IN ('pub', 'bar')
      AND p.venue_category = sqlc.narg(category)::text
    )
  )
  AND (
    cardinality(sqlc.arg(exclude_ids)::uuid[]) = 0
    OR NOT (p.id = ANY (sqlc.arg(exclude_ids)::uuid[]))
  )
  AND (
    6371000 * acos(
      least(
        1.0,
        greatest(
          -1.0,
          cos(radians(sqlc.arg(lat)::float8)) * cos(radians(p.latitude::float8)) *
          cos(radians(p.longitude::float8) - radians(sqlc.arg(lng)::float8)) +
          sin(radians(sqlc.arg(lat)::float8)) * sin(radians(p.latitude::float8))
        )
      )
    )
  ) <= sqlc.arg(radius_m)::float8
ORDER BY random()
LIMIT sqlc.arg(limit_count);

-- name: CountBookmarksForPlaces :many
SELECT
  place_id::text AS place_id,
  COUNT(*)::integer AS bookmark_count
FROM place_bookmarks
WHERE place_id = ANY (sqlc.arg(place_ids)::uuid[])
GROUP BY place_id;

-- name: ListTopCompanionsForPlaces :many
SELECT
  vta.place_id::text AS place_id,
  vta.tag_slug,
  COALESCE(tt.label, vta.tag_slug) AS label,
  vta.percentage,
  vta.raw_count
FROM venue_tag_aggregates vta
LEFT JOIN tag_taxonomy tt
  ON tt.slug = vta.tag_slug
 AND tt.tag_type = 'companion'
WHERE vta.place_id = ANY (sqlc.arg(place_ids)::uuid[])
  AND vta.tag_type = 'companion'
  AND vta.raw_count > 0
ORDER BY vta.place_id, vta.percentage DESC, vta.tag_slug;
