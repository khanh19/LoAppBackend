-- Stamp & ranking system schema.

BEGIN;

SET search_path = public;

-- ---------------------------------------------------------------------------
-- ranking_config
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.ranking_config (
  key TEXT PRIMARY KEY,
  value JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO public.ranking_config (key, value) VALUES
  ('bands', '{
    "must-repeat": {"lo": 7.0, "hi": 10.0},
    "good-pick": {"lo": 4.5, "hi": 7.0},
    "mid": {"lo": 2.5, "hi": 4.5},
    "not-for-me": {"lo": 0.0, "hi": 2.5}
  }'::jsonb),
  ('verdict_values', '{
    "must-repeat": 9.0,
    "good-pick": 6.5,
    "mid": 4.0,
    "not-for-me": 1.5
  }'::jsonb),
  ('quality_values', '{
    "outstanding": 9.5,
    "good": 7.0,
    "okay": 4.5,
    "not-good": 1.5
  }'::jsonb),
  ('verification_weights', '{
    "booked": 1.0,
    "gps": 0.8,
    "repeat": 0.5,
    "first_time": 0.3
  }'::jsonb),
  ('composite_weights', '{
    "sentiment": 0.50,
    "quality": 0.35,
    "pairwise": 0.15,
    "pairwise_min_comparisons": 5
  }'::jsonb),
  ('shrinkage', '{
    "m": 5,
    "C": {"restaurant": 6.5, "cafe": 6.5, "bar": 6.5, "club": 6.5, "other": 6.5}
  }'::jsonb),
  ('placement_evidence', '{
    "s_min": 0.40,
    "s_max": 1.00,
    "k_min": 8
  }'::jsonb),
  ('comparability_weights', '{
    "bisection": 0.40,
    "geo": 0.20,
    "price": 0.15,
    "vibe": 0.15,
    "recency": 0.10
  }'::jsonb),
  ('gps_radius_meters', '200'::jsonb),
  ('display_thresholds', '{
    "early": 5,
    "full": 15
  }'::jsonb)
ON CONFLICT (key) DO NOTHING;

-- ---------------------------------------------------------------------------
-- tag_taxonomy
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.tag_taxonomy (
  slug TEXT NOT NULL,
  tag_type TEXT NOT NULL,
  label TEXT NOT NULL,
  venue_category TEXT,
  is_active BOOLEAN NOT NULL DEFAULT true,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (slug, tag_type),
  CONSTRAINT tag_taxonomy_type_valid CHECK (tag_type IN ('companion', 'vibe')),
  CONSTRAINT tag_taxonomy_category_valid CHECK (
    venue_category IS NULL
    OR venue_category IN ('restaurant', 'cafe', 'bar', 'club', 'other')
  )
);

INSERT INTO public.tag_taxonomy (slug, tag_type, label, sort_order) VALUES
  ('family', 'companion', 'Family', 1),
  ('squad', 'companion', 'Squad', 2),
  ('my-date', 'companion', 'My Date', 3),
  ('solo', 'companion', 'Solo', 4),
  ('cozy', 'vibe', 'Cozy', 1),
  ('romantic', 'vibe', 'Romantic', 2),
  ('lively', 'vibe', 'Lively', 3),
  ('chill', 'vibe', 'Chill', 4),
  ('trendy', 'vibe', 'Trendy', 5),
  ('luxury', 'vibe', 'Luxury', 6),
  ('late-night', 'vibe', 'Late-night', 7),
  ('long-convo', 'vibe', 'Great for long-convo', 8)
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- stamps
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.stamps (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
  place_id UUID NOT NULL REFERENCES public.places(id) ON DELETE CASCADE,
  venue_category TEXT NOT NULL,
  verdict TEXT NOT NULL,
  quality TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  visit_count INTEGER NOT NULL DEFAULT 1,
  verification_level TEXT NOT NULL DEFAULT 'first_time',
  weight NUMERIC(4, 3) NOT NULL DEFAULT 0.300,
  device_latitude NUMERIC(9, 6),
  device_longitude NUMERIC(9, 6),
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT stamps_verdict_valid CHECK (
    verdict IN ('must-repeat', 'good-pick', 'mid', 'not-for-me')
  ),
  CONSTRAINT stamps_quality_valid CHECK (
    quality IN ('outstanding', 'good', 'okay', 'not-good')
  ),
  CONSTRAINT stamps_verification_valid CHECK (
    verification_level IN ('booked', 'gps', 'repeat', 'first_time')
  ),
  CONSTRAINT stamps_category_valid CHECK (
    venue_category IN ('restaurant', 'cafe', 'bar', 'club', 'other')
  ),
  CONSTRAINT stamps_visit_count_valid CHECK (visit_count > 0),
  CONSTRAINT stamps_weight_valid CHECK (weight > 0 AND weight <= 1)
);

CREATE UNIQUE INDEX IF NOT EXISTS stamps_user_place_active_uidx
  ON public.stamps (user_id, place_id)
  WHERE is_active = true;

CREATE INDEX IF NOT EXISTS stamps_place_active_idx
  ON public.stamps (place_id)
  WHERE is_active = true;

CREATE INDEX IF NOT EXISTS stamps_user_category_idx
  ON public.stamps (user_id, venue_category)
  WHERE is_active = true;

DROP TRIGGER IF EXISTS stamps_set_updated_at ON public.stamps;
CREATE TRIGGER stamps_set_updated_at
BEFORE UPDATE ON public.stamps
FOR EACH ROW
EXECUTE FUNCTION public.set_updated_at();

-- ---------------------------------------------------------------------------
-- stamp_tags
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.stamp_tags (
  stamp_id UUID NOT NULL REFERENCES public.stamps(id) ON DELETE CASCADE,
  tag_type TEXT NOT NULL,
  tag_slug TEXT NOT NULL,
  position INTEGER NOT NULL DEFAULT 0,
  is_custom BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (stamp_id, tag_type, tag_slug),
  CONSTRAINT stamp_tags_type_valid CHECK (tag_type IN ('companion', 'vibe')),
  CONSTRAINT stamp_tags_position_valid CHECK (position >= 0)
);

CREATE INDEX IF NOT EXISTS stamp_tags_slug_type_idx
  ON public.stamp_tags (tag_type, tag_slug);

-- ---------------------------------------------------------------------------
-- stamp_photos
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.stamp_photos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  stamp_id UUID NOT NULL REFERENCES public.stamps(id) ON DELETE CASCADE,
  storage_path TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT 'other',
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT stamp_photos_label_valid CHECK (
    label IN ('food_drink', 'space_vibe', 'other')
  )
);

