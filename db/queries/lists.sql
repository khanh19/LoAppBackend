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
  COALESCE(cover.cover_place_id, '')::text AS cover_place_id,
  COALESCE(cover.cover_image_url, '')::text AS cover_image_url
FROM place_lists pl
JOIN cities c ON c.id = pl.city_id
LEFT JOIN LATERAL (
  SELECT
    p.id::text AS cover_place_id,
    CASE
      WHEN p.cover_image_url NOT ILIKE '%googleusercontent.com%'
        THEN p.cover_image_url
      ELSE NULL
    END AS cover_image_url
  FROM place_list_entries ple
  JOIN places p ON p.id = ple.place_id
  WHERE ple.list_id = pl.id
    AND ple.is_active = true
    AND (
      cardinality(p.photo_names) > 0
      OR (
        p.cover_image_url IS NOT NULL
        AND p.cover_image_url NOT ILIKE '%googleusercontent.com%'
      )
    )
  ORDER BY ple.stop_order
  LIMIT 1
) cover ON true
WHERE pl.is_active = true
  AND pl.list_type = 'curated'
ORDER BY pl.sort_order, pl.title;

-- name: ListActivePlans :many
SELECT
  pl.id::text AS id,
  pl.slug,
  pl.title,
  pl.subtitle,
  pl.category,
  pl.area,
  c.slug AS city_slug,
  c.name AS city_name,
  pl.saves_count,
  (
    SELECT COUNT(*)::integer
    FROM place_list_entries ple
    WHERE ple.list_id = pl.id
      AND ple.is_active = true
  ) AS stops_count,
  COALESCE(cover.cover_place_id, '')::text AS cover_place_id,
  COALESCE(cover.cover_image_url, '')::text AS cover_image_url,
  COALESCE(NULLIF(trim(concat_ws(' ', up.first_name, up.last_name)), ''), up.username::text, u.primary_email) AS creator_display_name
FROM place_lists pl
JOIN cities c ON c.id = pl.city_id
LEFT JOIN users u ON u.id = pl.creator_user_id
LEFT JOIN user_profiles up ON up.user_id = pl.creator_user_id
LEFT JOIN LATERAL (
  SELECT
    p.id::text AS cover_place_id,
    CASE
      WHEN ple.image_url NOT ILIKE '%googleusercontent.com%' THEN ple.image_url
      WHEN p.cover_image_url NOT ILIKE '%googleusercontent.com%' THEN p.cover_image_url
      ELSE NULL
    END AS cover_image_url
  FROM place_list_entries ple
  JOIN places p ON p.id = ple.place_id
  WHERE ple.list_id = pl.id
    AND ple.is_active = true
    AND (
      cardinality(p.photo_names) > 0
      OR (
        ple.image_url IS NOT NULL
        AND ple.image_url NOT ILIKE '%googleusercontent.com%'
      )
      OR (
        p.cover_image_url IS NOT NULL
        AND p.cover_image_url NOT ILIKE '%googleusercontent.com%'
      )
    )
  ORDER BY ple.stop_order
  LIMIT 1
) cover ON true
WHERE pl.is_active = true
  AND pl.list_type = 'user_plan'
  AND pl.visibility = 'public'
  AND (sqlc.narg(city_slug)::text IS NULL OR c.slug = sqlc.narg(city_slug))
ORDER BY pl.sort_order, pl.title
LIMIT sqlc.arg(limit_val)::int;

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
  ple.stop_order,
  ple.note,
  ple.time_label,
  ple.activity_type,
  COALESCE(
    CASE
      WHEN ple.image_url NOT ILIKE '%googleusercontent.com%' THEN ple.image_url
      WHEN p.cover_image_url NOT ILIKE '%googleusercontent.com%' THEN p.cover_image_url
      ELSE NULL
    END,
    ''
  )::text AS image_url,
  p.id::text AS place_id,
  p.name AS place_name,
  p.google_place_id,
  p.address,
  p.neighborhood,
  p.latitude,
  p.longitude,
  p.rating_cached,
  p.price_level,
  CASE
    WHEN p.cover_image_url IS NOT NULL
      AND p.cover_image_url NOT ILIKE '%googleusercontent.com%'
      THEN p.cover_image_url
    ELSE ''
  END AS cover_image_url,
  p.tags
