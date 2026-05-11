# 🎬 PHASE 3: Frontend MVP — Architecture & Plan

**Date:** 2026-05-11  
**Phase:** 3 (Frontend Implementation)  
**Duration:** Estimated 3-4 days  
**Status:** 🟡 **STARTING**

---

## 🎯 Phase 3 Objectives

Build a **web-based MVP frontend** for geo-mobile137 that:

1. ✅ **Quest System UI** — Display available quests, start/complete/abandon, track progress
2. ✅ **Map Integration** — Show cadastral entities, quest locations, POI
3. ✅ **Cosmetics Shop** — Browse/purchase cosmetics with tier discounts
4. ✅ **Leaderboard Display** — Global, regional, weekly rankings
5. ✅ **User Dashboard** — Progression, tier status, stats
6. ✅ **Payment Integration** — Tier upgrade & cosmetic purchase flows
7. ✅ **Real-time Updates** — WebSocket for quest progress & leaderboard
8. ✅ **Responsive Design** — Mobile-first, works on all devices

---

## 📐 Frontend Architecture

### Technology Stack

```
Frontend Framework:   React 18+ or Vue 3+
State Management:     Redux Toolkit or Pinia
Maps:                 Leaflet.js or Mapbox GL
UI Components:        Material-UI or Shadcn/UI
Real-time:           Socket.io or native WebSocket
Build Tool:          Vite or Next.js
Package Manager:     npm or pnpm
Styling:             Tailwind CSS or styled-components
Charts:              Recharts or Chart.js (for leaderboard)
```

### Recommended: React + Redux + Leaflet

```
React 18           — Component-based UI
Redux Toolkit      — State management (quest state, user progress)
Leaflet.js         — Map rendering
Shadcn/UI          — Accessible components
Tailwind CSS       — Utility-first styling
Vite               — Fast build & dev server
Socket.io          — Real-time updates
```

---

## 🗂️ Frontend Project Structure