CREATE INDEX IF NOT EXISTS stamp_photos_stamp_idx
  ON public.stamp_photos (stamp_id, sort_order);

-- ---------------------------------------------------------------------------
-- user_pool_entries
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.user_pool_entries (
  user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
  place_id UUID NOT NULL REFERENCES public.places(id) ON DELETE CASCADE,
  venue_category TEXT NOT NULL,
  band TEXT NOT NULL,
  rank_key NUMERIC NOT NULL,
  score NUMERIC(4, 1) NOT NULL,
  placement_confidence NUMERIC(4, 3) NOT NULL DEFAULT 0,
  comparisons_count INTEGER NOT NULL DEFAULT 0,
  contradiction_count INTEGER NOT NULL DEFAULT 0,
  is_provisional BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, place_id),
  CONSTRAINT user_pool_band_valid CHECK (
    band IN ('must-repeat', 'good-pick', 'mid', 'not-for-me')
  ),
  CONSTRAINT user_pool_category_valid CHECK (
    venue_category IN ('restaurant', 'cafe', 'bar', 'club', 'other')
  ),
  CONSTRAINT user_pool_score_valid CHECK (score >= 0 AND score <= 10),
  CONSTRAINT user_pool_confidence_valid CHECK (
    placement_confidence >= 0 AND placement_confidence <= 1
  )
);

CREATE INDEX IF NOT EXISTS user_pool_user_band_idx
  ON public.user_pool_entries (user_id, venue_category, band, rank_key);

DROP TRIGGER IF EXISTS user_pool_entries_set_updated_at ON public.user_pool_entries;
CREATE TRIGGER user_pool_entries_set_updated_at
BEFORE UPDATE ON public.user_pool_entries
FOR EACH ROW
EXECUTE FUNCTION public.set_updated_at();

-- ---------------------------------------------------------------------------
-- pairwise_comparisons (append-only)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.pairwise_comparisons (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
  stamp_id UUID REFERENCES public.stamps(id) ON DELETE SET NULL,
  place_a_id UUID NOT NULL REFERENCES public.places(id) ON DELETE CASCADE,
  place_b_id UUID NOT NULL REFERENCES public.places(id) ON DELETE CASCADE,
  winner_place_id UUID REFERENCES public.places(id) ON DELETE SET NULL,
  venue_category TEXT NOT NULL,
  band TEXT NOT NULL,
  tier INTEGER NOT NULL DEFAULT 1,
  comparability NUMERIC(5, 4) NOT NULL DEFAULT 0,
  surface TEXT NOT NULL DEFAULT 'stamp_flow',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT pairwise_tier_valid CHECK (tier BETWEEN 1 AND 5),
  CONSTRAINT pairwise_surface_valid CHECK (
    surface IN ('stamp_flow', 'standalone', 'calibration')
  ),
  CONSTRAINT pairwise_distinct_places CHECK (place_a_id <> place_b_id)
);

