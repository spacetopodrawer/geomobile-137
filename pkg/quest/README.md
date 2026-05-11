# 🎮 Module pkg/quest — Lékié Quest VR & Freemium System

**Version:** 0.1.0 (Alpha)  
**Status:** In Development  
**Part of:** geo-mobile137 v0.2.1+

---

## 📌 Overview

The `quest` module provides a complete **quest/game management system** for Lékié Quest VR, including:

- ✅ Quest lifecycle management (available → active → complete)
- ✅ User progression (XP, levels, badges)
- ✅ Freemium subscription tiers (6 levels: Free → PRO ATELIER)
- ✅ In-app purchases (cosmetics, tier upgrades)
- ✅ Leaderboard (global, regional, weekly)
- ✅ Analytics & quest metrics

---

## 🎯 Freemium Tiers

| Tier | Name | Quests/week | XP/week | Cosmetic Discount | Price |
|------|------|-------------|---------|-------------------|-------|
| 0 | Free | 3 | 50 | 0% | Free |
| 1 | Casual | 6 | 500 | 10% | 2,000 XAF |
| 2 | Player | 10 | 2,000 | 20% | 5,000 XAF |
| 3 | Expert | 15 | 10,000 | 30% | 15,000 XAF |
| 4 | Pro | Unlimited | 50,000 | 50% | 50,000 XAF |
| 5 | PRO ATELIER | Unlimited | Unlimited | 100% | 150,000 XAF |

---

## 🎮 Quest Types

### 1. **Timeline Quest** (`timeline`)
**Goal:** Reconstruct a commune's cadastral evolution 2021-2025

- Player receives partial snapshots (2021, 2023, 2025)
- Must order/match them correctly
- Uses actual LEKIE_ dataset
- **XP Reward:** 100-300 XP

**Example:** "Arrange Monatélé's maps in chronological order"

### 2. **POI Hunt** (`poi_hunt`)
**Goal:** Locate specific amenities on the map

- "Find all hospitals in Lékié"
- "Identify 5 markets in Batchenga"
- Uses OSM + cadastral POI data
- **XP Reward:** 50-200 XP

### 3. **Parcel Challenge** (`parcel`)
**Goal:** Quiz on cadastral attributes

- "Who owns parcel DEP_LEKIE_001_002_003?"
- "What is the land use of this parcel?"
- "Estimate the area of this building"
- **XP Reward:** 75-250 XP

### 4. **Building Detective** (`detective`)
**Goal:** Count and identify buildings using satellite imagery

- Photogrammetry overlay + SVG tileset
- "How many buildings are in Monatélé town center?"
- Precision-based rewards (±5 = full XP, ±10 = half XP)
- **XP Reward:** 100-300 XP

### 5. **Temporal Glitch** (`glitch`)
**Goal:** Spot differences between two years' maps

- Side-by-side 2021 vs 2025 map comparison
- "Circle the new building"
- "Highlight the road that was added"
- **XP Reward:** 150-350 XP

### 6. **Owner Quiz** (`owner_quiz`)
**Goal:** Predict owner attributes from parcel characteristics

- Given: parcel geometry, location, history
- Predict: ownership type (private/public/cooperative)
- Uses machine learning hints
- **XP Reward:** 100-250 XP

---

## 🏗️ Architecture

```
pkg/quest/
├── types.go          # Data structures (Quest, UserProgress, etc.)
├── service.go        # Business logic (StartQuest, CompleteObjective, etc.)
├── handlers.go       # HTTP endpoints (/api/v1/quest/...)
└── README.md         # This file
```

### Dependencies

- **Database:** PostgreSQL (user_progress, quests, quest_sessions, leaderboard tables)
- **Cache:** Redis (user progress caching, leaderboard caching)
- **Payment Gateway:** Flutterwave, Paytech, or custom (for tier upgrades + cosmetics)
- **Analytics:** PostHog, Mixpanel, or Amplitude (optional, for quest metrics)

---

## 🚀 Quick Start

### 1. Initialize Service