```
frontend/
├── src/
│   ├── components/
│   │   ├── QuestUI/
│   │   │   ├── QuestList.tsx        — Available quests
│   │   │   ├── QuestDetail.tsx      — Single quest view
│   │   │   ├── QuestSession.tsx     — Active quest UI
│   │   │   ├── ObjectiveTracker.tsx — Objective progress
│   │   │   └── QuestRewards.tsx     — Completion rewards
│   │   │
│   │   ├── MapUI/
│   │   │   ├── CadastreMap.tsx      — Main map component
│   │   │   ├── EntityMarker.tsx     — Entity on map
│   │   │   ├── QuestMarker.tsx      — Quest location
│   │   │   └── MapControls.tsx      — Zoom, layer toggle
│   │   │
│   │   ├── ShopUI/
│   │   │   ├── CosmeticsShop.tsx    — Shop display
│   │   │   ├── CosmeticCard.tsx     — Item card
│   │   │   ├── PricingDisplay.tsx   — Price w/ discount
│   │   │   └── CartCheckout.tsx     — Purchase flow
│   │   │
│   │   ├── LeaderboardUI/
│   │   │   ├── LeaderboardTabs.tsx  — Global/regional/weekly
│   │   │   ├── RankingTable.tsx     — Rankings table
│   │   │   ├── RankCard.tsx         — Single rank item
│   │   │   └── UserRank.tsx         — User's rank highlight
│   │   │
│   │   ├── UserDashboard/
│   │   │   ├── ProgressCard.tsx     — XP/level display
│   │   │   ├── TierCard.tsx         — Tier status
│   │   │   ├── StatsPanel.tsx       — User stats
│   │   │   ├── QuestHistory.tsx     — Completed quests
│   │   │   └── BadgeDisplay.tsx     — Earned badges
│   │   │
│   │   ├── PaymentUI/
│   │   │   ├── TierUpgradeModal.tsx — Upgrade dialog
│   │   │   ├── TierComparison.tsx   — Feature comparison
│   │   │   ├── PaymentConfirm.tsx   — Payment confirmation
│   │   │   └── PaymentStatus.tsx    — Payment result
│   │   │
│   │   ├── Navigation/
│   │   │   ├── Header.tsx           — Top navigation
│   │   │   ├── SideNav.tsx          — Side menu
│   │   │   ├── BottomNav.tsx        — Mobile bottom nav
│   │   │   └── BreadCrumbs.tsx      — Navigation context
│   │   │
│   │   └── Common/
│   │       ├── Loading.tsx          — Spinners
│   │       ├── ErrorBoundary.tsx    — Error handling
│   │       ├── ConfirmDialog.tsx    — Confirmations
│   │       └── Toast.tsx            — Notifications
│   │
│   ├── pages/
│   │   ├── QuestsPage.tsx           — Quests hub
│   │   ├── MapPage.tsx              — Map view
│   │   ├── LeaderboardPage.tsx      — Leaderboard
│   │   ├── ShopPage.tsx             — Cosmetics shop
│   │   ├── ProfilePage.tsx          — User profile
│   │   ├── TiersPage.tsx            — Subscription info
│   │   └── PaymentResultPage.tsx    — Payment callback
│   │
│   ├── redux/
│   │   ├── slices/
│   │   │   ├── authSlice.ts         — User auth state
│   │   │   ├── questSlice.ts        — Quest state
│   │   │   ├── userSlice.ts         — User progress
│   │   │   ├── leaderboardSlice.ts  — Rankings
│   │   │   ├── cosmticSlice.ts      — Cosmetics
│   │   │   └── paymentSlice.ts      — Payment state
│   │   ├── middleware/
│   │   │   └── websocketMiddleware.ts — Real-time updates
│   │   └── store.ts                 — Redux store config
│   │
│   ├── services/
│   │   ├── api.ts                   — API client (axios/fetch)
│   │   ├── websocket.ts             — WebSocket connection
│   │   ├── auth.ts                  — Auth service
│   │   ├── quest.ts                 — Quest API calls
│   │   ├── leaderboard.ts           — Leaderboard API
│   │   ├── payment.ts               — Payment service
│   │   └── cosmetics.ts             — Cosmetics API
│   │
│   ├── hooks/
│   │   ├── useQuest.ts              — Quest logic
│   │   ├── useLeaderboard.ts        — Leaderboard fetch
│   │   ├── useUser.ts               — User progress
│   │   ├── useWebSocket.ts          — WebSocket hook
│   │   └── usePayment.ts            — Payment flow
│   │
│   ├── utils/
│   │   ├── constants.ts             — Constants & enums
│   │   ├── helpers.ts               — Utility functions
│   │   ├── formatters.ts            — XP, time formatting
│   │   └── validators.ts            — Form validation
│   │
│   ├── styles/
│   │   ├── globals.css              — Global styles
│   │   ├── variables.css            — CSS variables
│   │   └── responsive.css           — Responsive rules
│   │
│   ├── App.tsx                      — Root component
│   └── main.tsx                     — Entry point
│
├── public/
│   ├── assets/
│   │   ├── logos/                   — Brand assets
│   │   ├── icons/                   — UI icons
│   │   └── placeholders/            — Placeholder images
│   │
│   └── index.html                   — HTML template
│
├── tests/
│   ├── components/
│   │   └── *.test.tsx               — Component tests
│   ├── redux/
│   │   └── *.test.ts                — Reducer tests
│   └── services/
│       └── *.test.ts                — Service tests
│
├── package.json
├── vite.config.ts                   — Build config
├── tsconfig.json                    — TypeScript config
└── tailwind.config.js               — Tailwind config
```

---

## 🔄 Data Flow

### Quest Flow
```
User Sees Available Quests
    ↓
Redux: questSlice.fetchAvailableQuests()
    ↓
API: GET /api/v1/quest/available (X-User-ID header)
    ↓
Backend: Returns quest list
    ↓
Redux: questSlice.setAvailableQuests(quests)
    ↓
Component: QuestList re-renders with quests
    ↓
User Clicks "Start Quest"
    ↓
API: POST /api/v1/quest/start { quest_id }
    ↓
Backend: Creates session, returns QuestSession
    ↓
Redux: questSlice.startQuest(session)
    ↓
Component: QuestSession renders with objectives
    ↓
WebSocket: Real-time objective progress updates
    ↓
User Completes All Objectives
    ↓
API: POST /api/v1/quest/complete { session_id }
    ↓
Backend: Awards XP, updates tier, grants badges
    ↓
Redux: userSlice.addXP(xpAmount)
    ↓
Component: Show completion modal with rewards
```