FROM place_list_entries ple
JOIN places p ON p.id = ple.place_id
WHERE ple.list_id = sqlc.arg(list_id)::uuid
  AND ple.is_active = true
ORDER BY ple.stop_order;

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
  COALESCE(sqlc.arg(occasions), '{}'::text[]),
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
  COALESCE(sqlc.arg(tags), '{}'::text[]),
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
  stop_order,
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
  sqlc.arg(stop_order)::integer,
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
  stop_order,
  note,
  is_active
)
VALUES (
  sqlc.arg(list_id)::uuid,
  sqlc.arg(place_id)::uuid,
  sqlc.arg(seed_name),
  sqlc.arg(stop_order)::integer,
  sqlc.arg(note),
  true
)
ON CONFLICT (list_id, seed_name) DO UPDATE
SET
  place_id = EXCLUDED.place_id,
  stop_order = EXCLUDED.stop_order,
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

-- name: RelinkPlaceListEntries :exec
UPDATE place_list_entries
SET
  place_id = sqlc.arg(new_place_id)::uuid,
  updated_at = now()
WHERE place_id = sqlc.arg(old_place_id)::uuid
  AND NOT EXISTS (
    SELECT 1
    FROM place_list_entries other
    WHERE other.list_id = place_list_entries.list_id
      AND other.place_id = sqlc.arg(new_place_id)::uuid
  );

-- name: UpdatePlaceCoverOnly :exec
UPDATE places
SET
  neighborhood = COALESCE(sqlc.narg(neighborhood), neighborhood),
  address = COALESCE(sqlc.narg(address), address),
  latitude = COALESCE(sqlc.narg(latitude), latitude),
  longitude = COALESCE(sqlc.narg(longitude), longitude),
  price_level = COALESCE(sqlc.narg(price_level), price_level),
  rating_cached = COALESCE(sqlc.narg(rating_cached), rating_cached),
  cover_image_url = COALESCE(sqlc.narg(cover_image_url), cover_image_url),
  photo_names = CASE
    WHEN cardinality(COALESCE(sqlc.narg(photo_names), '{}'::text[])) > 0
      THEN COALESCE(sqlc.narg(photo_names), '{}'::text[])
    ELSE photo_names
  END,
  tags = CASE
    WHEN cardinality(COALESCE(sqlc.arg(tags), '{}'::text[])) > 0
      THEN COALESCE(sqlc.arg(tags), '{}'::text[])
    ELSE tags
  END,
  last_synced_at = now(),
  updated_at = now()
WHERE id = sqlc.arg(id)::uuid;

-- name: ListPlanPlacesMissingCover :many
SELECT
  p.id::text AS id,
  p.name,
  p.city_id::text AS city_id,
  c.slug AS city_slug,
  c.latitude,
  c.longitude
FROM places p
JOIN place_list_entries ple ON ple.place_id = p.id AND ple.is_active = true
JOIN place_lists pl ON pl.id = ple.list_id
JOIN cities c ON c.id = p.city_id
WHERE pl.is_active = true
  AND pl.list_type = 'user_plan'
  AND p.is_active = true
  AND (p.cover_image_url IS NULL OR btrim(p.cover_image_url) = '')
GROUP BY p.id, p.name, p.city_id, c.slug, c.latitude, c.longitude
ORDER BY p.name
LIMIT sqlc.arg(limit_val)::integer;

