-- name: ListActivePlaceLists :many
SELECT
  pl.id::text AS id,
  pl.slug,
  pl.title,
  pl.category,
  pl.area,
  pl.occasions,
  pl.sort_order,
  c.slug AS city_slug,
  c.name AS city_name,
  (
    SELECT COUNT(*)::integer
    FROM place_list_entries ple
    WHERE ple.list_id = pl.id
      AND ple.is_active = true
  ) AS entry_count,
  cover.cover_image_url
FROM place_lists pl
JOIN cities c ON c.id = pl.city_id
LEFT JOIN LATERAL (
  SELECT p.cover_image_url
  FROM place_list_entries ple
  JOIN places p ON p.id = ple.place_id
  WHERE ple.list_id = pl.id
    AND ple.is_active = true
    AND p.cover_image_url IS NOT NULL
  ORDER BY ple.rank
  LIMIT 1
) cover ON true
WHERE pl.is_active = true
ORDER BY pl.sort_order, pl.title;

-- name: GetPlaceListBySlug :one
SELECT
  pl.id::text AS id,
  pl.slug,
  pl.title,
  pl.subtitle,
  pl.category,
  pl.area,
  pl.occasions,
  pl.sort_order,
  pl.source_hash,
  pl.list_type,
  pl.visibility,
  pl.saves_count,
  pl.city_id::text AS city_id,
  c.slug AS city_slug,
  c.name AS city_name,
  COALESCE(pl.creator_user_id::text, ''::text)::text AS creator_user_id,
  COALESCE(NULLIF(trim(concat_ws(' ', up.first_name, up.last_name)), ''), up.username::text, u.primary_email) AS creator_display_name,
  COALESCE(up.username::text, ''::text)::text AS creator_username,
  up.avatar_url AS creator_avatar_url,
  (
    SELECT COUNT(*)::integer
    FROM place_list_entries ple
    WHERE ple.list_id = pl.id
      AND ple.is_active = true
  ) AS stops_count
FROM place_lists pl
JOIN cities c ON c.id = pl.city_id
LEFT JOIN users u ON u.id = pl.creator_user_id
LEFT JOIN user_profiles up ON up.user_id = pl.creator_user_id
WHERE pl.slug = sqlc.arg(slug)
  AND pl.is_active = true;

-- name: GetPlaceListByID :one
SELECT
  pl.id::text AS id,
  pl.slug,
  pl.title,
  pl.subtitle,
  pl.category,
  pl.area,
  pl.occasions,
  pl.sort_order,
  pl.source_hash,
  pl.list_type,
  pl.visibility,
  pl.saves_count,
  pl.city_id::text AS city_id,
  c.slug AS city_slug,
  c.name AS city_name,
  COALESCE(pl.creator_user_id::text, ''::text)::text AS creator_user_id,
  COALESCE(NULLIF(trim(concat_ws(' ', up.first_name, up.last_name)), ''), up.username::text, u.primary_email) AS creator_display_name,
  COALESCE(up.username::text, ''::text)::text AS creator_username,
  up.avatar_url AS creator_avatar_url,
  (
    SELECT COUNT(*)::integer
    FROM place_list_entries ple
    WHERE ple.list_id = pl.id
      AND ple.is_active = true
  ) AS stops_count
FROM place_lists pl
JOIN cities c ON c.id = pl.city_id
LEFT JOIN users u ON u.id = pl.creator_user_id
LEFT JOIN user_profiles up ON up.user_id = pl.creator_user_id
WHERE pl.id = sqlc.arg(id)::uuid;

-- name: ListPlaceListEntriesByListID :many
SELECT
  ple.seed_name,
  ple.rank,
  ple.note,
  ple.time_label,
  ple.activity_type,
  COALESCE(ple.image_url, p.cover_image_url) AS image_url,
  p.id::text AS place_id,
  p.name AS place_name,
  p.google_place_id,
  p.address,
  p.neighborhood,
  p.latitude,
  p.longitude,
  p.rating_cached,
  p.price_level,
  p.cover_image_url,
  p.tags
FROM place_list_entries ple
JOIN places p ON p.id = ple.place_id
WHERE ple.list_id = sqlc.arg(list_id)::uuid
  AND ple.is_active = true
ORDER BY ple.rank;

-- name: ListPlaceListEntrySeedNamesByListID :many
SELECT seed_name
FROM place_list_entries
WHERE list_id = sqlc.arg(list_id)::uuid
  AND is_active = true;

-- name: UpsertPlaceList :one
INSERT INTO place_lists (
  slug,
  title,
  category,
  area,
  occasions,
  city_id,
  sort_order,
  source_hash,
  last_synced_at
)
VALUES (
  sqlc.arg(slug),
  sqlc.arg(title),
  sqlc.arg(category),
  sqlc.arg(area),
  sqlc.arg(occasions),
  sqlc.arg(city_id)::uuid,
  sqlc.arg(sort_order)::integer,
  sqlc.arg(source_hash),
  now()
)
ON CONFLICT (slug) DO UPDATE
SET
  title = EXCLUDED.title,
  category = EXCLUDED.category,
  area = EXCLUDED.area,
  occasions = EXCLUDED.occasions,
  city_id = EXCLUDED.city_id,
  sort_order = EXCLUDED.sort_order,
  source_hash = EXCLUDED.source_hash,
  last_synced_at = now(),
  updated_at = now()
RETURNING id::text AS id;