### Leaderboard Flow
```
User Visits Leaderboard Page
    ↓
Redux: leaderboardSlice.fetchLeaderboard(scope, region)
    ↓
API: GET /api/v1/leaderboard?scope=global&region=lekie
    ↓
Backend: Returns rankings
    ↓
Redux: leaderboardSlice.setLeaderboard(rankings)
    ↓
Component: LeaderboardTabs renders rankings
    ↓
WebSocket: Real-time rank updates when XP changes
    ↓
User's rank updates live without page refresh
```

### Payment Flow
```
User Clicks "Upgrade to Player"
    ↓
Component: TierUpgradeModal opens
    ↓
Shows: Price (5,000 XAF), features, discount
    ↓
User Confirms
    ↓
Redux: paymentSlice.initiatePayment(tier, duration)
    ↓
API: POST /api/v1/payment/tier-upgrade
    ↓
Backend: Creates transaction, returns payment_link
    ↓
Frontend: Redirects to Flutterwave/Paytech
    ↓
User Completes Payment
    ↓
Webhook: Backend receives confirmation
    ↓
Backend: Updates user tier, grants access
    ↓
Redirect: Returns to payment-result page
    ↓
Component: Shows "Upgrade Successful!"
    ↓
Redux: userSlice.updateTier(newTier)
```

---

## 🎨 UI Component Specifications

### Quest List
```
QuestList Component
├── Filter/Sort Options
│   ├── By Difficulty (Easy, Normal, Hard, Master)
│   ├── By Type (Timeline, POI Hunt, etc.)
│   └── By Region (Lékié, Douala, etc.)
├── Quest Cards (Grid Layout)
│   ├── Quest Title & Description
│   ├── Difficulty Badge & XP Reward
│   ├── Min Tier/XP requirements
│   ├── Estimated Duration
│   ├── "Start Quest" Button
│   └── Progress ring (if in-progress)
└── Active Quest Card (if any)
    ├── Time elapsed
    ├── Objectives checklist
    ├── Partial XP earned
    └── "Abandon" & "Complete" buttons
```

### Cosmetics Shop
```
CosmeticsShop Component
├── Category Tabs
│   ├── All Items
│   ├── Avatars
│   ├── Emotes
│   ├── Borders
│   ├── Titles
│   └── Effects
├── Item Grid
│   └── CosmeticCard
│       ├── Item image/preview
│       ├── Name & description
│       ├── Original price (greyed if tier discount)
│       ├── Tier discount badge (if applicable)
│       ├── Final price (red if discounted)
│       ├── "Buy" button (or "Owned" if owned)
│       └── "Equip" button (if owned)
└── Currency display (XAF)
```

### Leaderboard
```
LeaderboardTabs Component
├── Scope Tabs: Global | Regional | Weekly
├── Region Filter (if Regional selected)
├── Ranking Table
│   ├── Rank (1, 2, 3, ...)
│   ├── Username/Avatar
│   ├── XP/Points
│   ├── Tier Badge
│   ├── Completed Quests
│   └── Distance to next rank
├── Highlight: User's own rank
└── Refresh indicator (live updates)
```

### User Progress Dashboard
```
UserDashboard Component
├── Header
│   ├── User avatar & name
│   ├── Current tier badge
│   ├── Tier expiration (if paid)
│   └── "Upgrade Tier" button
├── Progress Section
│   ├── Level: 5/999
│   ├── XP progress bar: 750/1000 XP
│   └── "Next level in X XP"
├── Stats Grid
│   ├── Total XP: 4,750
│   ├── Completed Quests: 12
│   ├── Global Rank: #42
│   ├── Region Rank: #3 (Lékié)
│   └── Badges Earned: 8
├── Active Quests
│   └── List of in-progress sessions
└── Recent Achievements
    └── Latest badges earned
```