-- name: UpdatePlaceFromGoogleResolve :exec
UPDATE places
SET
  google_place_id = sqlc.arg(google_place_id),
  source = 'google',
  name = sqlc.arg(name),
  neighborhood = COALESCE(sqlc.narg(neighborhood), neighborhood),
  address = COALESCE(sqlc.narg(address), address),
  latitude = COALESCE(sqlc.narg(latitude), latitude),
  longitude = COALESCE(sqlc.narg(longitude), longitude),
  price_level = COALESCE(sqlc.narg(price_level), price_level),
  rating_cached = COALESCE(sqlc.narg(rating_cached), rating_cached),
  cover_image_url = COALESCE(sqlc.narg(cover_image_url), cover_image_url),
  photo_names = CASE
    WHEN cardinality(COALESCE(sqlc.narg(photo_names), '{}'::text[])) > 0
      THEN COALESCE(sqlc.narg(photo_names), '{}'::text[])
    ELSE photo_names
  END,
  tags = CASE
    WHEN cardinality(COALESCE(sqlc.arg(tags), '{}'::text[])) > 0
      THEN COALESCE(sqlc.arg(tags), '{}'::text[])
    ELSE tags
  END,
  last_synced_at = now(),
  updated_at = now()
WHERE id = sqlc.arg(id)::uuid;

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
  photo_names,
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
  COALESCE(sqlc.narg(photo_names), '{}'::text[]),
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
  photo_names = CASE
    WHEN cardinality(EXCLUDED.photo_names) > 0 THEN EXCLUDED.photo_names
    ELSE places.photo_names
  END,
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

-- ---------------------------------------------------------------------------
-- User library: place playlists, place bookmarks, plan bookmarks, my plans
-- ---------------------------------------------------------------------------

-- name: ListUserPlaceLists :many
SELECT
  upl.id::text AS id,
  upl.title,
  upl.created_at,
  upl.updated_at,
  (
    SELECT COUNT(*)::integer
    FROM user_place_list_items upli
    WHERE upli.list_id = upl.id
  ) AS item_count,
  COALESCE(cover.cover_image_url, '')::text AS cover_image_url
FROM user_place_lists upl
LEFT JOIN LATERAL (
  SELECT
    CASE
      WHEN p.cover_image_url NOT ILIKE '%googleusercontent.com%'
        THEN p.cover_image_url
      ELSE NULL
    END AS cover_image_url
  FROM user_place_list_items upli
  JOIN places p ON p.id = upli.place_id AND p.is_active = true
  WHERE upli.list_id = upl.id
    AND (
      cardinality(p.photo_names) > 0
      OR (
        p.cover_image_url IS NOT NULL
        AND p.cover_image_url NOT ILIKE '%googleusercontent.com%'
      )
    )
  ORDER BY upli.position, upli.created_at
  LIMIT 1
) cover ON true
WHERE upl.owner_user_id = sqlc.arg(owner_user_id)::uuid
  AND (
    sqlc.narg(q)::text IS NULL
    OR btrim(sqlc.narg(q)::text) = ''
    OR upl.title ILIKE '%' || btrim(sqlc.narg(q)::text) || '%'
  )
ORDER BY
  CASE WHEN sqlc.arg(sort)::text = 'name' THEN lower(upl.title) END ASC,
  CASE WHEN sqlc.arg(sort)::text <> 'name' THEN upl.updated_at END DESC;

-- name: CreateUserPlaceList :one
INSERT INTO user_place_lists (owner_user_id, title)
VALUES (sqlc.arg(owner_user_id)::uuid, sqlc.arg(title))
RETURNING id::text AS id, title, created_at, updated_at;

-- name: GetUserPlaceListByID :one
SELECT
  upl.id::text AS id,
  upl.owner_user_id::text AS owner_user_id,
  upl.title,
  upl.created_at,
  upl.updated_at,
  (
    SELECT COUNT(*)::integer
    FROM user_place_list_items upli
    WHERE upli.list_id = upl.id
  ) AS item_count,
  COALESCE(cover.cover_image_url, '')::text AS cover_image_url
FROM user_place_lists upl
LEFT JOIN LATERAL (
  SELECT
    CASE
      WHEN p.cover_image_url NOT ILIKE '%googleusercontent.com%'
        THEN p.cover_image_url
      ELSE NULL
    END AS cover_image_url
  FROM user_place_list_items upli
  JOIN places p ON p.id = upli.place_id AND p.is_active = true
  WHERE upli.list_id = upl.id
    AND (
      cardinality(p.photo_names) > 0
      OR (
        p.cover_image_url IS NOT NULL
        AND p.cover_image_url NOT ILIKE '%googleusercontent.com%'
      )
    )
  ORDER BY upli.position, upli.created_at
  LIMIT 1
) cover ON true
WHERE upl.id = sqlc.arg(id)::uuid;