-- name: CreatePlan :one
INSERT INTO place_lists (
  slug,
  title,
  subtitle,
  category,
  area,
  occasions,
  city_id,
  creator_user_id,
  list_type,
  visibility,
  saves_count,
  sort_order,
  is_active
)
VALUES (
  sqlc.arg(slug),
  sqlc.arg(title),
  sqlc.arg(subtitle),
  sqlc.arg(category),
  sqlc.arg(area),
  sqlc.arg(occasions),
  sqlc.arg(city_id)::uuid,
  sqlc.arg(creator_user_id)::uuid,
  'user_plan',
  sqlc.arg(visibility),
  sqlc.arg(saves_count)::integer,
  sqlc.arg(sort_order)::integer,
  true
)
RETURNING id::text AS id;

-- name: CreatePlanPlace :one
INSERT INTO places (
  city_id,
  google_place_id,
  source,
  name,
  neighborhood,
  address,
  latitude,
  longitude,
  cover_image_url,
  tags,
  is_active
)
VALUES (
  sqlc.arg(city_id)::uuid,
  sqlc.narg(google_place_id),
  CASE WHEN sqlc.narg(google_place_id)::text IS NULL THEN 'curated' ELSE 'google' END,
  sqlc.arg(name),
  sqlc.narg(neighborhood),
  sqlc.narg(address),
  sqlc.narg(latitude),
  sqlc.narg(longitude),
  sqlc.narg(cover_image_url),
  sqlc.arg(tags),
  true
)
ON CONFLICT (google_place_id) DO UPDATE
SET
  name = EXCLUDED.name,
  neighborhood = COALESCE(EXCLUDED.neighborhood, places.neighborhood),
  address = COALESCE(EXCLUDED.address, places.address),
  latitude = COALESCE(EXCLUDED.latitude, places.latitude),
  longitude = COALESCE(EXCLUDED.longitude, places.longitude),
  cover_image_url = COALESCE(EXCLUDED.cover_image_url, places.cover_image_url),
  tags = CASE
    WHEN cardinality(EXCLUDED.tags) > 0 THEN EXCLUDED.tags
    ELSE places.tags
  END,
  updated_at = now()
RETURNING id::text AS id;

-- name: CreatePlanEntry :exec
INSERT INTO place_list_entries (
  list_id,
  place_id,
  seed_name,
  rank,
  note,
  time_label,
  activity_type,
  image_url,
  is_active
)
VALUES (
  sqlc.arg(list_id)::uuid,
  sqlc.arg(place_id)::uuid,
  sqlc.arg(seed_name),
  sqlc.arg(rank)::integer,
  sqlc.arg(note),
  sqlc.arg(time_label),
  sqlc.arg(activity_type),
  sqlc.narg(image_url),
  true
);

-- name: UpsertPlaceListEntry :exec
INSERT INTO place_list_entries (
  list_id,
  place_id,
  seed_name,
  rank,
  note,
  is_active
)
VALUES (
  sqlc.arg(list_id)::uuid,
  sqlc.arg(place_id)::uuid,
  sqlc.arg(seed_name),
  sqlc.arg(rank)::integer,
  sqlc.arg(note),
  true
)
ON CONFLICT (list_id, seed_name) DO UPDATE
SET
  place_id = EXCLUDED.place_id,
  rank = EXCLUDED.rank,
  note = EXCLUDED.note,
  is_active = true,
  updated_at = now();

-- name: DeactivatePlaceListEntriesNotInSeedNames :execrows
UPDATE place_list_entries
SET
  is_active = false,
  updated_at = now()
WHERE list_id = sqlc.arg(list_id)::uuid
  AND is_active = true
  AND NOT (seed_name = ANY(sqlc.arg(seed_names)::text[]));

-- name: GetCityByHint :one
SELECT
  id::text AS id,
  slug,
  name,
  latitude,
  longitude
FROM cities
WHERE is_active = true
  AND (
    lower(name) = lower(sqlc.arg(city_hint))
    OR lower(slug) = lower(sqlc.arg(city_hint))
    OR lower(name) = lower(sqlc.arg(city_hint_alt))
    OR lower(slug) = lower(sqlc.arg(city_hint_alt))
  )
ORDER BY sort_order
LIMIT 1;

-- name: FindPlaceByNameInCity :one
SELECT id::text AS id
FROM places
WHERE city_id = sqlc.arg(city_id)::uuid
  AND is_active = true
  AND lower(trim(name)) = lower(trim(sqlc.arg(name)))
LIMIT 1;

-- name: UpsertGooglePlace :one
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
  sqlc.arg(neighborhood),
  sqlc.arg(address),
  sqlc.arg(latitude),
  sqlc.arg(longitude),
  sqlc.arg(price_level),
  sqlc.arg(rating_cached),
  sqlc.arg(cover_image_url),
  sqlc.arg(tags),
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
  cover_image_url = COALESCE(EXCLUDED.cover_image_url, places.cover_image_url),
  tags = CASE
    WHEN cardinality(EXCLUDED.tags) > 0 THEN EXCLUDED.tags
    ELSE places.tags
  END,
  last_synced_at = now(),
  updated_at = now()
RETURNING id::text AS id;

-- name: CreateListSyncRun :one
INSERT INTO list_sync_runs (status)
VALUES ('running')
RETURNING id::text AS id;

-- name: FinishListSyncRun :exec
UPDATE list_sync_runs
SET
  finished_at = now(),
  status = sqlc.arg(status),
  entries_added = sqlc.arg(entries_added)::integer,
  entries_skipped = sqlc.arg(entries_skipped)::integer,
  entries_failed = sqlc.arg(entries_failed)::integer,
  details = sqlc.arg(details)
WHERE id = sqlc.arg(id)::uuid;

-- name: ListRecentSyncRuns :many
SELECT
  id::text AS id,
  started_at,
  finished_at,
  status,
  entries_added,
  entries_skipped,
  entries_failed,
  details
FROM list_sync_runs
ORDER BY started_at DESC
LIMIT sqlc.arg(limit_val)::integer;
