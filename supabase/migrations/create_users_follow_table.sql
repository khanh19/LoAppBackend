create table if not exists user_follows (
    follower_user_id uuid not null references users(id) on delete cascade,
    following_user_id uuid not null references users(id) on delete cascade,

    created_at timestamptz not null default now(),

    primary key (follower_user_id, following_user_id),

    constraint user_follows_no_self_follow_check
        check (follower_user_id <> following_user_id)
);

-- Fast lookup: get all followers of a user
create index if not exists idx_user_follows_following
    on user_follows(following_user_id);