-- name: DeleteUserPlaceList :execrows
DELETE FROM user_place_lists
WHERE id = sqlc.arg(id)::uuid
  AND owner_user_id = sqlc.arg(owner_user_id)::uuid;

-- name: ListUserPlaceListItems :many
SELECT
  p.id::text AS place_id,
  p.name,
  p.neighborhood,
  p.address,
  p.cover_image_url,
  c.slug AS city_slug,
  c.name AS city_name,
  upli.position,
  upli.created_at
FROM user_place_list_items upli
JOIN places p ON p.id = upli.place_id AND p.is_active = true
JOIN cities c ON c.id = p.city_id
WHERE upli.list_id = sqlc.arg(list_id)::uuid
ORDER BY upli.position, upli.created_at;

-- name: AddUserPlaceListItem :exec
INSERT INTO user_place_list_items (list_id, place_id, position)
VALUES (
  sqlc.arg(list_id)::uuid,
  sqlc.arg(place_id)::uuid,
  COALESCE(
    (
      SELECT MAX(position) + 1
      FROM user_place_list_items
      WHERE list_id = sqlc.arg(list_id)::uuid
    ),
    0
  )
)
ON CONFLICT (list_id, place_id) DO NOTHING;

-- name: RemoveUserPlaceListItem :execrows
DELETE FROM user_place_list_items
WHERE list_id = sqlc.arg(list_id)::uuid
  AND place_id = sqlc.arg(place_id)::uuid;

-- name: TouchUserPlaceList :exec
UPDATE user_place_lists
SET updated_at = now()
WHERE id = sqlc.arg(id)::uuid;

-- name: CountPlaceBookmarks :one
SELECT COUNT(*)::integer AS count
FROM place_bookmarks
WHERE user_id = sqlc.arg(user_id)::uuid;

-- name: ListPlaceBookmarks :many
SELECT
  p.id::text AS place_id,
  p.name,
  p.neighborhood,
  p.address,
  p.cover_image_url,
  p.price_level,
  p.tags,
  p.rating_cached,
  c.slug AS city_slug,
  c.name AS city_name,
  pb.created_at
FROM place_bookmarks pb
JOIN places p ON p.id = pb.place_id AND p.is_active = true
JOIN cities c ON c.id = p.city_id
WHERE pb.user_id = sqlc.arg(user_id)::uuid
  AND (
    sqlc.narg(q)::text IS NULL
    OR btrim(sqlc.narg(q)::text) = ''
    OR p.name ILIKE '%' || btrim(sqlc.narg(q)::text) || '%'
  )
ORDER BY
  CASE WHEN sqlc.arg(sort)::text = 'name' THEN lower(p.name) END ASC,
  CASE WHEN sqlc.arg(sort)::text <> 'name' THEN pb.created_at END DESC;

-- name: UpsertPlaceBookmark :exec
INSERT INTO place_bookmarks (user_id, place_id)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(place_id)::uuid)
ON CONFLICT (user_id, place_id) DO NOTHING;

-- name: DeletePlaceBookmark :execrows
DELETE FROM place_bookmarks
WHERE user_id = sqlc.arg(user_id)::uuid
  AND place_id = sqlc.arg(place_id)::uuid;

-- name: PlaceExists :one
SELECT id::text AS id
FROM places
WHERE id = sqlc.arg(id)::uuid
  AND is_active = true;

-- name: ListUserStampedPlaces :many
SELECT
  p.id::text AS place_id,
  p.name,
  p.neighborhood,
  p.address,
  p.cover_image_url,
  p.price_level,
  p.tags,
  c.slug AS city_slug,
  c.name AS city_name,
  MAX(s.created_at)::timestamptz AS stamped_at,
  pe.score AS stamp_score
FROM stamps s
JOIN places p ON p.id = s.place_id AND p.is_active = true
JOIN cities c ON c.id = p.city_id
LEFT JOIN user_pool_entries pe
  ON pe.user_id = s.user_id AND pe.place_id = p.id
