-- Add Visitor companion option used by stamp flow step 1.
INSERT INTO public.tag_taxonomy (slug, tag_type, label, sort_order)
VALUES ('visitor', 'companion', 'Visitor', 5)
ON CONFLICT DO NOTHING;
