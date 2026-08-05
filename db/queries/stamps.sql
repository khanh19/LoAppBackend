-- name: GetRankingConfig :one
SELECT key, value, updated_at
FROM ranking_config
WHERE key = sqlc.arg(key);

-- name: ListRankingConfig :many
SELECT key, value, updated_at
FROM ranking_config
ORDER BY key;

-- name: GetStampPlaceContext :one
SELECT
  p.id::text AS id,
  p.name,
  p.city_id::text AS city_id,
  c.slug AS city_slug,
  p.neighborhood,
  p.address,
  p.latitude,
  p.longitude,
  p.price_level,
  p.cover_image_url,
  p.venue_category,
  p.venue_archetype,
  p.tags
FROM places p
JOIN cities c ON c.id = p.city_id
WHERE p.id = sqlc.arg(id)::uuid
  AND p.is_active = true;

-- name: GetActiveStampByUserPlace :one
SELECT
  id::text AS id,
  user_id::text AS user_id,
  place_id::text AS place_id,
  venue_category,
  verdict,
  quality,
  note,
  visit_count,
  verification_level,
  weight,
  device_latitude,
  device_longitude,
  created_at,
  updated_at
FROM stamps
WHERE user_id = sqlc.arg(user_id)::uuid
  AND place_id = sqlc.arg(place_id)::uuid
  AND is_active = true;

-- name: GetStampByID :one
SELECT
  id::text AS id,
  user_id::text AS user_id,
  place_id::text AS place_id,
  venue_category,
  verdict,
  quality,
  note,
  visit_count,
  verification_level,
  weight,
  device_latitude,
  device_longitude,
  created_at,
  updated_at
FROM stamps
WHERE id = sqlc.arg(id)::uuid
  AND is_active = true;

-- name: UpsertStamp :one
INSERT INTO stamps (
  user_id,
  place_id,
  venue_category,
  verdict,
  quality,
  note,
  visit_count,
  verification_level,
  weight,
  device_latitude,
  device_longitude,
  is_active
)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.arg(place_id)::uuid,
  sqlc.arg(venue_category),
  sqlc.arg(verdict),
  sqlc.arg(quality),
  COALESCE(sqlc.arg(note), ''),
  1,
  sqlc.arg(verification_level),
  sqlc.arg(weight),
  sqlc.narg(device_latitude),
  sqlc.narg(device_longitude),
  true
)
ON CONFLICT (user_id, place_id) WHERE is_active = true
DO UPDATE SET
  venue_category = EXCLUDED.venue_category,
  verdict = EXCLUDED.verdict,
  quality = EXCLUDED.quality,
  note = EXCLUDED.note,
  visit_count = stamps.visit_count + 1,
  verification_level = EXCLUDED.verification_level,
  weight = EXCLUDED.weight,
  device_latitude = EXCLUDED.device_latitude,
  device_longitude = EXCLUDED.device_longitude,
  updated_at = now()
RETURNING
  id::text AS id,
  user_id::text AS user_id,
  place_id::text AS place_id,
  venue_category,
  verdict,
  quality,
  note,
  visit_count,
  verification_level,
  weight,
  device_latitude,
  device_longitude,
  created_at,
  updated_at;

-- name: DeleteStampTags :exec
DELETE FROM stamp_tags
WHERE stamp_id = sqlc.arg(stamp_id)::uuid;

-- name: InsertStampTag :exec
INSERT INTO stamp_tags (stamp_id, tag_type, tag_slug, position, is_custom)
VALUES (
  sqlc.arg(stamp_id)::uuid,
  sqlc.arg(tag_type),
  sqlc.arg(tag_slug),
  sqlc.arg(position),
  sqlc.arg(is_custom)
)
ON CONFLICT (stamp_id, tag_type, tag_slug) DO UPDATE
SET position = EXCLUDED.position,
    is_custom = EXCLUDED.is_custom;

-- name: ListStampTags :many
SELECT stamp_id::text AS stamp_id, tag_type, tag_slug, position, is_custom
FROM stamp_tags
WHERE stamp_id = sqlc.arg(stamp_id)::uuid
ORDER BY tag_type, position, tag_slug;