WHERE s.user_id = sqlc.arg(user_id)::uuid
  AND s.is_active = true
  AND (
    sqlc.narg(q)::text IS NULL
    OR btrim(sqlc.narg(q)::text) = ''
    OR p.name ILIKE '%' || btrim(sqlc.narg(q)::text) || '%'
  )
GROUP BY
  p.id, p.name, p.neighborhood, p.address, p.cover_image_url,
  p.price_level, p.tags, c.slug, c.name, pe.score
ORDER BY
  CASE WHEN sqlc.arg(sort)::text = 'name' THEN lower(p.name) ELSE NULL END ASC NULLS LAST,
  CASE WHEN sqlc.arg(sort)::text = 'score' THEN pe.score ELSE NULL END DESC NULLS LAST,
  CASE WHEN sqlc.arg(sort)::text = 'score' OR sqlc.arg(sort)::text = 'name' THEN NULL ELSE MAX(s.created_at) END DESC NULLS LAST;

-- name: ListMyPlans :many
SELECT
  pl.id::text AS id,
  pl.slug,
  pl.title,
  pl.subtitle,
  pl.category,
  pl.area,
  c.slug AS city_slug,
  c.name AS city_name,
  pl.saves_count,
  pl.visibility,
  pl.created_at,
  (
    SELECT COUNT(*)::integer
    FROM place_list_entries ple
    WHERE ple.list_id = pl.id
      AND ple.is_active = true
  ) AS stops_count,
  COALESCE(cover.cover_place_id, '')::text AS cover_place_id,
  COALESCE(cover.cover_image_url, '')::text AS cover_image_url,
  COALESCE(NULLIF(trim(concat_ws(' ', up.first_name, up.last_name)), ''), up.username::text, u.primary_email) AS creator_display_name,
  COALESCE(up.username::text, ''::text)::text AS creator_username,
  up.avatar_url AS creator_avatar_url
FROM place_lists pl
JOIN cities c ON c.id = pl.city_id
LEFT JOIN users u ON u.id = pl.creator_user_id
LEFT JOIN user_profiles up ON up.user_id = pl.creator_user_id
LEFT JOIN LATERAL (
  SELECT
    p.id::text AS cover_place_id,
    CASE
      WHEN ple.image_url NOT ILIKE '%googleusercontent.com%' THEN ple.image_url
      WHEN p.cover_image_url NOT ILIKE '%googleusercontent.com%' THEN p.cover_image_url
      ELSE NULL
    END AS cover_image_url
  FROM place_list_entries ple
  JOIN places p ON p.id = ple.place_id
  WHERE ple.list_id = pl.id
    AND ple.is_active = true
    AND (
      cardinality(p.photo_names) > 0
      OR (
        ple.image_url IS NOT NULL
        AND ple.image_url NOT ILIKE '%googleusercontent.com%'
      )
      OR (
        p.cover_image_url IS NOT NULL
        AND p.cover_image_url NOT ILIKE '%googleusercontent.com%'
      )
    )
  ORDER BY ple.stop_order
  LIMIT 1
) cover ON true
WHERE pl.is_active = true
  AND pl.list_type = 'user_plan'
  AND pl.creator_user_id = sqlc.arg(creator_user_id)::uuid
  AND (
    sqlc.narg(q)::text IS NULL
    OR btrim(sqlc.narg(q)::text) = ''
    OR pl.title ILIKE '%' || btrim(sqlc.narg(q)::text) || '%'
  )
ORDER BY
  CASE WHEN sqlc.arg(sort)::text = 'name' THEN lower(pl.title) END ASC,
  CASE WHEN sqlc.arg(sort)::text <> 'name' THEN pl.created_at END DESC
LIMIT sqlc.arg(limit_val)::int
OFFSET sqlc.arg(offset_val)::int;

-- name: CountMyPlans :one
SELECT COUNT(*)::integer AS count
FROM place_lists pl
WHERE pl.is_active = true
  AND pl.list_type = 'user_plan'
  AND pl.creator_user_id = sqlc.arg(creator_user_id)::uuid
  AND (
    sqlc.narg(q)::text IS NULL
    OR btrim(sqlc.narg(q)::text) = ''
    OR pl.title ILIKE '%' || btrim(sqlc.narg(q)::text) || '%'
  );

