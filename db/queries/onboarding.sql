-- name: ListActiveCities :many
SELECT
  id::text AS id,
  slug,
  name,
  country_code,
  cover_image_url,
  sort_order
FROM cities
WHERE is_active = true
ORDER BY sort_order, name;

-- name: ListActivePurposeCategories :many
SELECT
  id::text AS id,
  slug,
  name,
  icon_name,
  sort_order
FROM purpose_categories
WHERE is_active = true
ORDER BY sort_order, name;

-- name: ListActiveVibes :many
SELECT
  id::text AS id,
  slug,
  name,
  description,
  sort_order
FROM vibes
WHERE is_active = true
ORDER BY sort_order, name;

-- name: ListCuratedVenuesByCity :many
SELECT
  p.id::text AS id,
  p.slug,
  p.name,
  COALESCE(pc.name, 'Restaurant') AS category,
  p.rating_cached,
  p.price_level,
  p.neighborhood,
  p.cover_image_url,
  p.tags
FROM places p
LEFT JOIN LATERAL (
  SELECT c.name
  FROM place_categories pc
  JOIN purpose_categories c ON c.id = pc.category_id
  WHERE pc.place_id = p.id
  ORDER BY c.sort_order
  LIMIT 1
) pc ON true
WHERE p.city_id = sqlc.arg(city_id)::uuid
  AND p.source = 'curated'
  AND p.is_active = true
ORDER BY p.name;

-- name: DeleteUserExploreCities :exec
DELETE FROM user_explore_cities
WHERE user_id = sqlc.arg(user_id)::uuid;

-- name: InsertUserExploreCity :exec
INSERT INTO user_explore_cities (user_id, city_id, priority)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.arg(city_id)::uuid,
  sqlc.arg(priority)::integer
);

-- name: DeleteUserPurposeCategories :exec
DELETE FROM user_purpose_categories
WHERE user_id = sqlc.arg(user_id)::uuid;

-- name: InsertUserPurposeCategory :exec
INSERT INTO user_purpose_categories (user_id, category_id)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.arg(category_id)::uuid
);

-- name: DeleteUserVibes :exec
DELETE FROM user_vibes
WHERE user_id = sqlc.arg(user_id)::uuid;

-- name: InsertUserVibe :exec
INSERT INTO user_vibes (user_id, vibe_id, weight)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.arg(vibe_id)::uuid,
  sqlc.arg(weight)::numeric
);

-- name: DeleteUserLikedPlacesBySource :exec
DELETE FROM user_liked_places
WHERE user_id = sqlc.arg(user_id)::uuid
  AND source = sqlc.arg(source);

-- name: InsertUserLikedPlace :exec
INSERT INTO user_liked_places (user_id, place_id, source)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.arg(place_id)::uuid,
  sqlc.arg(source)
);

-- name: CompleteUserOnboarding :one
UPDATE users
SET
  onboarding_status = 'completed',
  onboarding_completed_at = now()
WHERE id = sqlc.arg(user_id)::uuid
RETURNING onboarding_status::text AS onboarding_status;