-- name: UpsertUserPoolEntry :one
INSERT INTO user_pool_entries (
  user_id,
  place_id,
  venue_category,
  band,
  rank_key,
  score,
  placement_confidence,
  comparisons_count,
  contradiction_count,
  is_provisional
)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.arg(place_id)::uuid,
  sqlc.arg(venue_category),
  sqlc.arg(band),
  sqlc.arg(rank_key),
  sqlc.arg(score),
  sqlc.arg(placement_confidence),
  sqlc.arg(comparisons_count),
  sqlc.arg(contradiction_count),
  sqlc.arg(is_provisional)
)
ON CONFLICT (user_id, place_id) DO UPDATE
SET
  venue_category = EXCLUDED.venue_category,
  band = EXCLUDED.band,
  rank_key = EXCLUDED.rank_key,
  score = EXCLUDED.score,
  placement_confidence = EXCLUDED.placement_confidence,
  comparisons_count = EXCLUDED.comparisons_count,
  contradiction_count = EXCLUDED.contradiction_count,
  is_provisional = EXCLUDED.is_provisional,
  updated_at = now()
RETURNING
  user_id::text AS user_id,
  place_id::text AS place_id,
  venue_category,
  band,
  rank_key,
  score,
  placement_confidence,
  comparisons_count,
  contradiction_count,
  is_provisional;

-- name: ListUserPoolBand :many
SELECT
  upe.user_id::text AS user_id,
  upe.place_id::text AS place_id,
  upe.venue_category,
  upe.band,
  upe.rank_key,
  upe.score,
  upe.placement_confidence,
  upe.comparisons_count,
  upe.contradiction_count,
  upe.is_provisional,
  p.name AS place_name,
  p.neighborhood,
  p.price_level,
  p.cover_image_url,
  p.venue_archetype,
  p.tags
FROM user_pool_entries upe
JOIN places p ON p.id = upe.place_id
WHERE upe.user_id = sqlc.arg(user_id)::uuid
  AND upe.venue_category = sqlc.arg(venue_category)
  AND upe.band = sqlc.arg(band)
ORDER BY upe.rank_key ASC;

-- name: ListUserPoolByCategory :many
SELECT
  upe.user_id::text AS user_id,
  upe.place_id::text AS place_id,
  upe.venue_category,
  upe.band,
  upe.rank_key,
  upe.score,
  upe.placement_confidence,
  upe.comparisons_count,
  upe.contradiction_count,
  upe.is_provisional,
  p.name AS place_name,
  p.neighborhood,
  p.price_level,
  p.cover_image_url,
  p.venue_archetype,
  p.tags
FROM user_pool_entries upe
JOIN places p ON p.id = upe.place_id
WHERE upe.user_id = sqlc.arg(user_id)::uuid
  AND upe.venue_category = sqlc.arg(venue_category)
ORDER BY
  CASE upe.band
    WHEN 'must-repeat' THEN 1
    WHEN 'good-pick' THEN 2
    WHEN 'mid' THEN 3
    ELSE 4
  END,
  upe.rank_key ASC;

-- name: GetUserPoolEntry :one
SELECT
  upe.user_id::text AS user_id,
  upe.place_id::text AS place_id,
  upe.venue_category,
  upe.band,
  upe.rank_key,
  upe.score,
  upe.placement_confidence,
  upe.comparisons_count,
  upe.contradiction_count,
  upe.is_provisional
FROM user_pool_entries upe
WHERE upe.user_id = sqlc.arg(user_id)::uuid
  AND upe.place_id = sqlc.arg(place_id)::uuid;

-- name: UpdateUserPoolEntryScores :exec
UPDATE user_pool_entries
SET rank_key = sqlc.arg(rank_key),
    score = sqlc.arg(score),
    placement_confidence = sqlc.arg(placement_confidence),
    comparisons_count = sqlc.arg(comparisons_count),
    contradiction_count = sqlc.arg(contradiction_count),
    updated_at = now()
WHERE user_id = sqlc.arg(user_id)::uuid
  AND place_id = sqlc.arg(place_id)::uuid;

-- name: InsertPairwiseComparison :one
INSERT INTO pairwise_comparisons (
  user_id,
  stamp_id,
  place_a_id,
  place_b_id,
  winner_place_id,
  venue_category,
  band,
  tier,
  comparability,
  surface
)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.narg(stamp_id)::uuid,
  sqlc.arg(place_a_id)::uuid,
  sqlc.arg(place_b_id)::uuid,
  sqlc.narg(winner_place_id)::uuid,
  sqlc.arg(venue_category),
  sqlc.arg(band),
  sqlc.arg(tier),
  sqlc.arg(comparability),
  sqlc.arg(surface)
)
RETURNING id::text AS id;

-- name: SumBandComparability :one
SELECT COALESCE(SUM(comparability), 0)::float8 AS total
FROM pairwise_comparisons
WHERE user_id = sqlc.arg(user_id)::uuid
  AND venue_category = sqlc.arg(venue_category)
  AND band = sqlc.arg(band);