CREATE INDEX IF NOT EXISTS pairwise_user_created_idx
  ON public.pairwise_comparisons (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS pairwise_places_idx
  ON public.pairwise_comparisons (place_a_id, place_b_id);

-- ---------------------------------------------------------------------------
-- placement_sessions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.placement_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
  stamp_id UUID NOT NULL REFERENCES public.stamps(id) ON DELETE CASCADE,
  place_id UUID NOT NULL REFERENCES public.places(id) ON DELETE CASCADE,
  venue_category TEXT NOT NULL,
  band TEXT NOT NULL,
  lo_index INTEGER NOT NULL DEFAULT 1,
  hi_index INTEGER NOT NULL DEFAULT 1,
  comparisons_done INTEGER NOT NULL DEFAULT 0,
  is_complete BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS placement_sessions_user_active_idx
  ON public.placement_sessions (user_id, is_complete)
  WHERE is_complete = false;

DROP TRIGGER IF EXISTS placement_sessions_set_updated_at ON public.placement_sessions;
CREATE TRIGGER placement_sessions_set_updated_at
BEFORE UPDATE ON public.placement_sessions
FOR EACH ROW
EXECUTE FUNCTION public.set_updated_at();

-- ---------------------------------------------------------------------------
-- user_place_familiarity
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.user_place_familiarity (
  user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
  place_id UUID NOT NULL REFERENCES public.places(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'unknown',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, place_id),
  CONSTRAINT familiarity_status_valid CHECK (
    status IN ('been', 'not_been', 'unknown')
  )
);

DROP TRIGGER IF EXISTS user_place_familiarity_set_updated_at ON public.user_place_familiarity;
CREATE TRIGGER user_place_familiarity_set_updated_at
BEFORE UPDATE ON public.user_place_familiarity
FOR EACH ROW
EXECUTE FUNCTION public.set_updated_at();

-- ---------------------------------------------------------------------------
-- venue_stamp_aggregates
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.venue_stamp_aggregates (
  place_id UUID PRIMARY KEY REFERENCES public.places(id) ON DELETE CASCADE,
  stamp_count INTEGER NOT NULL DEFAULT 0,
  weighted_stamp_sum NUMERIC(12, 4) NOT NULL DEFAULT 0,
  sentiment_score NUMERIC(4, 2),
  quality_score NUMERIC(4, 2),
  pairwise_strength NUMERIC(4, 2),
  pairwise_count INTEGER NOT NULL DEFAULT 0,
  composite_score NUMERIC(4, 2),
  final_score NUMERIC(4, 1),
  display_state TEXT NOT NULL DEFAULT 'new',
  vibe_conflict BOOLEAN NOT NULL DEFAULT false,
  quality_conflict BOOLEAN NOT NULL DEFAULT false,
  conflict_note TEXT,
  velocity_score NUMERIC(8, 4) NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT aggregates_display_state_valid CHECK (
    display_state IN ('new', 'early', 'full')
  )
);

DROP TRIGGER IF EXISTS venue_stamp_aggregates_set_updated_at ON public.venue_stamp_aggregates;
CREATE TRIGGER venue_stamp_aggregates_set_updated_at
BEFORE UPDATE ON public.venue_stamp_aggregates
FOR EACH ROW
EXECUTE FUNCTION public.set_updated_at();

-- ---------------------------------------------------------------------------
-- venue_tag_aggregates
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.venue_tag_aggregates (
  place_id UUID NOT NULL REFERENCES public.places(id) ON DELETE CASCADE,
  tag_type TEXT NOT NULL,
  tag_slug TEXT NOT NULL,
  weighted_count NUMERIC(12, 4) NOT NULL DEFAULT 0,
  raw_count INTEGER NOT NULL DEFAULT 0,
  percentage NUMERIC(5, 2) NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (place_id, tag_type, tag_slug),
  CONSTRAINT venue_tag_type_valid CHECK (tag_type IN ('companion', 'vibe'))
);

-- ---------------------------------------------------------------------------
-- venue_score_history
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.venue_score_history (
  id BIGSERIAL PRIMARY KEY,
  place_id UUID NOT NULL REFERENCES public.places(id) ON DELETE CASCADE,
  snapshot_date DATE NOT NULL DEFAULT CURRENT_DATE,
  stamp_count INTEGER NOT NULL DEFAULT 0,
  weighted_stamp_sum NUMERIC(12, 4) NOT NULL DEFAULT 0,
  sentiment_score NUMERIC(4, 2),
  quality_score NUMERIC(4, 2),
  pairwise_strength NUMERIC(4, 2),
  composite_score NUMERIC(4, 2),
  final_score NUMERIC(4, 1),
  display_state TEXT NOT NULL DEFAULT 'new',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (place_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS venue_score_history_place_date_idx
  ON public.venue_score_history (place_id, snapshot_date DESC);

COMMIT;
