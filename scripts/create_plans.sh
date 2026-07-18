#!/usr/bin/env bash
# Seed Discover itineraries via POST /plans (CreatePlan).
# Usage:
#   AUTH_TOKEN="<clerk-jwt>" ./scripts/create_plans.sh
# Optional:
#   BASE_URL="http://127.0.0.1:4000" AUTH_TOKEN="..." ./scripts/create_plans.sh

set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:4000}"

if [[ -z "${AUTH_TOKEN:-}" ]]; then
  echo "AUTH_TOKEN is required (Bearer token for //encore:api auth POST /plans)" >&2
  exit 1
fi

post_plan() {
  local name="$1"
  local body="$2"
  echo "==> Creating plan: ${name}"
  curl -sS -X POST "${BASE_URL}/plans" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "${body}"
  echo
  echo
}

# 1. The Perfect D1 Date Night
post_plan "The Perfect D1 Date Night" '{
  "title": "The Perfect D1 Date Night",
  "subtitle": "Tight and romantic, built around the Bến Thành / Lý Tự Trọng cocktail cluster so you'\''re never more than a few minutes between stops.",
  "city_slug": "hcmc",
  "category": "Date night",
  "area": "District 1",
  "occasions": ["Date"],
  "visibility": "public",
  "stops": [
    {
      "stop_order": 1,
      "place_name": "Aiii",
      "time_label": "6:30 PM",
      "activity_type": "restaurant",
      "note": "Dinner. Contemporary Vietnamese-French, 40 seats, open kitchen. Book ahead — small room."
    },
    {
      "stop_order": 2,
      "place_name": "Summer Experiment",
      "time_label": "8:30 PM",
      "activity_type": "bar",
      "note": "First cocktails. Intimate, herb-garden balcony, conceptual menu. The 365 Days of Summer."
    },
    {
      "stop_order": 3,
      "place_name": "Stir",
      "time_label": "9:45 PM",
      "activity_type": "bar",
      "note": "Quieter second round. Ground floor, classic-focused — somewhere you can actually talk."
    },
    {
      "stop_order": 4,
      "place_name": "Ủ Bar",
      "time_label": "11:00 PM",
      "activity_type": "bar",
      "note": "Nightcap. Fermentation-led, slow-paced, meditative. The wind-down before home."
    }
  ]
}'

# 2. First 24 Hours in Saigon
post_plan "First 24 Hours in Saigon" '{
  "title": "First 24 Hours in Saigon",
  "subtitle": "The greatest-hits starter for someone who'\''s never been — the dishes and views everyone should hit first, paced across a full day.",
  "city_slug": "hcmc",
  "category": "Visitor itinerary",
  "area": "Cross-city",
  "occasions": ["Visitor"],
  "visibility": "public",
  "stops": [
    {
      "stop_order": 1,
      "place_name": "Phở Hòa Pasteur",
      "time_label": "8:00 AM",
      "activity_type": "restaurant",
      "note": "Breakfast. Phở since 1968, English-friendly, order with quẩy."
    },
    {
      "stop_order": 2,
      "place_name": "The Workshop",
      "time_label": "9:30 AM",
      "activity_type": "cafe",
      "note": "Coffee. Saigon'\''s original specialty café — see what Vietnamese coffee can be."
    },
    {
      "stop_order": 3,
      "place_name": "Bánh Mì Huỳnh Hoa",
      "time_label": "12:00 PM",
      "activity_type": "restaurant",
      "note": "Lunch on the go. The loaded bánh mì everyone means. Takeaway, queue moves fast."
    },
    {
      "stop_order": 4,
      "place_name": "Maison Marou",
      "time_label": "2:00 PM",
      "activity_type": "cafe",
      "note": "Afternoon break. Bean-to-bar chocolate, dark-chocolate egg cream."
    },
    {
      "stop_order": 5,
      "place_name": "Pizza 4P'\''s",
      "time_label": "6:30 PM",
      "activity_type": "restaurant",
      "note": "Dinner. The cult Vietnamese-Japanese pizza, house-made cheese. Book ahead."
    },
    {
      "stop_order": 6,
      "place_name": "Chill Sky Bar",
      "time_label": "8:30 PM",
      "activity_type": "bar",
      "note": "Sunset-into-night cocktails. 26th-floor rooftop, the classic first-night view."
    },
    {
      "stop_order": 7,
      "place_name": "The Gangs Central",
      "time_label": "10:00 PM",
      "activity_type": "bar",
      "note": "Nightcap. Open-air Nguyễn Huệ beer garden, high energy, easy to end on."
    }
  ]
}'