-- name: CreatePlacementSession :one
INSERT INTO placement_sessions (
  user_id,
  stamp_id,
  place_id,
  venue_category,
  band,
  lo_index,
  hi_index,
  comparisons_done,
  is_complete
)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.arg(stamp_id)::uuid,
  sqlc.arg(place_id)::uuid,
  sqlc.arg(venue_category),
  sqlc.arg(band),
  sqlc.arg(lo_index),
  sqlc.arg(hi_index),
  sqlc.arg(comparisons_done),
  sqlc.arg(is_complete)
)
RETURNING
  id::text AS id,
  user_id::text AS user_id,
  stamp_id::text AS stamp_id,
  place_id::text AS place_id,
  venue_category,
  band,
  lo_index,
  hi_index,
  comparisons_done,
  is_complete;

-- name: GetActivePlacementSession :one
SELECT
  id::text AS id,
  user_id::text AS user_id,
  stamp_id::text AS stamp_id,
  place_id::text AS place_id,
  venue_category,
  band,
  lo_index,
  hi_index,
  comparisons_done,
  is_complete
FROM placement_sessions
WHERE stamp_id = sqlc.arg(stamp_id)::uuid
  AND is_complete = false
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdatePlacementSession :one
UPDATE placement_sessions
SET lo_index = sqlc.arg(lo_index),
    hi_index = sqlc.arg(hi_index),
    comparisons_done = sqlc.arg(comparisons_done),
    is_complete = sqlc.arg(is_complete),
    updated_at = now()
WHERE id = sqlc.arg(id)::uuid
RETURNING
  id::text AS id,
  user_id::text AS user_id,
  stamp_id::text AS stamp_id,
  place_id::text AS place_id,
  venue_category,
  band,
  lo_index,
  hi_index,
  comparisons_done,
  is_complete;

-- name: ListLikedPlacesForPairwise :many
SELECT
  p.id::text AS id,
  p.name,
  p.neighborhood,
  p.price_level,
  p.cover_image_url,
  p.venue_category,
  p.venue_archetype,
  p.tags,
  p.city_id::text AS city_id
FROM user_liked_places ulp
JOIN places p ON p.id = ulp.place_id
WHERE ulp.user_id = sqlc.arg(user_id)::uuid
  AND p.is_active = true
  AND p.venue_category = sqlc.arg(venue_category)
  AND p.city_id = sqlc.arg(city_id)::uuid
  AND p.id <> sqlc.arg(exclude_place_id)::uuid
  AND NOT EXISTS (
    SELECT 1 FROM user_pool_entries upe
    WHERE upe.user_id = ulp.user_id AND upe.place_id = p.id
  );

-- name: ListCuratedPlacesForProbe :many
SELECT
  p.id::text AS id,
  p.name,
  p.neighborhood,
  p.price_level,
  p.cover_image_url,
  p.venue_category,
  p.venue_archetype,
  p.tags
FROM places p
WHERE p.is_active = true
  AND p.source = 'curated'
  AND p.venue_category = sqlc.arg(venue_category)
  AND p.city_id = sqlc.arg(city_id)::uuid
  AND p.id <> sqlc.arg(exclude_place_id)::uuid
  AND NOT EXISTS (
    SELECT 1 FROM user_place_familiarity upf
    WHERE upf.user_id = sqlc.arg(user_id)::uuid
      AND upf.place_id = p.id
      AND upf.status IN ('been', 'not_been')
  )
  AND NOT EXISTS (
    SELECT 1 FROM user_pool_entries upe
    WHERE upe.user_id = sqlc.arg(user_id)::uuid AND upe.place_id = p.id
  )
ORDER BY p.name
LIMIT 10;

-- name: UpsertFamiliarity :exec
INSERT INTO user_place_familiarity (user_id, place_id, status)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.arg(place_id)::uuid,
  sqlc.arg(status)
)
ON CONFLICT (user_id, place_id) DO UPDATE
SET status = EXCLUDED.status,
    updated_at = now();

-- name: GetFamiliarity :one
SELECT user_id::text AS user_id, place_id::text AS place_id, status
FROM user_place_familiarity
WHERE user_id = sqlc.arg(user_id)::uuid
  AND place_id = sqlc.arg(place_id)::uuid;