```go
import "cadastre_ia/pkg/quest"

// Setup
db := setupPostgres()
cache := setupRedis()

questService := quest.NewQuestService(db, cache)
questHandlers := quest.NewQuestHandlers(questService)

// Register HTTP routes
mux := http.NewServeMux()
quest.RegisterRoutes(mux, questHandlers)
```

### 2. Start a Quest

```bash
POST /api/v1/quest/start
Content-Type: application/json
X-User-ID: user_12345

{
  "quest_id": "timeline_lekie_001"
}

# Response:
{
  "id": "qs_1715XXX",
  "user_id": "user_12345",
  "quest_id": "timeline_lekie_001",
  "started_at": "2026-05-11T10:30:00Z",
  "objective_progress": {
    "obj_001": 0,
    "obj_002": 0,
    "obj_003": 0
  }
}
```

### 3. Complete an Objective

```bash
POST /api/v1/quest/objective/complete
Content-Type: application/json

{
  "session_id": "qs_1715XXX",
  "objective_id": "obj_001"
}

# Triggers auto-check: if all objectives complete → quest completes
```

### 4. Check Progress

```bash
GET /api/v1/quest/progress
X-User-ID: user_12345

# Response:
{
  "user_id": "user_12345",
  "tier": 1,
  "total_xp": 850,
  "level": 1,
  "current_level_xp": 850,
  "available_xp_today": 150,
  "completed_quests": ["timeline_lekie_001", "poi_hunt_lekie_001"],
  "badges": [
    {"name": "Cartography Expert", "rarity": "rare"}
  ]
}
```

### 5. Upgrade Tier

```bash
POST /api/v1/quest/tier/upgrade
X-User-ID: user_12345
Content-Type: application/json

{
  "new_tier": 2,
  "duration_months": 1,
  "payment_id": "pay_flutterwave_12345"  // After payment success
}
```

---

## 📊 Quest Design Guide

### XP Scaling

- **Easy quests:** 50-150 XP (5-15 min)
- **Normal quests:** 150-300 XP (15-30 min)
- **Hard quests:** 300-500 XP (30-60 min)
- **Master quests:** 500-1000 XP (60+ min, rare)

**Level Progression:** 1000 XP per level (1→999 possible)

### Difficulty Progression

New users should start with **Easy** quests. After 500 XP, unlock **Normal**. After 2000 XP, unlock **Hard**.

```sql
-- Query to recommend difficulty for user
SELECT
  CASE
    WHEN total_xp < 500 THEN 'easy'
    WHEN total_xp < 2000 THEN 'normal'
    WHEN total_xp < 10000 THEN 'hard'
    ELSE 'master'
  END as recommended_difficulty
FROM user_progress
WHERE user_id = ?;
```

---

## 💳 Monetization Model

### Revenue Streams

1. **Subscription Tiers** (Monthly SaaS)
   - Casual → PRO: 2K-150K XAF/month
   - ~500 users × 10K XAF avg = 5M XAF/month (P1)

2. **Cosmetics IAP** (In-App Purchase)
   - Skins, avatars, map themes: 1K-50K XAF
   - Freemium users: conversion ~2-5%
   - Estimate: 1-5M XAF/month (P1)

3. **Battle Pass** (Seasonal, future)
   - 5K XAF/season, 10 unlockable cosmetics
   - Post-P1 feature

### Payment Flow

```
User clicks "Upgrade to Player Tier"
  ↓
Frontend opens Flutterwave/Paytech payment modal
  ↓
User pays 5,000 XAF
  ↓
Payment callback → /api/v1/quest/tier/upgrade
  ↓
Backend calls qs.UpgradeTier(userID, TierPlayer, 1 month)
  ↓
User's tier updated, XAF endpoint logs revenue
  ↓
Frontend celebrates with confetti animation
```

---

## 📈 Analytics & Metrics

Key metrics to track in PostHog/Mixpanel:

- **Engagement:**
  - Quests started per user
  - Completion rate (%)
  - Average time to complete
  
- **Monetization:**
  - Tier conversion rate (Free → Paid)
  - ARPPU (Average Revenue Per Paying User)
  - Cosmetic attachment rate
  
- **Retention:**
  - D1 (Day 1), D7, D30 retention
  - Weekly active users (WAU)
  