-- name: ListBookmarkedPlans :many
SELECT
  pl.id::text AS id,
  pl.slug,
  pl.title,
  pl.subtitle,
  pl.category,
  pl.area,
  c.slug AS city_slug,
  c.name AS city_name,
  pl.saves_count,
  pl.visibility,
  pb.created_at AS bookmarked_at,
  pl.created_at,
  (
    SELECT COUNT(*)::integer
    FROM place_list_entries ple
    WHERE ple.list_id = pl.id
      AND ple.is_active = true
  ) AS stops_count,
  COALESCE(cover.cover_place_id, '')::text AS cover_place_id,
  COALESCE(cover.cover_image_url, '')::text AS cover_image_url,
  COALESCE(NULLIF(trim(concat_ws(' ', up.first_name, up.last_name)), ''), up.username::text, u.primary_email) AS creator_display_name,
  COALESCE(up.username::text, ''::text)::text AS creator_username,
  up.avatar_url AS creator_avatar_url
FROM plan_bookmarks pb
JOIN place_lists pl ON pl.id = pb.list_id AND pl.is_active = true AND pl.list_type = 'user_plan'
JOIN cities c ON c.id = pl.city_id
LEFT JOIN users u ON u.id = pl.creator_user_id
LEFT JOIN user_profiles up ON up.user_id = pl.creator_user_id
LEFT JOIN LATERAL (
  SELECT
    p.id::text AS cover_place_id,
    CASE
      WHEN ple.image_url NOT ILIKE '%googleusercontent.com%' THEN ple.image_url
      WHEN p.cover_image_url NOT ILIKE '%googleusercontent.com%' THEN p.cover_image_url
      ELSE NULL
    END AS cover_image_url
  FROM place_list_entries ple
  JOIN places p ON p.id = ple.place_id
  WHERE ple.list_id = pl.id
    AND ple.is_active = true
    AND (
      cardinality(p.photo_names) > 0
      OR (
        ple.image_url IS NOT NULL
        AND ple.image_url NOT ILIKE '%googleusercontent.com%'
      )
      OR (
        p.cover_image_url IS NOT NULL
        AND p.cover_image_url NOT ILIKE '%googleusercontent.com%'
      )
    )
  ORDER BY ple.stop_order
  LIMIT 1
) cover ON true
WHERE pb.user_id = sqlc.arg(user_id)::uuid
  AND (
    sqlc.narg(q)::text IS NULL
    OR btrim(sqlc.narg(q)::text) = ''
    OR pl.title ILIKE '%' || btrim(sqlc.narg(q)::text) || '%'
  )
ORDER BY
  CASE WHEN sqlc.arg(sort)::text = 'name' THEN lower(pl.title) END ASC,
  CASE WHEN sqlc.arg(sort)::text <> 'name' THEN pb.created_at END DESC
LIMIT sqlc.arg(limit_val)::int
OFFSET sqlc.arg(offset_val)::int;

-- name: CountBookmarkedPlans :one
SELECT COUNT(*)::integer AS count
FROM plan_bookmarks pb
JOIN place_lists pl ON pl.id = pb.list_id AND pl.is_active = true AND pl.list_type = 'user_plan'
WHERE pb.user_id = sqlc.arg(user_id)::uuid
  AND (
    sqlc.narg(q)::text IS NULL
    OR btrim(sqlc.narg(q)::text) = ''
    OR pl.title ILIKE '%' || btrim(sqlc.narg(q)::text) || '%'
  );

-- name: GetPlanIDBySlug :one
SELECT id::text AS id, saves_count
FROM place_lists
WHERE slug = sqlc.arg(slug)
  AND is_active = true
  AND list_type = 'user_plan';

-- name: UpsertPlanBookmark :exec
INSERT INTO plan_bookmarks (user_id, list_id)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(list_id)::uuid)
ON CONFLICT (user_id, list_id) DO NOTHING;

-- name: DeletePlanBookmark :execrows
DELETE FROM plan_bookmarks
WHERE user_id = sqlc.arg(user_id)::uuid
  AND list_id = sqlc.arg(list_id)::uuid;