-- name: ListLowConfidencePoolEntries :many
SELECT
  upe.user_id::text AS user_id,
  upe.place_id::text AS place_id,
  upe.venue_category,
  upe.band,
  upe.rank_key,
  upe.score,
  upe.placement_confidence,
  upe.comparisons_count,
  p.name AS place_name,
  p.neighborhood,
  p.cover_image_url,
  p.venue_archetype,
  p.price_level,
  p.tags
FROM user_pool_entries upe
JOIN places p ON p.id = upe.place_id
WHERE upe.user_id = sqlc.arg(user_id)::uuid
  AND upe.placement_confidence < 1
  AND upe.is_provisional = false
ORDER BY upe.placement_confidence ASC, upe.updated_at ASC
LIMIT sqlc.arg(limit_val)::integer;

-- name: ListPlaceStampsForAggregate :many
SELECT
  s.id::text AS id,
  s.verdict,
  s.quality,
  s.weight,
  s.created_at
FROM stamps s
WHERE s.place_id = sqlc.arg(place_id)::uuid
  AND s.is_active = true;

-- name: CountPlacePairwise :one
SELECT COUNT(*)::integer AS count
FROM pairwise_comparisons
WHERE place_a_id = sqlc.arg(place_id)::uuid
   OR place_b_id = sqlc.arg(place_id)::uuid;

-- name: CountPlacePairwiseWins :one
SELECT COUNT(*)::integer AS count
FROM pairwise_comparisons
WHERE winner_place_id = sqlc.arg(place_id)::uuid;

-- name: UpsertVenueStampAggregate :exec
INSERT INTO venue_stamp_aggregates (
  place_id,
  stamp_count,
  weighted_stamp_sum,
  sentiment_score,
  quality_score,
  pairwise_strength,
  pairwise_count,
  composite_score,
  final_score,
  display_state,
  vibe_conflict,
  quality_conflict,
  conflict_note,
  velocity_score
)
VALUES (
  sqlc.arg(place_id)::uuid,
  sqlc.arg(stamp_count),
  sqlc.arg(weighted_stamp_sum),
  sqlc.narg(sentiment_score),
  sqlc.narg(quality_score),
  sqlc.narg(pairwise_strength),
  sqlc.arg(pairwise_count),
  sqlc.narg(composite_score),
  sqlc.narg(final_score),
  sqlc.arg(display_state),
  sqlc.arg(vibe_conflict),
  sqlc.arg(quality_conflict),
  sqlc.narg(conflict_note),
  sqlc.arg(velocity_score)
)
ON CONFLICT (place_id) DO UPDATE
SET
  stamp_count = EXCLUDED.stamp_count,
  weighted_stamp_sum = EXCLUDED.weighted_stamp_sum,
  sentiment_score = EXCLUDED.sentiment_score,
  quality_score = EXCLUDED.quality_score,
  pairwise_strength = EXCLUDED.pairwise_strength,
  pairwise_count = EXCLUDED.pairwise_count,
  composite_score = EXCLUDED.composite_score,
  final_score = EXCLUDED.final_score,
  display_state = EXCLUDED.display_state,
  vibe_conflict = EXCLUDED.vibe_conflict,
  quality_conflict = EXCLUDED.quality_conflict,
  conflict_note = EXCLUDED.conflict_note,
  velocity_score = EXCLUDED.velocity_score,
  updated_at = now();

-- name: GetVenueStampAggregate :one
SELECT
  place_id::text AS place_id,
  stamp_count,
  weighted_stamp_sum,
  sentiment_score,
  quality_score,
  pairwise_strength,
  pairwise_count,
  composite_score,
  final_score,
  display_state,
  vibe_conflict,
  quality_conflict,
  conflict_note,
  velocity_score,
  updated_at
FROM venue_stamp_aggregates
WHERE place_id = sqlc.arg(place_id)::uuid;

-- name: DeleteVenueTagAggregates :exec
DELETE FROM venue_tag_aggregates
WHERE place_id = sqlc.arg(place_id)::uuid;

-- name: UpsertVenueTagAggregate :exec
INSERT INTO venue_tag_aggregates (
  place_id, tag_type, tag_slug, weighted_count, raw_count, percentage
)
VALUES (
  sqlc.arg(place_id)::uuid,
  sqlc.arg(tag_type),
  sqlc.arg(tag_slug),
  sqlc.arg(weighted_count),
  sqlc.arg(raw_count),
  sqlc.arg(percentage)
)
ON CONFLICT (place_id, tag_type, tag_slug) DO UPDATE
SET
  weighted_count = EXCLUDED.weighted_count,
  raw_count = EXCLUDED.raw_count,
  percentage = EXCLUDED.percentage,
  updated_at = now();

