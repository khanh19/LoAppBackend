-- name: ListUpcomingEvents :many
SELECT
  e.id::text AS id,
  e.event_name,
  e.event_url,
  e.venue_name,
  e.venue_url,
  e.schedule_text,
  e.music_text,
  e.description,
  e.poster_image_url,
  e.imported_at,
  c.slug AS city_slug,
  c.name AS city_name,
  es.slug AS source_slug,
  es.name AS source_name
FROM external_events e
JOIN event_sources es ON es.id = e.source_id AND es.is_active = true
LEFT JOIN cities c ON c.id = e.city_id AND c.is_active = true
WHERE e.poster_image_url IS NOT NULL
  AND (sqlc.narg(city_slug)::text IS NULL OR c.slug = sqlc.narg(city_slug))
ORDER BY e.imported_at DESC, e.event_name ASC
LIMIT sqlc.arg(limit_val)::int;

-- name: ListExternalEvents :many
SELECT
  e.id::text AS id,
  e.event_name,
  e.event_url,
  e.venue_name,
  e.venue_url,
  e.schedule_text,
  e.music_text,
  e.description,
  e.poster_image_url,
  e.imported_at,
  c.slug AS city_slug,
  c.name AS city_name,
  es.slug AS source_slug,
  es.name AS source_name
FROM external_events e
JOIN event_sources es ON es.id = e.source_id AND es.is_active = true
LEFT JOIN cities c ON c.id = e.city_id AND c.is_active = true
WHERE (sqlc.narg(city_slug)::text IS NULL OR c.slug = sqlc.narg(city_slug))
ORDER BY e.imported_at DESC, e.event_name ASC
LIMIT sqlc.arg(limit_val)::int
OFFSET sqlc.arg(offset_val)::int;

-- name: CountExternalEvents :one
SELECT COUNT(*)::bigint AS count
FROM external_events e
JOIN event_sources es ON es.id = e.source_id AND es.is_active = true
LEFT JOIN cities c ON c.id = e.city_id AND c.is_active = true
WHERE (sqlc.narg(city_slug)::text IS NULL OR c.slug = sqlc.narg(city_slug));