---

## 📡 API Integration

### API Client Setup
```typescript
// src/services/api.ts
const apiClient = axios.create({
  baseURL: process.env.REACT_APP_API_URL || 'http://localhost:8080/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add user ID header
apiClient.interceptors.request.use((config) => {
  const userId = localStorage.getItem('userId');
  if (userId) {
    config.headers['X-User-ID'] = userId;
  }
  return config;
});
```

### Key API Calls
```typescript
// Quests
getAvailableQuests()
startQuest(questId)
completeQuest(sessionId)
abandonQuest(sessionId)
getQuestSession(sessionId)

// User
getUserProgress()
upgradeTier(newTier, durationDays)
getLeaderboard(scope, region, limit)

// Cosmetics
getCosmeticsList()
purchaseCosmetic(cosmeticId)

// Payments
initiateTierUpgrade(tier, duration)
initiateCosmeticPurchase(cosmeticId)
verifyPayment(transactionId)
```

---

## 🔌 WebSocket Integration

### Real-time Events
```typescript
// Connect to WebSocket
const socket = io('http://localhost:8080', {
  query: { userId: localStorage.getItem('userId') },
});

// Listen for events
socket.on('quest:objective_complete', (data) => {
  // Update quest progress in Redux
});

socket.on('user:xp_gained', (data) => {
  // Update user XP & potentially level up
});

socket.on('leaderboard:rank_updated', (data) => {
  // Update user rank live
});

socket.on('payment:completed', (data) => {
  // Confirm tier upgrade without page refresh
});
```

---

## 🎬 Implementation Timeline

### Day 1: Core Setup & Navigation
- [ ] Initialize React/Vite project
- [ ] Setup Redux store with auth/quest/user slices
- [ ] Create main navigation (Header, SideNav, BottomNav)
- [ ] Setup API client with interceptors
- [ ] Create basic page routes

### Day 2: Quest & Map UI
- [ ] Implement QuestList component with filters
- [ ] Implement QuestSession component with objectives
- [ ] Integrate Leaflet.js for map display
- [ ] Show entities on map with quest markers
- [ ] Connect to backend quest API

### Day 3: Cosmetics & Leaderboard
- [ ] Implement CosmeticsShop with tier discounts
- [ ] Implement LeaderboardTabs (global/regional/weekly)
- [ ] Show user rank highlight
- [ ] Add real-time rank updates via WebSocket

### Day 4: Payment & Polish
- [ ] Integrate Flutterwave/Paytech payment flow
- [ ] TierUpgradeModal with pricing
- [ ] Payment confirmation & result pages
- [ ] UserDashboard with stats
- [ ] Responsive design polish
- [ ] Error handling & loading states

---

## ✅ Quality Checklist

- [ ] TypeScript strict mode
- [ ] Component tests (85%+ coverage)
- [ ] API integration tests
- [ ] Responsive design (mobile, tablet, desktop)
- [ ] Accessibility (WCAG 2.1 AA)
- [ ] Performance (Lighthouse 90+)
- [ ] Error boundaries & error handling
- [ ] Loading states & skeleton screens
- [ ] Form validation & user feedback
- [ ] Security: XSS prevention, CSRF tokens

---

## 🚀 Deployment

### Development
```bash
npm run dev          # Start dev server (Vite)
npm run build        # Build production bundle
npm run preview      # Preview build locally
```

### Production
```bash
npm run build
# Deploy dist/ to CDN or server
```

---

## 📋 Success Criteria

- ✅ All pages functional and responsive
- ✅ Real-time updates via WebSocket working
- ✅ Payment flow tested with Flutterwave/Paytech sandbox
- ✅ API integration complete
- ✅ 85%+ test coverage
- ✅ Performance targets met (Lighthouse 90+)
- ✅ Accessibility verified
- ✅ Ready for Phase 4 load testing

---

**Status:** 🟡 **PHASE 3 ARCHITECTURE COMPLETE — READY FOR IMPLEMENTATION**

Next: **Begin frontend implementation** 🎬

---

**Prepared:** 2026-05-11  
**Ready:** Phase 3 Frontend MVP Implementation
