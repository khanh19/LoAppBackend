# Lo App Backend

Encore.go backend for **Lo** — a place-discovery and day-planning app focused on Vietnam (Hanoi, Ho Chi Minh City, Da Nang). Clients search places via Google Places, build day plans, browse curated lists, complete onboarding preferences, and discover nightlife events.

**Encore app ID:** `loappbackend-a9f2`

---

## Table of contents

1. [Tech stack](#tech-stack)
2. [System architecture](#system-architecture)
3. [Repository layout](#repository-layout)
4. [Services and endpoints](#services-and-endpoints)
5. [Google Places API integration](#google-places-api-integration)
6. [Authentication (Auth0)](#authentication-auth0)
7. [Data model](#data-model)
8. [Configuration and secrets](#configuration-and-secrets)
9. [Local development](#local-development)
10. [Testing](#testing)
11. [Deployment](#deployment)

---

## Tech stack

| Layer | Choice | Notes |
|-------|--------|-------|
| Framework | [Encore.go](https://encore.dev) v1.52.1 | Type-safe APIs, auth handler, cron, secrets, tracing |
| Language | Go 1.26 | Module `encore.app` |
| Database | Supabase Postgres | External DB via `pgx/v5` — **not** Encore-managed `sqldb` |
| SQL tooling | [sqlc](https://sqlc.dev) | Typed queries from `db/queries/*.sql` → `internal/dbgen/` |
| Identity | [Auth0](https://auth0.com) | OIDC ID tokens + passwordless email OTP |
| Places data | [Google Places API (New)](https://developers.google.com/maps/documentation/places/web-service) | Autocomplete, Details, Text Search, Photo Media |
| Events data | Vietnam Nightlife scrape | Imported via `replace_external_events()` stored procedure |

### Why an external database?

This app deliberately uses Supabase Postgres instead of Encore-provisioned databases:

- Schema and RLS live in `supabase/migrations/` and are applied with the Supabase CLI (`supabase db push`).
- There is no `encore db shell` / `encore db reset` for app data — use Supabase Studio or `psql` against the connection string.
- All services share one lazily-initialized `pgxpool` from [`pkg/externaldb`](pkg/externaldb/externaldb.go).

---

## System architecture

```mermaid
flowchart TB
  subgraph clients [Clients]
    Mobile[Mobile App Expo]
    Web[Web / curl]
  end

  subgraph encore [Encore Backend]
    Gateway[API Gateway]
    AuthHandler[auth.AuthHandler]
    AuthSvc[auth]
    UsersSvc[users]
    OnboardingSvc[onboarding]
    PlacesSvc[places]
    ListsSvc[lists]
    EventsSvc[events]
    DebugSvc[debug]
    CronJob["cron: sync-place-lists"]
  end

  subgraph data [Data]
    Pool[externaldb pgxpool]
    Supabase[(Supabase Postgres)]
  end

  subgraph external [External providers]
    Auth0[Auth0 OIDC]
    Google[Google Places API New]
    Scraper[vietnamnightlife scraper]
  end

  Mobile --> Gateway
  Web --> Gateway
  Gateway --> AuthHandler
  AuthHandler --> AuthSvc
  Gateway --> AuthSvc
  Gateway --> UsersSvc
  Gateway --> OnboardingSvc
  Gateway --> PlacesSvc
  Gateway --> ListsSvc
  Gateway --> EventsSvc
  Gateway --> DebugSvc

  AuthSvc -->|"UpsertFromAuth"| UsersSvc
  ListsSvc -->|"ResolvePlaceForPlan"| PlacesSvc
  CronJob --> ListsSvc

  AuthSvc --> Auth0
  PlacesSvc --> Google
  ListsSvc --> Google

  AuthSvc --> Pool
  UsersSvc --> Pool
  OnboardingSvc --> Pool
  PlacesSvc --> Pool
  ListsSvc --> Pool
  EventsSvc --> Pool
  DebugSvc --> Pool
  Pool --> Supabase
  Scraper -->|"replace_external_events()"| Supabase
```

### Authenticated request lifecycle

```mermaid
sequenceDiagram
  participant C as Client
  participant G as Encore Gateway
  participant AH as AuthHandler
  participant A0 as Auth0 OIDC
  participant U as users service
  participant S as Target service
  participant DB as Supabase

  C->>G: Request + Bearer ID token
  G->>AH: AuthHandler(token)
  AH->>A0: Verify ID token JWT
  A0-->>AH: Claims sub email picture
  AH->>U: UpsertFromAuth private
  U->>DB: Find/create user + identity
  U-->>AH: users.id UUID
  AH-->>G: auth.UID + ProfileData
  G->>S: Handler with auth context
  S->>DB: Business queries
  S-->>C: Response
```

### Inter-service calls

Only two Encore service-to-service API calls exist; everything else shares state through Postgres:

| Caller | Callee | When |
|--------|--------|------|
| `auth` | `users.UpsertFromAuth` | Every login and every authenticated request |
| `lists` | `places.ResolvePlaceForPlan` | Creating a plan stop with `google_place_id` |

---

## Repository layout

```
lo-app-backend/
├── encore.app                 # App id: loappbackend-a9f2
├── go.mod / go.sum
├── sqlc.yaml                  # sqlc → internal/dbgen
├── .secrets.local.cue         # Local secret overrides
│
├── auth/                      # Auth0 login, OTP, auth handler
├── users/                     # User upsert + profiles
├── onboarding/                # Cities, categories, vibes, complete
├── places/                    # Place cache + Google Autocomplete/Details
├── lists/                     # Curated lists, user plans, seed sync
├── events/                    # External nightlife events (read-only)
├── debug/                     # DB connectivity check
├── hello/                     # Stale encore.gen.go only — not a real service
│
├── pkg/externaldb/            # Shared Supabase pgx pool
├── db/queries/                # Hand-written SQL for sqlc
├── internal/dbgen/            # Generated typed query code
├── supabase/migrations/       # Schema source of truth
└── scripts/create_plans.sh    # Seed demo plans via POST /plans
```

### Per-service file convention

Most services follow the same shape:

| File | Role |
|------|------|
| `service.go` | `//encore:service` struct + `initService()` DI |
| `api.go` / `*_api.go` | `//encore:api` endpoints |
| `types.go` | Request/response DTOs |
| `repository.go` | Maps `dbgen` rows → API types |
| `dbgen_types.go` | UUID / numeric / pgtype helpers |
| `validation.go` | Input normalization |
| `google_client.go` | Google Places HTTP client (`places`, `lists` only) |

---

## Services and endpoints

Access levels:

- **public** — anyone on the internet
- **auth** — requires valid `Authorization: Bearer <Auth0 ID token>`
- **private** — only callable from other Encore services / cron (not exposed publicly)

---

### `auth` — Identity and sessions

**Purpose:** Talk to Auth0, verify ID tokens, and establish the Encore auth context. Does not store sessions server-side — clients hold the Auth0 ID token.

**Dependencies:** Auth0 OIDC + OAuth2 client (`Authenticator`), Encore config from `auth-config.cue`, private call to `users.UpsertFromAuth`.

| Access | Method | Path | Function | Description |
|--------|--------|------|----------|-------------|
| public | POST | `/auth/login` | `Login` | Returns OAuth `state` + Auth0 authorize URL |
| public | POST | `/auth/callback` | `Callback` | Exchanges auth `code` for ID token; upserts user |
| public | GET | `/auth/logout` | `Logout` | Returns Auth0 `/v2/logout` redirect URL |
| public | POST | `/auth/otp/start` | `StartEmailOTP` | Sends passwordless email OTP via Auth0 |
| public | POST | `/auth/otp/verify` | `VerifyEmailOTP` | Exchanges OTP for ID token; upserts user |
| auth | GET | `/profile` | `GetProfile` | Returns email + picture from JWT claims |
| — | — | `//encore:authhandler` | `AuthHandler` | Verifies Bearer token on every `auth` endpoint |

**Key payloads**

`LoginResponse`: `state`, `auth_code_url`

`CallbackRequest`: `code`

`CallbackResponse`: `token` (Auth0 ID token), `onboarding_status`, `profile_exists`

`OTPStartRequest`: `email` (+ optional `X-Forwarded-For` header)

`OTPVerifyRequest`: `email`, `code`

`ProfileData`: `email`, `picture`

See [Authentication (Auth0)](#authentication-auth0) for full flow diagrams.

---

### `users` — Accounts and profiles

**Purpose:** Own the `users` / `user_auth_identities` / `user_profiles` tables. Called by `auth` on every authenticated request to link Auth0 subjects to internal UUIDs.

**Dependencies:** `externaldb` pool.

| Access | Method | Path | Function | Description |
|--------|--------|------|----------|-------------|
| private | POST | `/users/upsert-from-auth` | `UpsertFromAuth` | Find/create user + identity; update last login |
| auth | POST | `/users/profile` | `UpsertProfile` | Create or update the authenticated user's profile |
| auth | GET | `/users/profile` | `GetProfile` | Return profile (404 if none) |
| auth | GET | `/users/username-availability` | `CheckUsernameAvailability` | Query `?username=` — is it free? |

**`UpsertFromAuthParams`:** `provider`, `provider_user_id`, `email`, `email_verified`, `raw_profile`

**`UpsertUserProfileRequest`:** `first_name`, `last_name`, `username`, `phone_country_code`, `phone_national_number`, `date_of_birth`, `avatar_object_key`, `avatar_url`, `bio`

**Validation highlights:** username 3–20 chars `[a-z0-9_.]`; names 1–80 chars; phone E.164; DOB not in the future.

**Tables:** `users`, `user_auth_identities`, `user_profiles` (writes). `user_follows` exists in migrations but has **no API yet**.

---

### `onboarding` — First-run preferences

**Purpose:** Serve reference data for the onboarding wizard and persist the user's choices.

**Dependencies:** `externaldb` pool; requires auth for completion.

| Access | Method | Path | Function | Description |
|--------|--------|------|----------|-------------|
| public | GET | `/onboarding/cities` | `ListCities` | Active cities (Hanoi, HCMC, Da Nang, …) |
| public | GET | `/onboarding/categories` | `ListCategories` | Purpose categories (Restaurants, Cafes, …) |
| public | GET | `/onboarding/vibes` | `ListVibes` | Vibe options (Chill, Trendy, …) |
| public | GET | `/onboarding/venues` | `ListVenues` | Curated venues for `?city_id=` (required) |
| auth | POST | `/onboarding/complete` | `CompleteOnboarding` | Save preferences and mark onboarding done |

**`CompleteOnboardingRequest`:** `skip`, `city_ids[]`, `category_ids[]` (max 3), `vibe_ids[]`, `place_ids[]`

When `skip` is false, completion runs in one transaction:

1. Wipe and re-insert `user_explore_cities`, `user_purpose_categories`, `user_vibes`, `user_liked_places` (source `"onboarding"`).
2. Set `users.onboarding_status = 'completed'` and `onboarding_completed_at = now()`.

**Tables:** reads `cities`, `purpose_categories`, `vibes`, `places`; writes preference junction tables + `users.onboarding_*`.

---

### `places` — Place cache and Google proxy

**Purpose:** Interactive place lookup for the client. Proxies Google Autocomplete + Details, caches results in Postgres, and serves place detail/photo APIs.

**Dependencies:** `externaldb` pool, `googlePlacesClient` (`GooglePlacesAPIKey`).

| Access | Method | Path | Function | Description |
|--------|--------|------|----------|-------------|
| public | GET | `/places/:id` | `GetPlace` | Load cached place by UUID; async refresh if stale |
| public | GET | `/places/:id/photo` | `GetPlacePhoto` | Live photo URL by `?index=` + `?w=` (default 800, max 1600) |
| public | POST | `/places/autocomplete` | `AutocompletePlaces` | Proxy Google Autocomplete (New) |
| public | POST | `/places/resolve` | `ResolvePlace` | Upsert Google place → internal UUID + full detail |
| private | — | (RPC) | `ResolvePlaceForPlan` | Same resolve for `lists` plan creation; returns `place_id` only |

**`AutocompleteRequest`:** `input` (≥2 chars), `session_token` (required), `city_slug` (accepted but unused), `latitude`, `longitude`

**`ResolvePlaceRequest`:** `google_place_id`, `session_token`, `city_slug`, optional lat/lng (lat/lng currently ignored for city resolution)

**`PlaceDetail` fields:** `id`, `google_place_id`, `name`, `address`, `neighborhood`, `latitude`, `longitude`, `rating`, `user_rating_count`, `price_level`, `hours`, `hours_json`, `phone`, `website`, `business_status`, `cover_image_url`, `photo_urls`, `tags`, `city_slug`, `city_name`, `last_synced_at`, `is_stale`

**Tables:** `places` (upsert), `cities` (read for city hint).

See [Google Places API integration](#google-places-api-integration) for caching, field masks, and retries.

---

### `lists` — Curated lists and user plans

**Purpose:** Serve curated place lists (seeded from JSON + Google Text Search) and user-created day plans. Both live in the same `place_lists` / `place_list_entries` tables, discriminated by `list_type` (`curated` vs `user_plan`).

**Dependencies:** `externaldb` pool, own `googlePlacesClient`, private call to `places.ResolvePlaceForPlan`.

| Access | Method | Path | Function | Description |
|--------|--------|------|----------|-------------|
| public | GET | `/lists` | `ListPlaceLists` | Active curated lists with cover + entry count |
| public | GET | `/lists/details/:slug` | `GetPlaceList` | Full list/plan detail including stops |
| public | GET | `/plans` | `ListPlans` | Public user plans (`?limit`, `?city_slug`) |
| auth | POST | `/plans` | `CreatePlan` | Create a user plan with stops |
| private | POST | `/lists/sync` | `SyncPlaceLists` | Sync curated lists from embedded seed JSON |
| private | GET | `/lists/sync-runs` | `ListSyncRuns` | Recent sync audit runs (`?limit`) |
| private | POST | `/lists/enrich-plan-places` | `EnrichPlanPlaces` | Backfill missing cover images on plan places |

**Cron job**

```go
cron.NewJob("sync-place-lists", cron.JobConfig{
    Title:    "Sync curated place lists from seed JSON",
    Every:    24 * cron.Hour,
    Endpoint: SyncPlaceLists,
})
```

Runs every 24 hours (not locally / not in preview envs). Seed data is embedded from `lists/seeds/seed-lists.json`.

**`CreatePlanRequest`:** `slug?`, `title` (required), `subtitle`, `city_slug` (required), `category`, `area`, `occasions[]`, `visibility` (`public`/`private`), `saves_count`, `stops[]`

Each stop needs `place_id` **or** `google_place_id`, plus `place_name`, `stop_order`, optional `time_label`, `activity_type`, `note`, `image_url`, `tags`.

**Tables:** `place_lists`, `place_list_entries`, `list_sync_runs`; reads/writes `places` and `cities`; joins `users` / `user_profiles` for creator display.

---

### `events` — Nightlife events

**Purpose:** Read-only feed of external events (currently Vietnam Nightlife). The Go service never calls an external events API — rows are imported out-of-band.

**Dependencies:** `externaldb` pool.

| Access | Method | Path | Function | Description |
|--------|--------|------|----------|-------------|
| public | GET | `/events/upcoming` | `ListUpcomingEvents` | Small hero set with posters (`?limit` default 4 max 20, `?city_slug`) |
| public | GET | `/events` | `ListEvents` | Paginated list (`?limit` default 50 max 100, `?offset`, `?city_slug`) |

**`Event` fields:** `id`, `event_name`, `event_url`, `venue_name`, `venue_url`, `schedule_text`, `music_text`, `description`, `poster_image_url`, `imported_at`, `city_slug`, `city_name`, `source_slug`, `source_name`

**Import path:** a scraper (or ops script) calls the Postgres function `replace_external_events(p_source_slug, p_events JSONB)` with service-role credentials. Seeded source: `vietnamnightlife` → `https://vietnamnightlife.com`.

**Tables:** `event_sources`, `external_events`.

---

### `debug` — Connectivity check

| Access | Method | Path | Function | Description |
|--------|--------|------|----------|-------------|
| public | GET | `/debug/db-ping` | `DBPing` | Pings the Supabase pool; returns `{ ok, message }` |

**Production note:** This endpoint is public and unauthenticated. It does not leak data, but it confirms DB reachability and can be abused for load. Lock it down (`private` / auth) or remove it before production.

---

### Endpoint count summary

| Service | Public | Auth | Private | Total |
|---------|--------|------|---------|-------|
| auth | 5 | 1 | 0 (+ authhandler) | 6 |
| users | 0 | 3 | 1 | 4 |
| onboarding | 4 | 1 | 0 | 5 |
| places | 4 | 0 | 1 | 5 |
| lists | 3 | 1 | 3 | 7 |
| events | 2 | 0 | 0 | 2 |
| debug | 1 | 0 | 0 | 1 |
| **Total** | **19** | **6** | **5** | **30** |

---

## Google Places API integration

Lo uses the **Google Places API (New)** (`places.googleapis.com/v1`). There are **two independent HTTP clients** in different packages. Both read the Encore secret `GooglePlacesAPIKey` (each package declares its own `secrets` struct against the same secret name).

```mermaid
flowchart LR
  subgraph interactive [Interactive path - places service]
    Auto[Autocomplete]
    Details[Place Details]
    PhotoP[Photo Media]
  end

  subgraph batch [Batch path - lists service]
    Text[Text Search]
    PhotoL[Photo Media]
  end

  Client -->|typeahead + pick| Auto
  Auto --> Details
  Details --> PhotoP
  Details --> Cache[(places table)]

  Cron[Daily cron / enrich] --> Search
  Search --> PhotoL
  Search --> Cache
```

### Why two clients?

| Client | File | Role |
|--------|------|------|
| Interactive | [`places/google_client.go`](places/google_client.go) | Client typeahead → resolve → cache |
| Batch | [`lists/google_client.go`](lists/google_client.go) | Seed sync + cover-image enrichment |

They call **disjoint** Google endpoints in practice:

| Google API | Method | URL | Used by |
|------------|--------|-----|---------|
| Autocomplete (New) | POST | `https://places.googleapis.com/v1/places:autocomplete` | `places` |
| Place Details (New) | GET | `https://places.googleapis.com/v1/places/{id}` | `places` |
| Text Search (New) | POST | `https://places.googleapis.com/v1/places:searchText` | `lists` |
| Photo Media (New) | GET | `https://places.googleapis.com/v1/{photoName}/media?maxWidthPx=N&skipHttpRedirect=true` | both |

Not used: Directions, Geocoding, Maps JS, Places Nearby Search.

> **Dead code note:** `places` declares `googlePlacesSearchURL` / `textSearchRequest` but never calls Text Search. Text Search lives only in `lists`.

### Shared HTTP conventions

All Google calls go through a small `doJSON` / `doJSONWithRetry` layer:

| Concern | Behavior |
|---------|----------|
| Auth header | `X-Goog-Api-Key: {GooglePlacesAPIKey}` |
| Field selection | `X-Goog-FieldMask: …` (drives Google SKU pricing) |
| Session billing | `X-Goog-Session-Token` on Autocomplete + matching Details resolve |
| Content type | `application/json` |
| Timeout | 20s per HTTP request |
| Retries | 4 attempts, backoff 0s / 1s / 2s / 4s |
| Retryable | HTTP 429, 500, 502, 503, 504 only |
| Region / language | `regionCode: "VN"`, `languageCode: "en"` |
| Location bias | Optional 15 km circle around lat/lng |

Missing API key → `"google places api key is not configured"`.

### `places` client — Autocomplete + Details + Photo

**Autocomplete** (`POST /places/autocomplete`)

Request body highlights:

```json
{
  "input": "<user text>",
  "sessionToken": "<client-generated>",
  "includedPrimaryTypes": ["restaurant", "cafe", "bar", "bakery", "food"],
  "languageCode": "en",
  "regionCode": "VN",
  "locationBias": { "circle": { "center": { "latitude": …, "longitude": … }, "radius": 15000 } }
}
```

Field mask:

```
suggestions.placePrediction.placeId,suggestions.placePrediction.text,suggestions.placePrediction.structuredFormat
```

Predictions are **not** stored in Postgres.

**Place Details** (`ResolvePlace` / background refresh)

Pro field mask (business data, no photos):

```
id,displayName,formattedAddress,shortFormattedAddress,addressComponents,location,rating,userRatingCount,priceLevel,regularOpeningHours,nationalPhoneNumber,websiteUri,businessStatus,types,primaryTypeDisplayName
```

With photos (resolve + photo refresh): same mask + `,photos`.

**Photo Media** — returns `{ "photoUri": "…" }`. Used for cover image on resolve and for `GET /places/:id/photo`.

### `lists` client — Text Search + Photo

Used by seed sync and `EnrichPlanPlaces`.

Field mask for Text Search:

```
places.id,places.displayName,places.shortFormattedAddress,places.addressComponents,places.location,places.rating,places.userRatingCount,places.priceLevel,places.primaryTypeDisplayName,places.types,places.photos
```

Request: `textQuery`, `pageSize: 1`, optional 15 km `locationBias`. Query construction joins seed name + location hint + city hint (e.g. `"Some Cafe, District 1, Ho Chi Minh City"`).

Concurrency: **3 workers** for both sync and enrich paths.

### Google → Postgres field mapping

| Google field | `places` column / API field |
|--------------|-----------------------------|
| `id` | `google_place_id` |
| `displayName.text` | `name` |
| `shortFormattedAddress` / `formattedAddress` | `address` |
| `addressComponents` (sublocality_level_1 or administrative_area_level_2) | `neighborhood` |
| `location.latitude/longitude` | `latitude`, `longitude` |
| `rating` | `rating_cached` |
| `userRatingCount` | `user_rating_count` |
| `priceLevel` enum | `price_level` 1–4 |
| `regularOpeningHours` | `hours_json` (JSONB) |
| `nationalPhoneNumber` | `phone` |
| `websiteUri` | `website` |
| `businessStatus` | `business_status` |
| `photos[].name` | `photo_names[]` |
| first photo → media `photoUri` | `cover_image_url` |
| `primaryTypeDisplayName` + `types[]` | `tags[]` |

**Price level mapping:** `INEXPENSIVE`→1, `MODERATE`→2, `EXPENSIVE`→3, `VERY_EXPENSIVE`→4.

**Tag humanization:** primary type first, then types with underscores → Title Case. Skipped types: `point_of_interest`, `establishment`, `food`, `store`, `premise`.

### Caching and staleness (`places` service)

| Rule | Constant | Effect |
|------|----------|--------|
| Pro data stale | `proRefreshAfter = 7 days` | `is_stale = true`; `GetPlace` fires background refresh |
| Photos need refresh | `photoRefreshAfter = 30 days` OR empty `photo_names` | Refresh uses Details field mask **with** photos |

Refresh behavior:

- `GetPlace` returns cached data immediately and starts a fire-and-forget goroutine (`maybeRefreshPlace`) with a **10s** context timeout.
- `ResolvePlace` / `ResolvePlaceForPlan` on cache hit return the existing UUID **without** refreshing, even if stale.
- Upsert (`UpsertPlaceFromGoogle`) uses `COALESCE(new, old)` so null Google fields never wipe good cached data. Non-empty `photo_names` / `tags` replace; empty arrays leave prior values.

`lists` has no Redis/HTTP cache — it relies on DB lookups (`FindPlaceByNameInCity`, existing `google_place_id`) and `source_hash` short-circuiting unchanged seed lists.

### Interactive flow: autocomplete → resolve → plan stop

```mermaid
sequenceDiagram
  participant C as Client
  participant P as places
  participant L as lists
  participant G as Google Places
  participant DB as Supabase

  C->>P: POST /places/autocomplete
  P->>G: places:autocomplete + session token
  G-->>P: predictions
  P-->>C: google_place_id list

  C->>P: POST /places/resolve
  P->>DB: GetPlaceByGooglePlaceID
  alt cache miss
    P->>G: GET places/{id} + photos + session token
    G-->>P: details + photo names
    P->>G: GET photo media
    P->>DB: UpsertPlaceFromGoogle
  end
  P-->>C: place_id + PlaceDetail

  C->>L: POST /plans Authorization Bearer
  L->>P: ResolvePlaceForPlan google_place_id
  P->>DB: upsert if needed
  P-->>L: place_id
  L->>DB: insert place_lists + place_list_entries
  L-->>C: PlaceListDetail
```

### Cost and resilience controls

- Prefer DB cache hits before any Google call.
- Autocomplete → Details session tokens so Google bills Autocomplete as a session, not per keystroke + Details.
- Narrow field masks (SKU-aware).
- Cap Text Search `pageSize` at 1.
- Worker pools of 3 for batch work; retry only on rate limits / 5xx.
- Seed sync skips Google when `source_hash` is unchanged and all seed names already link.

### Known gaps

- `places` declares Text Search URL/types but never uses them.
- `AutocompleteRequest.CitySlug` is accepted but ignored.
- `ResolvePlaceRequest.Latitude/Longitude` are accepted but ignored in city resolution.
- `ResolvePlace` cache hits never trigger background refresh (only `GetPlace` does).
- `lists` writes to `places` directly during sync/enrich, bypassing the `places` service.

---

## Authentication (Auth0)

**Identity provider:** Auth0 (OIDC + OAuth2).  
**Not used:** Supabase Auth / GoTrue. Supabase is Postgres-only.

Config lives in [`auth/auth-config.cue`](auth/auth-config.cue):

| Field | Local (`encore run`) | Cloud / native |
|-------|----------------------|----------------|
| `ClientID` | `xkRK6mYn9bVq1S8ASiwKmT2oWwX2vtAs` | same |
| `Domain` | `dev-nnz0pmv3onjn82oq.us.auth0.com` | same |
| `CallbackURL` | `http://localhost:8081/callback` | `loapp://callback` |
| `LogoutURL` | `http://localhost:8081/` | `loapp://` |

Secret: `Auth0ClientSecret`. Scopes: `openid profile email`.

### Session model

Stateless. There is **no server-side session store**. The client keeps the Auth0 **ID token** (JWT) and sends `Authorization: Bearer <id_token>` on every `auth`-annotated endpoint. Refresh tokens from Auth0 are obtained during OTP verify but **not** returned to the client.

`auth.UID` is the internal `users.id` UUID — **not** the Auth0 `sub`.

### Important performance note

`AuthHandler` calls `users.UpsertFromAuth` on **every** authenticated request (not just login). That transaction finds/creates the user, upserts the identity, and updates `last_login_at`.

### Flow A — OAuth authorization code (Universal Login)

Supports Auth0 social connections (Google, Apple, Facebook, …) and database users.

```mermaid
sequenceDiagram
  participant C as Client
  participant A as auth service
  participant A0 as Auth0
  participant U as users service

  C->>A: POST /auth/login
  A-->>C: state + auth_code_url
  C->>A0: Redirect to authorize URL
  A0-->>C: Redirect to callback?code=…
  C->>A: POST /auth/callback code
  A->>A0: Exchange code → tokens
  A->>A0: Verify ID token
  A->>U: UpsertFromAuth
  A-->>C: token + onboarding_status + profile_exists
```

### Flow B — Passwordless email OTP

1. `POST /auth/otp/start` `{ "email": "…" }`  
   → Auth0 `POST /passwordless/start` with `connection: email`, `send: code`.  
   → `{ "sent": true }`
2. User receives numeric code by email.
3. `POST /auth/otp/verify` `{ "email", "code" }`  
   → Auth0 `POST /oauth/token` with grant  
   `http://auth0.com/oauth/grant-type/passwordless/otp`  
   → verify ID token → same `CallbackResponse` as OAuth.

If the client sends `X-Forwarded-For`, it is forwarded as Auth0's `auth0-forwarded-for` header for brute-force protection.

Auth0 HTTP errors map to Encore codes: 429 → `ResourceExhausted`, 400 → `InvalidArgument`, misconfiguration → `FailedPrecondition`, bad code → `PermissionDenied`.

### Flow C — Per-request Bearer verification

1. Client sends `Authorization: Bearer <id_token>`.
2. Encore invokes `AuthHandler`.
3. `oidc.Verifier` validates the JWT against Auth0's discovery document.
4. Claims extracted; `sub` parsed by [`auth/subject.go`](auth/subject.go).
5. `users.UpsertFromAuth` runs.
6. Handler receives `auth.UID` = `users.id` and `auth.Data()` = `ProfileData{Email, Picture}`.

### Google Sign-In (indirect)

There is **no** Google client ID and **no** Google token verification in this backend.

Google login works only when configured as an Auth0 social connection. Auth0 issues a normal ID token whose `sub` looks like `google-oauth2|<google_user_id>`. The backend maps prefixes:

| Auth0 `sub` prefix | Stored `provider` |
|--------------------|-------------------|
| `google-oauth2` | `google` |
| `apple` | `apple` |
| `facebook` | `facebook` |
| `email` / `auth0` | `email` |
| other | `auth0` |

Stored in `user_auth_identities` as `(provider, provider_user_id)`.

### Logout

`GET /auth/logout` returns  
`https://{Domain}/v2/logout?returnTo={LogoutURL}&client_id={ClientID}`.  
The client redirects there and must discard its local ID token. The backend does not revoke tokens.

---

## Data model

```mermaid
erDiagram
  users ||--o{ user_auth_identities : has
  users ||--o| user_profiles : has
  users ||--o{ user_follows : follower
  users ||--o{ user_follows : following
  users ||--o{ user_explore_cities : prefers
  users ||--o{ user_purpose_categories : prefers
  users ||--o{ user_vibes : prefers
  users ||--o{ user_liked_places : likes
  users ||--o{ place_lists : creates

  cities ||--o{ places : contains
  cities ||--o{ place_lists : scopes
  cities ||--o{ user_explore_cities : selected
  cities ||--o{ external_events : hosts

  purpose_categories ||--o{ place_categories : tags
  purpose_categories ||--o{ user_purpose_categories : selected
  vibes ||--o{ place_vibes : tags
  vibes ||--o{ user_vibes : selected
  places ||--o{ place_categories : in
  places ||--o{ place_vibes : in
  places ||--o{ user_liked_places : liked
  places ||--o{ place_list_entries : stop
  places ||--o{ external_events : venue

  place_lists ||--o{ place_list_entries : contains
  event_sources ||--o{ external_events : imports
```

### Identity

| Table | Role |
|-------|------|
| `users` | Canonical account; `id` becomes `auth.UID`. Soft-delete via `deleted_at`. |
| `user_auth_identities` | Linked Auth0 providers (`email`, `google`, `apple`, `facebook`, `auth0`) |
| `user_profiles` | Display name, username, phone, DOB, avatar, bio |
| `user_follows` | Follower graph — **schema only, no Go API yet** |

`users.onboarding_status` enum: `not_started` \| `in_progress` \| `completed`.

### Reference data

| Table | Role |
|-------|------|
| `cities` | Hanoi / HCMC / Da Nang, etc. |
| `purpose_categories` | Restaurants, Cafes, Bars, … |
| `vibes` | Chill, Trendy, Social, Unique, … |
| `places` | Curated (`source=curated`) or Google-cached (`source=google`) venues |

`places` Google cache columns (migration `20260721100000_places_google_cache.sql`): `user_rating_count`, `phone`, `website`, `hours_json`, `photo_names[]`, `business_status`.

### User preferences (onboarding)

`user_explore_cities`, `user_purpose_categories`, `user_vibes`, `user_liked_places` — wiped and rewritten on `POST /onboarding/complete`.

### Content (lists / plans)

| Table | Role |
|-------|------|
| `place_lists` | Unified header: `list_type` = `curated` \| `user_plan`; `visibility` = `public` \| `private` |
| `place_list_entries` | Stops: `stop_order`, `time_label`, `activity_type`, `seed_name`, … |
| `list_sync_runs` | Audit log for curated seed sync |

### Events

| Table | Role |
|-------|------|
| `event_sources` | e.g. `vietnamnightlife` |
| `external_events` | Scraped event rows; unique `(source_id, event_url)` |

---

## Configuration and secrets

### Secrets

| Secret | Consumers | Purpose |
|--------|-----------|---------|
| `ExternalDBPassword` | All DB services via `externaldb` | Supabase Postgres password |
| `Auth0ClientSecret` | `auth` | OAuth + passwordless client secret |
| `GooglePlacesAPIKey` | `places`, `lists` | Google Places API (New) key |

Set with Encore CLI:

```bash
encore secret set --type local ExternalDBPassword
encore secret set --type development Auth0ClientSecret
encore secret set --type production GooglePlacesAPIKey
```

Types: `local`, `development`, `preview`, `production`.  
Local overrides can also live in `.secrets.local.cue` (gitignored values — do not commit real secrets).

### Non-secret config

Auth0 `ClientID`, `Domain`, `CallbackURL`, `LogoutURL` are in `auth/auth-config.cue` (CUE + Encore `#Meta.Environment`).

DB host is hardcoded in `pkg/externaldb`:

```
postgresql://postgres:{ExternalDBPassword}@db.xofbxjbrzpsocnjdsmmk.supabase.co:5432/postgres
```

---

## Local development

### Prerequisites

- [Encore CLI](https://encore.dev/docs/install) — `brew install encoredev/tap/encore`
- [Supabase CLI](https://supabase.com/docs/guides/cli) — `brew install supabase/tap/supabase`
- [sqlc](https://docs.sqlc.dev) — `brew install sqlc`
- Docker (optional; not required for the external DB, but useful for other tooling)

### Run the app

```bash
# Set secrets (once)
encore secret set --type local ExternalDBPassword
encore secret set --type local Auth0ClientSecret
encore secret set --type local GooglePlacesAPIKey

encore run
```

- API: `http://localhost:4000`
- Local developer dashboard (traces, service catalog, architecture): [http://localhost:9400](http://localhost:9400/)

Example:

```bash
curl http://localhost:4000/onboarding/cities
curl "http://localhost:4000/debug/db-ping"
curl -H "Authorization: Bearer $AUTH_TOKEN" http://localhost:4000/users/profile
```

### Supabase migrations

```bash
supabase link --project-ref xofbxjbrzpsocnjdsmmk
supabase db push
```

Schema source of truth: `supabase/migrations/*.sql`.

### sqlc typed queries

| Path | Role |
|------|------|
| `sqlc.yaml` | Config |
| `supabase/migrations/*.sql` | Schema for type inference |
| `db/queries/*.sql` | Query definitions |
| `internal/dbgen/*` | Generated Go |

After changing migrations or queries:

```bash
sqlc generate
```

Services keep business logic in `*/api.go` etc. and call `dbgen.New(db)` / `dbgen.New(tx)`. Transactions use `pgx` (`BeginTx`) with queries bound to the tx.

### Seed demo plans

[`scripts/create_plans.sh`](scripts/create_plans.sh) POSTs five public Discover plans to `/plans`:

```bash
export AUTH_TOKEN="<Auth0 ID token>"
# optional: export BASE_URL=http://127.0.0.1:4000
./scripts/create_plans.sh
```

---

## Testing

```bash
encore test ./...
```

| Package | Tests |
|---------|-------|
| `auth` | Passwordless / OTP helpers |
| `users` | Profile API |
| `onboarding` | Complete-onboarding / dbgen types |
| `lists` | Google utils, plan create, plan list, seeds |
| `events` | Validation |
| `places` | **None yet** |

Traces for tests are available via the local dashboard while Encore is running.

---

## Deployment

### Encore Cloud

```bash
git add -A && git commit -m "…"
git push encore
# or link GitHub in the Cloud Dashboard for branch-push deploys
```

Open the [Cloud Dashboard](https://app.encore.dev) for environments, secrets, and logs:

```bash
encore logs --env=prod
```

### Self-hosting

```bash
encore build docker --push
```

See [Encore self-host docs](https://encore.dev/docs/go/self-host/docker-build). Ensure `ExternalDBPassword`, `Auth0ClientSecret`, and `GooglePlacesAPIKey` are set for the target environment, and that Auth0 callback URLs match `auth-config.cue` for non-local clouds (`loapp://callback`).

### Cron

`sync-place-lists` runs every 24 hours in deployed environments. It does **not** run during local `encore run` or in Preview Environments.