-- name: ListVenueTagAggregates :many
SELECT
  place_id::text AS place_id,
  tag_type,
  tag_slug,
  weighted_count,
  raw_count,
  percentage
FROM venue_tag_aggregates
WHERE place_id = sqlc.arg(place_id)::uuid
ORDER BY tag_type, percentage DESC, tag_slug;

-- name: ListPlaceTagVotes :many
SELECT
  st.tag_type,
  st.tag_slug,
  st.is_custom,
  s.weight
FROM stamp_tags st
JOIN stamps s ON s.id = st.stamp_id
WHERE s.place_id = sqlc.arg(place_id)::uuid
  AND s.is_active = true;

-- name: ListAllActivePlaceIDsWithStamps :many
SELECT DISTINCT place_id::text AS place_id
FROM stamps
WHERE is_active = true;

-- name: InsertVenueScoreHistory :exec
INSERT INTO venue_score_history (
  place_id,
  snapshot_date,
  stamp_count,
  weighted_stamp_sum,
  sentiment_score,
  quality_score,
  pairwise_strength,
  composite_score,
  final_score,
  display_state
)
SELECT
  place_id,
  CURRENT_DATE,
  stamp_count,
  weighted_stamp_sum,
  sentiment_score,
  quality_score,
  pairwise_strength,
  composite_score,
  final_score,
  display_state
FROM venue_stamp_aggregates
ON CONFLICT (place_id, snapshot_date) DO UPDATE
SET
  stamp_count = EXCLUDED.stamp_count,
  weighted_stamp_sum = EXCLUDED.weighted_stamp_sum,
  sentiment_score = EXCLUDED.sentiment_score,
  quality_score = EXCLUDED.quality_score,
  pairwise_strength = EXCLUDED.pairwise_strength,
  composite_score = EXCLUDED.composite_score,
  final_score = EXCLUDED.final_score,
  display_state = EXCLUDED.display_state;

-- name: InsertStampPhoto :one
INSERT INTO stamp_photos (stamp_id, storage_path, label, sort_order)
VALUES (
  sqlc.arg(stamp_id)::uuid,
  sqlc.arg(storage_path),
  sqlc.arg(label),
  sqlc.arg(sort_order)
)
RETURNING id::text AS id, stamp_id::text AS stamp_id, storage_path, label, sort_order;

-- name: ListStampPhotos :many
SELECT id::text AS id, stamp_id::text AS stamp_id, storage_path, label, sort_order
FROM stamp_photos
WHERE stamp_id = sqlc.arg(stamp_id)::uuid
ORDER BY sort_order, created_at;

-- name: CountRecentPairwisePair :one
SELECT COUNT(*)::integer AS count
FROM pairwise_comparisons
WHERE user_id = sqlc.arg(user_id)::uuid
  AND (
    (place_a_id = sqlc.arg(place_a_id)::uuid AND place_b_id = sqlc.arg(place_b_id)::uuid)
    OR (place_a_id = sqlc.arg(place_b_id)::uuid AND place_b_id = sqlc.arg(place_a_id)::uuid)
  )
  AND created_at > now() - interval '30 days';

-- name: ListFeedStamps :many
SELECT
  s.id::text AS stamp_id,
  s.user_id::text AS user_id,
  up.first_name,
  up.last_name,
  up.username,
  up.avatar_url,
  p.id::text AS place_id,
  p.name AS place_name,
  p.cover_image_url AS place_image_url,
  s.venue_category,
  s.verdict,
  pe.band,
  pe.score AS personal_score,
  s.note,
  COALESCE(ph.storage_path, '') AS photo_storage_path,
  s.created_at
FROM stamps s
JOIN places p ON p.id = s.place_id AND p.is_active = true
JOIN user_profiles up ON up.user_id = s.user_id
LEFT JOIN user_pool_entries pe
  ON pe.user_id = s.user_id AND pe.place_id = s.place_id
LEFT JOIN LATERAL (
  SELECT sp.storage_path
  FROM stamp_photos sp
  WHERE sp.stamp_id = s.id
  ORDER BY sp.sort_order, sp.created_at
  LIMIT 1
) ph ON true
WHERE s.is_active = true
  AND (
    s.user_id = sqlc.arg(viewer_id)::uuid
    OR s.user_id IN (
      SELECT uf.following_user_id
      FROM user_follows uf
      WHERE uf.follower_user_id = sqlc.arg(viewer_id)::uuid
    )
  )
ORDER BY s.created_at DESC
LIMIT sqlc.arg(limit_val)::int
OFFSET sqlc.arg(offset_val)::int;
