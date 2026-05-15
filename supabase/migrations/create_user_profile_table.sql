create extension if not exists "pgcrypto";
create extension if not exists "citext";

create table if not exists user_profiles (
    user_id uuid primary key references users(id) on delete cascade,

    first_name text not null,
    last_name text not null,

    username citext not null unique,

    phone_e164 text,
    phone_country_code text,
    phone_national_number text,

    date_of_birth date,

    avatar_object_key text,
    avatar_url text,

    bio text,

    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    constraint user_profiles_username_length_check
        check (char_length(username::text) between 3 and 20),

    constraint user_profiles_username_format_check
        check (username::text ~ '^[a-zA-Z0-9_\.]+$'),

    constraint user_profiles_first_name_length_check
        check (char_length(first_name) between 1 and 80),

    constraint user_profiles_last_name_length_check
        check (char_length(last_name) between 1 and 80),

    constraint user_profiles_phone_e164_format_check
        check (phone_e164 is null or phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),

    constraint user_profiles_dob_check
        check (date_of_birth is null or date_of_birth <= current_date)
);

create index if not exists idx_user_profiles_username
    on user_profiles(username);

create index if not exists idx_user_profiles_phone_e164
    on user_profiles(phone_e164)
    where phone_e164 is not null;