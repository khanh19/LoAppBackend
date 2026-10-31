-- The onboarding seed venues were labelled with pre-2025 districts. Relabel
-- them with the wards that replaced those districts.

UPDATE public.places p
SET ward       = v.ward,
    updated_at = now()
FROM (
    VALUES
        ('pizzas-4ps', 'Phường Hoàn Kiếm'),
        ('a-choens-grill', 'Phường Hoàn Kiếm'),
        ('banh-mi-huynh-hoa', 'Phường Bến Thành'),
        ('the-workshop', 'Phường Hải Châu'),
        ('noir-bar', 'Phường Bình Thạnh')
) AS v(slug, ward)
WHERE p.slug = v.slug
  AND p.source = 'curated';