# 3. Squad Big Day Out
post_plan "Squad Big Day Out" '{
  "title": "Squad Big Day Out",
  "subtitle": "Built to escalate — food and beer early, energy climbing to the clubs. Paced for a group.",
  "city_slug": "hcmc",
  "category": "Squad night",
  "area": "District 1",
  "occasions": ["Squad"],
  "visibility": "public",
  "stops": [
    {
      "stop_order": 1,
      "place_name": "Quán Ụt Ụt",
      "time_label": "4:00 PM",
      "activity_type": "restaurant",
      "note": "Late lunch / pre-game. House-smoked BBQ, craft beer on tap, communal tables."
    },
    {
      "stop_order": 2,
      "place_name": "Pasteur Street Brewing",
      "time_label": "6:00 PM",
      "activity_type": "bar",
      "note": "Beers. The OG taproom, Jasmine IPA. Alley setting, buzzy."
    },
    {
      "stop_order": 3,
      "place_name": "The Gangs Central",
      "time_label": "8:00 PM",
      "activity_type": "bar",
      "note": "Beer garden turns club-energy. Open-air on Nguyễn Huệ, grilled food, live music."
    },
    {
      "stop_order": 4,
      "place_name": "Bodega Saigon",
      "time_label": "10:30 PM",
      "activity_type": "club",
      "note": "Warm-up club. Techno/hip-hop, ~200 capacity, arrive before it fills."
    },
    {
      "stop_order": 5,
      "place_name": "Ciné Saigon",
      "time_label": "12:00 AM",
      "activity_type": "club",
      "note": "Main event. Heritage cinema venue, DJMag Top 100, best sound system in the city."
    },
    {
      "stop_order": 6,
      "place_name": "Cơm Tấm Hồng Calmette",
      "time_label": "2:00 AM",
      "activity_type": "restaurant",
      "note": "Late-night landing. D4, open till 3AM, grilled ribs. The classic Saigon way to end a big night."
    }
  ]
}'

# 4. Saigon Food Crawl: Street to Star
post_plan "Saigon Food Crawl: Street to Star" '{
  "title": "Saigon Food Crawl: Street to Star",
  "subtitle": "Designed as an eating progression — street food institutions early, building to the city'\''s only Michelin star at night.",
  "city_slug": "hcmc",
  "category": "Food crawl",
  "area": "Cross-city",
  "occasions": ["Food-focused"],
  "visibility": "public",
  "stops": [
    {
      "stop_order": 1,
      "place_name": "Cơm Tấm Ba Ghiền",
      "time_label": "9:00 AM",
      "activity_type": "restaurant",
      "note": "Breakfast. Bib Gourmand broken rice — the giant grilled pork chop. Share one."
    },
    {
      "stop_order": 2,
      "place_name": "Hủ Tiếu Mỹ Tho Thanh Xuân",
      "time_label": "11:30 AM",
      "activity_type": "restaurant",
      "note": "Late morning noodles. 70+ years, My Tho-style, crab variation the highlight."
    },
    {
      "stop_order": 3,
      "place_name": "Bánh Xèo 46A",
      "time_label": "1:30 PM",
      "activity_type": "restaurant",
      "note": "Lunch. Bib Gourmand, four-generation crispy rice-flour crepe. Wrap and dip."
    },
    {
      "stop_order": 4,
      "place_name": "Maison Marou",
      "time_label": "4:00 PM",
      "activity_type": "cafe",
      "note": "Sweet reset. Chocolate-forward, light — just enough to bridge to dinner."
    },
    {
      "stop_order": 5,
      "place_name": "Mặn Mòi",
      "time_label": "6:30 PM",
      "activity_type": "restaurant",
      "note": "Early dinner. Michelin-listed regional Vietnamese, sharing plates."
    },
    {
      "stop_order": 6,
      "place_name": "Akuna",
      "time_label": "8:30 PM",
      "activity_type": "restaurant",
      "note": "The finish. One Michelin Star — the city'\''s only one. Tasting menu, reservations essential."
    }
  ]
}'

# 5. Slow Saigon: Coffee & Calm
post_plan "Slow Saigon: Coffee & Calm" '{
  "title": "Slow Saigon: Coffee & Calm",
  "subtitle": "A low-key day for someone exploring alone or wanting calm — specialty coffee, quiet corners, no queues-or-bust stops.",
  "city_slug": "hcmc",
  "category": "Coffee & calm",
  "area": "Mixed (tight clusters)",
  "occasions": ["Solo"],
  "visibility": "public",
  "stops": [
    {
      "stop_order": 1,
      "place_name": "16 Grams Café",
      "time_label": "9:00 AM",
      "activity_type": "cafe",
      "note": "Slow start. Tân Định alley near the pink church, single-origin pour-overs, quiet."
    },
    {
      "stop_order": 2,
      "place_name": "Okkio Caffè – Đồng Khởi",
      "time_label": "11:00 AM",
      "activity_type": "cafe",
      "note": "Second coffee. 2nd-floor, Opera House views, good people-watching solo."
    },
    {
      "stop_order": 3,
      "place_name": "Bếp Mẹ Ỉn",
      "time_label": "1:00 PM",
      "activity_type": "restaurant",
      "note": "Lunch. Saigonese home cooking, sharing plates but fine solo, warm and unpretentious."
    },
    {
      "stop_order": 4,
      "place_name": "Maison Marou",
      "time_label": "3:00 PM",
      "activity_type": "cafe",
      "note": "Afternoon sit. Chocolate and a book — designed to linger."
    },
    {
      "stop_order": 5,
      "place_name": "Every Half Coffee Roasters",
      "time_label": "4:30 PM",
      "activity_type": "cafe",
      "note": "Thảo Điền pivot. Flagship roastery, natural light, the best beans in the neighbourhood."
    },
    {
      "stop_order": 6,
      "place_name": "Okra FoodBar",
      "time_label": "6:30 PM",
      "activity_type": "restaurant",
      "note": "Early solo dinner at Okra'\''s counter — ~20 seats facing the open kitchen, easy to eat alone."
    }
  ]
}'

echo "Done. Seeded 5 Discover plans."
