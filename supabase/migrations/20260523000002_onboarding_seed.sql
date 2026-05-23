BEGIN;

SET search_path = public;

INSERT INTO public.cities (slug, name, country_code, cover_image_url, sort_order)
VALUES
    (
        'hanoi',
        'Hanoi',
        'VN',
        'https://xofbxjbrzpsocnjdsmmk.supabase.co/storage/v1/object/sign/LoAppBucket/Onboarding/Hanoi%201.jpg?token=eyJraWQiOiJzdG9yYWdlLXVybC1zaWduaW5nLWtleV9hNzgwYjgxMC05NDMyLTRmOGYtYjJjMi1iM2RhMTAxMGNjNDUiLCJhbGciOiJIUzI1NiJ9.eyJ1cmwiOiJMb0FwcEJ1Y2tldC9PbmJvYXJkaW5nL0hhbm9pIDEuanBnIiwiaWF0IjoxNzc5NTc1MzU4LCJleHAiOjE4MTExMTEzNTh9.fHya7ub2CxeYGbwYN6E7iaEGoAXNXNnhrOHuk-Letyk',
        1
    ),
    (
        'hcmc',
        'Ho Chi Minh City',
        'VN',
        'https://xofbxjbrzpsocnjdsmmk.supabase.co/storage/v1/object/sign/LoAppBucket/Onboarding/HCM.jpg?token=eyJraWQiOiJzdG9yYWdlLXVybC1zaWduaW5nLWtleV9hNzgwYjgxMC05NDMyLTRmOGYtYjJjMi1iM2RhMTAxMGNjNDUiLCJhbGciOiJIUzI1NiJ9.eyJ1cmwiOiJMb0FwcEJ1Y2tldC9PbmJvYXJkaW5nL0hDTS5qcGciLCJpYXQiOjE3Nzk1NzU0MjQsImV4cCI6MTgxMTExMTQyNH0.nCqDC1xTmBfxIInnBanbLY4mu3Rtg2sgWdrwVfhzd-g',
        2
    ),
    (
        'danang',
        'Da Nang City',
        'VN',
        'https://xofbxjbrzpsocnjdsmmk.supabase.co/storage/v1/object/sign/LoAppBucket/Onboarding/danang.jpeg?token=eyJraWQiOiJzdG9yYWdlLXVybC1zaWduaW5nLWtleV9hNzgwYjgxMC05NDMyLTRmOGYtYjJjMi1iM2RhMTAxMGNjNDUiLCJhbGciOiJIUzI1NiJ9.eyJ1cmwiOiJMb0FwcEJ1Y2tldC9PbmJvYXJkaW5nL2RhbmFuZy5qcGVnIiwiaWF0IjoxNzc5NTc1OTc1LCJleHAiOjE4MTExMTE5NzV9.wgRSSsn_OrLrvzeuciw42dK3K0jWgxdEgGVQfTnpZdM',
        3
    )
ON CONFLICT (slug) DO NOTHING;

INSERT INTO public.purpose_categories (slug, name, icon_name, sort_order)
VALUES
    ('restaurants', 'Restaurants', 'restaurant-outline', 1),
    ('cafes', 'Cafes', 'cafe-outline', 2),
    ('bars', 'Bars', 'wine-outline', 3),
    ('entertainment', 'Entertainment', 'sparkles-outline', 4),
    ('itineraries', 'Itineraries', 'map-outline', 5),
    ('visits', 'Visits', 'eye-outline', 6)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO public.vibes (slug, name, description, sort_order)
VALUES
    (
        'chill',
        'Chill & Relaxed',
        'Slow pace, comfortable spots, easy conversations',
        1
    ),
    (
        'trendy',
        'Trendy & Aesthetic',
        'Beautiful spaces, popular spots, worth sharing',
        2
    ),
    (
        'social',
        'Social & Lively',
        'Crowded energy, group vibes, buzzing atmosphere',
        3
    ),
    (
        'unique',
        'Unique & Hidden',
        'Unexpected finds, only local knows!',
        4
    )
ON CONFLICT (slug) DO NOTHING;

WITH city_ids AS (
    SELECT slug, id FROM public.cities WHERE slug IN ('hanoi', 'hcmc', 'hoian')
),
category_ids AS (
    SELECT slug, id FROM public.purpose_categories
    WHERE slug IN ('restaurants', 'cafes', 'bars')
)
INSERT INTO public.places (
    city_id,
    source,
    name,
    slug,
    neighborhood,
    price_level,
    rating_cached,
    cover_image_url,
    tags
)
SELECT
    c.id,
    'curated',
    v.name,
    v.slug,
    v.neighborhood,
    v.price_level,
    v.rating_cached,
    v.cover_image_url,
    v.tags
FROM (
    VALUES
        (
            'hanoi',
            'pizzas-4ps',
            'Pizza''s 4Ps',
            'Hoan Kiem District',
            3,
            4.7,
            'https://images.unsplash.com/photo-1555396273-367ea4eb4db5?w=800&q=80',
            ARRAY['Minimal', 'Quiet, chill', 'Aesthetic Decoration']::text[]
        ),
        (
            'hanoi',
            'a-choens-grill',
            'A Choen''s Grill',
            'Hoan Kiem District',
            3,
            4.7,
            'https://images.unsplash.com/photo-1544025162-d76694265947?w=800&q=80',
            ARRAY['Spacious', 'Noisy, Hyped', 'Street Style Decoration']::text[]
        ),
        (
            'hcmc',
            'banh-mi-huynh-hoa',
            'Banh Mi Huynh Hoa',
            'Ben Thanh',
            2,
            4.7,
            'https://images.unsplash.com/photo-1627308595229-7830a5c91f9f?w=800&q=80',
            ARRAY['Minimal', 'Busy, hustling', 'Top Attraction']::text[]
        ),
        (
            'danang',
            'the-workshop',
            'The Workshop Coffee',
            'Old Town',
            1,
            4.6,
            'https://images.unsplash.com/photo-1501339847302-ac426a4a7cbb?w=800&q=80',
            ARRAY['Industrial', 'Quiet, chill', 'Aesthetic Decoration']::text[]
        ),
        (
            'hcmc',
            'noir-bar',
            'Noir Speakeasy',
            'Binh Thanh District',
            2,
            4.8,
            'https://images.unsplash.com/photo-1470337458703-46ad1756a187?w=800&q=80',
            ARRAY['Dark & Moody', 'Lively', 'Cocktail Focused']::text[]
        )
) AS v(city_slug, slug, name, neighborhood, price_level, rating_cached, cover_image_url, tags)
JOIN city_ids c ON c.slug = v.city_slug
ON CONFLICT (city_id, slug) DO NOTHING;

INSERT INTO public.place_categories (place_id, category_id)
SELECT p.id, pc.id
FROM public.places p
JOIN public.purpose_categories pc ON pc.slug = CASE p.slug
    WHEN 'the-workshop' THEN 'cafes'
    WHEN 'noir-bar' THEN 'bars'
    ELSE 'restaurants'
END
WHERE p.source = 'curated'
ON CONFLICT DO NOTHING;

COMMIT;