- **Content:**
  - Most popular quests
  - Most difficult objectives
  - Churn points (where players abandon)

**Dashboard:** Create a Grafana/Metabase dashboard tracking these metrics daily.

---

## 🐛 Known Limitations (P1)

- [ ] Offline quest support (planned for P2)
- [ ] Multiplayer co-op quests (planned for P3)
- [ ] Custom quest creation (planned for P4)
- [ ] Mobile app (currently web-only PWA)
- [ ] VR headset support (name says "VR" but MVP is 2D web)

---

## 🔐 Security Considerations

- **XP Cheating:** Log all XP awards, flag suspicious patterns (1000 XP in 1 second)
- **Payment Verification:** Always verify payment_id with gateway before granting tier/cosmetics
- **API Rate Limiting:** Max 10 requests/second per user (prevent brute force quest completion)
- **Session Hijacking:** Use secure auth tokens, short-lived JWT
- **Data Privacy:** Comply with Cameroon data protection law + GDPR (if applicable)

---

## 📝 Database Schema (PostgreSQL)

```sql
-- User progression
CREATE TABLE user_progress (
  user_id TEXT PRIMARY KEY,
  tier INTEGER DEFAULT 0,  -- TierLevel enum
  tier_expires_at TIMESTAMP,
  total_xp INTEGER DEFAULT 0,
  current_level_xp INTEGER DEFAULT 0,  -- 0-999
  level INTEGER DEFAULT 1,
  available_xp_today INTEGER DEFAULT 50,
  last_xp_reset TIMESTAMP DEFAULT NOW(),
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Quests (predefined)
CREATE TABLE quests (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT,
  type TEXT NOT NULL,  -- "timeline", "poi_hunt", etc.
  difficulty TEXT NOT NULL,
  region TEXT,
  admin_unit TEXT,
  min_tier INTEGER DEFAULT 0,
  min_xp INTEGER DEFAULT 0,
  target_xp INTEGER NOT NULL,
  estimated_time_min INTEGER,
  status TEXT DEFAULT 'available',
  expires_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Quest sessions (active/completed)
CREATE TABLE quest_sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  quest_id TEXT NOT NULL,
  started_at TIMESTAMP DEFAULT NOW(),
  completed_at TIMESTAMP,
  elapsed_seconds INTEGER DEFAULT 0,
  partial_xp INTEGER DEFAULT 0,
  abandoned BOOLEAN DEFAULT FALSE,
  FOREIGN KEY (user_id) REFERENCES user_progress(user_id),
  FOREIGN KEY (quest_id) REFERENCES quests(id)
);

-- Leaderboard (materialized view, refreshed hourly)
CREATE TABLE leaderboard_global (
  user_id TEXT PRIMARY KEY,
  total_points INTEGER,
  global_rank INTEGER,
  regional_rank INTEGER,
  region TEXT,
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Cosmetics (purchasable items)
CREATE TABLE cosmetics (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL,  -- "skin", "avatar", "map_theme"
  price_xaf INTEGER,
  description TEXT,
  image_url TEXT,
  released_at TIMESTAMP,
  limited_edition BOOLEAN DEFAULT FALSE,
  available_until TIMESTAMP
);

-- Quest events (for analytics)
CREATE TABLE quest_events (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  type TEXT NOT NULL,  -- "quest_start", "objective_complete", "tier_upgrade"
  quest_id TEXT,
  data JSONB,
  created_at TIMESTAMP DEFAULT NOW(),
  FOREIGN KEY (user_id) REFERENCES user_progress(user_id)
);
```

---

## 🧪 Testing

```bash
# Run unit tests
go test ./pkg/quest -v

# Run integration tests (requires live DB)
go test ./pkg/quest -integration -v

# Load testing (simulate 1000 concurrent users)
go run loadtest.go --users=1000 --duration=60s
```

---

## 📚 References

- Freemium SaaS metrics: https://www.greylock.com/greymatter/saas-metrics/
- Game progression design: https://www.gamasutra.com/view/feature/195782/
- Cadastral data handling: OHADA framework, QGIS documentation

---

**Maintained by:** Cowork/Claude + EBOLO ETINGUE Wilfried  
**Last Updated:** 2026-05-11
