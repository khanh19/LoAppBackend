ALTER TABLE public.place_list_entries
    RENAME COLUMN rank TO stop_order;

ALTER TABLE public.place_list_entries
    RENAME CONSTRAINT place_list_entries_rank_valid
    TO place_list_entries_stop_order_valid;

ALTER INDEX public.place_list_entries_list_active_rank_idx
    RENAME TO place_list_entries_list_active_stop_order_idx;