-- name: IncrementPlanSavesCount :exec
UPDATE place_lists
SET saves_count = saves_count + 1, updated_at = now()
WHERE id = sqlc.arg(id)::uuid;

-- name: DecrementPlanSavesCount :exec
UPDATE place_lists
SET saves_count = GREATEST(saves_count - 1, 0), updated_at = now()
WHERE id = sqlc.arg(id)::uuid;

-- name: HasPlanBookmark :one
SELECT EXISTS (
  SELECT 1
  FROM plan_bookmarks
  WHERE user_id = sqlc.arg(user_id)::uuid
    AND list_id = sqlc.arg(list_id)::uuid
)::boolean AS exists;

-- name: UpdateUserPlanTitle :execrows
UPDATE place_lists
SET title = sqlc.arg(title),
    updated_at = now()
WHERE id = sqlc.arg(id)::uuid
  AND list_type = 'user_plan'
  AND is_active = true
  AND creator_user_id = sqlc.arg(creator_user_id)::uuid;

-- name: DeletePlaceListEntriesByListID :exec
DELETE FROM place_list_entries
WHERE list_id = sqlc.arg(list_id)::uuid;

-- name: DeactivateUserPlan :execrows
UPDATE place_lists
SET is_active = false,
    updated_at = now()
WHERE id = sqlc.arg(id)::uuid
  AND list_type = 'user_plan'
  AND is_active = true
  AND creator_user_id = sqlc.arg(creator_user_id)::uuid;

-- name: ListFeedPlans :many
SELECT
  pl.id::text AS plan_id,
  pl.slug,
  pl.title,
  pl.category,
  pl.saves_count,
  pl.visibility,
  pl.created_at,
  (
    SELECT COUNT(*)::integer
    FROM place_list_entries ple
    WHERE ple.list_id = pl.id
      AND ple.is_active = true
  ) AS stops_count,
  COALESCE(cover.cover_place_id, '')::text AS cover_place_id,
  COALESCE(cover.cover_image_url, '')::text AS cover_image_url,
  COALESCE(pl.creator_user_id::text, ''::text)::text AS creator_user_id,
  COALESCE(NULLIF(trim(concat_ws(' ', up.first_name, up.last_name)), ''), up.username::text, u.primary_email) AS creator_display_name,
  COALESCE(up.username::text, ''::text)::text AS creator_username,
  up.avatar_url AS creator_avatar_url
FROM place_lists pl
LEFT JOIN users u ON u.id = pl.creator_user_id
LEFT JOIN user_profiles up ON up.user_id = pl.creator_user_id
LEFT JOIN LATERAL (
  SELECT
    p.id::text AS cover_place_id,
    CASE
      WHEN ple.image_url NOT ILIKE '%googleusercontent.com%' THEN ple.image_url
      WHEN p.cover_image_url NOT ILIKE '%googleusercontent.com%' THEN p.cover_image_url
      ELSE NULL
    END AS cover_image_url
  FROM place_list_entries ple
  JOIN places p ON p.id = ple.place_id
  WHERE ple.list_id = pl.id
    AND ple.is_active = true
    AND (
      cardinality(p.photo_names) > 0
      OR (
        ple.image_url IS NOT NULL
        AND ple.image_url NOT ILIKE '%googleusercontent.com%'
      )
      OR (
        p.cover_image_url IS NOT NULL
        AND p.cover_image_url NOT ILIKE '%googleusercontent.com%'
      )
    )
  ORDER BY ple.stop_order
  LIMIT 1
) cover ON true
WHERE pl.is_active = true
  AND pl.list_type = 'user_plan'
  AND pl.creator_user_id IS NOT NULL
  AND (
    pl.creator_user_id = sqlc.arg(viewer_id)::uuid
    OR pl.creator_user_id IN (
      SELECT uf.following_user_id
      FROM user_follows uf
      WHERE uf.follower_user_id = sqlc.arg(viewer_id)::uuid
    )
  )
ORDER BY pl.created_at DESC
LIMIT sqlc.arg(limit_val)::int
OFFSET sqlc.arg(offset_val)::int;
