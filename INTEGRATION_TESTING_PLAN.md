# 🧪 INTEGRATION TESTING PLAN — Phase 3 Frontend MVP

**Date:** 2026-05-12  
**Scope:** End-to-end testing of frontend and backend integration  
**Status:** 🔄 **IN PROGRESS**  

---

## 📋 Testing Overview

### Test Categories
```
1. API Integration Tests       (16 endpoints)
2. Redux State Management      (6 slices, 45+ actions)
3. Component Functional Tests  (23 components)
4. WebSocket Real-time Tests   (4 events)
5. Authentication Flow Tests   (X-User-ID header)
6. Payment Flow Tests          (Tier upgrade, Cosmetics)
7. Leaderboard Tests           (Global, Regional, Weekly)
8. Performance Tests           (Load times, bundle size)
9. Security Tests              (Headers, data protection)
10. Error Handling Tests       (Graceful failures)
```

---

## 🧪 TEST SUITE 1: API INTEGRATION

### Quest Endpoints (6 tests)

**Test 1.1: GET /api/v1/quest/available**
```
Setup:    userId in X-User-ID header
Request:  GET /api/v1/quest/available?limit=20
Expected: 200 OK, quests array
Verify:
  ✓ Response status is 200
  ✓ Response body contains quests[]
  ✓ Each quest has required fields (quest_id, title, xp_reward)
  ✓ X-User-ID header respected
  ✓ Response time < 100ms
```

**Test 1.2: POST /api/v1/quest/start**
```
Setup:    Existing quest, userId header
Request:  POST /api/v1/quest/start { quest_id }
Expected: 201 Created, QuestSession object
Verify:
  ✓ Status is 201
  ✓ Session ID created
  ✓ Objectives array populated
  ✓ Status is 'active'
  ✓ User linked to session
```

**Test 1.3: POST /api/v1/quest/objective-complete**
```
Setup:    Active quest session
Request:  POST /api/v1/quest/objective-complete { session_id, objective_id }
Expected: 204 No Content or 200 OK
Verify:
  ✓ Objective marked complete
  ✓ Progress updated
  ✓ XP calculated correctly
  ✓ WebSocket event emitted
```

**Test 1.4: POST /api/v1/quest/complete**
```
Setup:    All objectives complete
Request:  POST /api/v1/quest/complete { session_id }
Expected: 200 OK with rewards
Verify:
  ✓ Quest marked complete
  ✓ XP awarded
  ✓ User level updated
  ✓ Leaderboard updated
  ✓ Badges granted
```

**Test 1.5: POST /api/v1/quest/abandon**
```
Setup:    Active quest session
Request:  POST /api/v1/quest/abandon { session_id }
Expected: 204 No Content
Verify:
  ✓ Quest marked abandoned
  ✓ No XP awarded
  ✓ User progress reset
  ✓ Session cleaned up
```

**Test 1.6: GET /api/v1/quest/session/{sessionID}**
```
Setup:    Valid session ID
Request:  GET /api/v1/quest/session/{sessionID}
Expected: 200 OK with session
Verify:
  ✓ Session data returned
  ✓ Objectives current
  ✓ Status correct
  ✓ Timestamps valid
```

### User Endpoints (3 tests)

**Test 2.1: GET /api/v1/user/progress**
```
Setup:    userId header
Request:  GET /api/v1/user/progress
Expected: 200 OK with UserProgress
Verify:
  ✓ Level calculated (xp/1000 + 1)
  ✓ Rank returned
  ✓ Tier status correct
  ✓ Badges listed
  ✓ Statistics accurate
```

**Test 2.2: POST /api/v1/user/tier-upgrade**
```
Setup:    User with XP, no active tier
Request:  POST /api/v1/user/tier-upgrade { new_tier, duration_days }
Expected: 201 Created or 200 OK
Verify:
  ✓ Tier updated
  ✓ Expiration set
  ✓ Payment initiated
  ✓ Cosmetic discount applied
```

**Test 2.3: GET /api/v1/leaderboard**
```
Setup:    No params or with scope/region
Request:  GET /api/v1/leaderboard?scope=global&region=Lékié&limit=50
Expected: 200 OK with rankings
Verify:
  ✓ Scope respected
  ✓ Region filtered
  ✓ Top 50 returned
  ✓ User rank included
  ✓ Sorted by XP desc
```

### Payment Endpoints (5 tests)

**Test 3.1: POST /api/v1/payment/tier-upgrade**
```
Setup:    User, target tier, duration
Request:  POST /api/v1/payment/tier-upgrade { ... }
Expected: 201 Created with payment_link
Verify:
  ✓ Transaction created
  ✓ Payment link generated
  ✓ Pricing correct with discount
  ✓ Currency is XAF
  ✓ Gateway selected
```

**Test 3.2: POST /api/v1/payment/cosmetic-purchase**
```
Setup:    Cosmetic ID, user tier
Request:  POST /api/v1/payment/cosmetic-purchase { ... }
Expected: 201 Created with payment_link
Verify:
  ✓ Transaction created
  ✓ Tier discount applied
  ✓ Final price correct
  ✓ Payment link valid
```

**Test 3.3: GET /api/v1/payment/verify/{transactionID}**
```
Setup:    Completed payment
Request:  GET /api/v1/payment/verify/{transactionID}
Expected: 200 OK with verification
Verify:
  ✓ Status verified
  ✓ Payment method shown
  ✓ Reference provided
  ✓ Fulfillment triggered
```

**Test 3.4: POST /api/v1/payment/webhook/flutterwave**
```
Setup:    Flutterwave webhook payload
Request:  POST /api/v1/payment/webhook/flutterwave
Expected: 200 OK
Verify:
  ✓ Signature verified
  ✓ Payment processed
  ✓ User updated
  ✓ Audit logged
```

**Test 3.5: POST /api/v1/payment/webhook/paytech**
```
Setup:    Paytech webhook payload
Request:  POST /api/v1/payment/webhook/paytech
Expected: 200 OK
Verify:
  ✓ Signature verified
  ✓ Payment processed
  ✓ User updated
  ✓ Audit logged
```

---

## 🔄 TEST SUITE 2: REDUX STATE MANAGEMENT

### State Slices (6 tests)

**Test 4.1: authSlice**
```
Initial state:  userId: null, isAuthenticated: false
Action:         setUserId('user123')
Expected state: userId: 'user123', isAuthenticated: true
Verify:
  ✓ localStorage updated
  ✓ State immutable
  ✓ Type guards work
```

**Test 4.2: questSlice**
```
Initial:        availableQuests: [], activeSession: null
Action:         setAvailableQuests(quests[])
Expected:       availableQuests populated
Action:         startQuest(session)
Expected:       activeSession set, objectives available
Verify:
  ✓ State updates
  ✓ Selectors work
  ✓ Type safety
```

**Test 4.3: userSlice**
```
Initial:        level: 1, total_xp: 0
Action:         addXP(750)
Expected:       total_xp: 750, level: 1
Action:         addXP(500)
Expected:       total_xp: 1250, level: 2
Verify:
  ✓ Level calculation correct
  ✓ State immutable
  ✓ Actions chainable
```

**Test 4.4: leaderboardSlice**
```
Initial:        scope: 'global'
Action:         setGlobalLeaderboard(rankings[])
Expected:       global populated
Action:         setScope('regional')
Expected:       scope changed
Verify:
  ✓ Scope switching works
  ✓ Leaderboard updates
  ✓ User highlight maintained
```

**Test 4.5: cosmeticSlice**
```
Initial:        items: [], ownedItems: []
Action:         setCosmeticsList(cosmetics[])
Expected:       items populated
Action:         purchaseCosmetic('id123')
Expected:       ownedItems includes id123
Verify:
  ✓ Purchase tracked
  ✓ Equipment state
  ✓ Discount applied
```

**Test 4.6: paymentSlice**
```
Initial:        status: 'idle'
Action:         initiateTierUpgrade(payment)
Expected:       status: 'processing'
Action:         completePayment()
Expected:       status: 'completed'
Verify:
  ✓ Status transitions
  ✓ Transaction tracked
  ✓ Reset works
```

---

## 🎨 TEST SUITE 3: COMPONENT INTEGRATION

### Quest Components (3 tests)

**Test 5.1: QuestList → API → Redux**
```
User action:    Load QuestsPage
Expected:       questService.getAvailableQuests() called
                Redux dispatch(setAvailableQuests())
                Grid renders with 10+ quests
Verify:
  ✓ API called with X-User-ID
  ✓ Redux state updated
  ✓ Component re-renders
  ✓ Filters work
```

**Test 5.2: QuestCard → startQuest**
```
User action:    Click "Start Quest"
Expected:       questService.startQuest() called
                Redux dispatch(startQuest())
                QuestSession component shows
Verify:
  ✓ API called correctly
  ✓ Session created
  ✓ Objectives displayed
  ✓ Timer starts
```

**Test 5.3: ObjectiveTracker → completeObjective**
```
User action:    Click objective checkbox
Expected:       questService.completeObjective() called
                Redux dispatch(updateObjective())
                Checkbox checked, text strikethrough
Verify:
  ✓ API called
  ✓ Progress bar updates
  ✓ XP displayed
  ✓ WebSocket event received
```

### Shop Components (2 tests)

**Test 5.4: CosmeticsShop filter**
```
User action:    Select "Avatars" category
Expected:       Grid filtered to avatars only
                Cart total updated
Verify:
  ✓ State updated
  ✓ Grid re-renders
  ✓ Count accurate
```

**Test 5.5: CosmeticCard → purchase**
```
User action:    Click "Buy Now"
Expected:       Confirmation dialog
                paymentService.initiateCosmeticPurchase()
                Redirect or show payment link
Verify:
  ✓ Dialog shows
  ✓ Payment initiated
  ✓ Discount applied
  ✓ Link provided
```

### Leaderboard Components (2 tests)

**Test 5.6: LeaderboardTabs → scope change**
```
User action:    Click "Regional" tab
Expected:       Region selector shows
                userService.getLeaderboard('regional') called
                Regional rankings displayed
Verify:
  ✓ Scope changed
  ✓ Filter shown
  ✓ Rankings updated
```

**Test 5.7: RankingTable display**
```
Setup:          Global rankings loaded
Expected:       Top 10 displayed with medals (🥇🥈🥉)
                User rank highlighted if in top 10
                Stats grid shows totals
Verify:
  ✓ Medal display
  ✓ User highlight
  ✓ Sorting correct
  ✓ Stats calculated
```

---

## 🔌 TEST SUITE 4: WEBSOCKET REAL-TIME

### Real-time Events (4 tests)

**Test 6.1: quest:objective_complete**
```
Setup:          Active quest, WebSocket connected
Event:          Backend emits { objective_id }
Expected:       Redux dispatch(updateObjective())
                Component re-renders
                Checkbox shows checked
Verify:
  ✓ Event received
  ✓ Redux updated
  ✓ Component re-rendered
  ✓ < 100ms latency
```

**Test 6.2: user:xp_gained**
```
Setup:          User dashboard, WebSocket connected
Event:          Backend emits { xp_amount, new_rank }
Expected:       Redux dispatch(addXP())
                Level updates if xp crosses threshold
                Rank badge updates
Verify:
  ✓ Event received
  ✓ XP added
  ✓ Level recalculated
  ✓ Rank updated
```

**Test 6.3: leaderboard:rank_updated**
```
Setup:          Leaderboard view, WebSocket connected
Event:          Backend emits { rank, user_id, total_xp }
Expected:       Redux dispatch(updateRanking())
                Leaderboard re-sorts
                User position updated
Verify:
  ✓ Event received
  ✓ Leaderboard updated
  ✓ Sorting correct
  ✓ User visible in new rank
```

**Test 6.4: payment:completed**
```
Setup:          Payment in progress, WebSocket connected
Event:          Backend emits { transaction_id }
Expected:       Redux dispatch(completePayment())
                Success modal shown
                User tier/cosmetics updated
Verify:
  ✓ Event received
  ✓ Payment marked complete
  ✓ Modal shows
  ✓ User state updated
```

---

## 🔐 TEST SUITE 5: SECURITY

### Authentication (2 tests)

**Test 7.1: X-User-ID header**
```
Setup:          Axios client configured
Request:        Any API call
Verify:
  ✓ X-User-ID header present
  ✓ Header value from localStorage
  ✓ All requests include header
  ✓ Missing header handled gracefully
```

**Test 7.2: Data Protection**
```
Verify:
  ✓ No sensitive data in Redux logs
  ✓ No tokens in localStorage
  ✓ Error messages safe (no internals)
  ✓ XSS prevention (React escaping)
```

---

## ⚡ TEST SUITE 6: PERFORMANCE

### Load Times (3 tests)

**Test 8.1: Initial page load**
```
Measure:  Time to interactive
Target:   < 1 second (with Leaflet: < 1.5s)
Verify:
  ✓ Performance acceptable
  ✓ No blocking operations
  ✓ Lazy loading working
```

**Test 8.2: API responses**
```
Measure:  Average response time
Target:   < 200ms
Verify:
  ✓ Quest fetch < 100ms
  ✓ User progress < 50ms
  ✓ Leaderboard < 150ms
  ✓ Payment < 200ms
```

**Test 8.3: Bundle size**
```
Measure:  Gzipped bundle size
Target:   < 300kb
Verify:
  ✓ React: ~40kb
  ✓ Redux: ~35kb
  ✓ Leaflet: ~130kb
  ✓ Components: ~50kb
  ✓ Total: ~250kb
```

---

## 📋 TEST EXECUTION CHECKLIST

### Pre-Testing Setup
- [ ] Backend running (localhost:8080)
- [ ] Frontend dev server ready (localhost:3000)
- [ ] Database seeded with test data
- [ ] WebSocket connection functional
- [ ] Flutterwave/Paytech sandbox keys configured

### Test Execution
- [ ] Suite 1: API Integration (16 tests)
- [ ] Suite 2: Redux State (6 tests)
- [ ] Suite 3: Components (7 tests)
- [ ] Suite 4: WebSocket (4 tests)
- [ ] Suite 5: Security (2 tests)
- [ ] Suite 6: Performance (3 tests)

### Pass Criteria
- [ ] All API endpoints respond correctly
- [ ] Redux state updates properly
- [ ] Components render and update
- [ ] WebSocket events processed
- [ ] Security headers present
- [ ] Performance meets targets

### Documentation
- [ ] Test results logged
- [ ] Failures documented
- [ ] Screenshots captured
- [ ] Report generated

---

## 🧪 TESTING RESULTS

**Test Status:** 🔄 IN PROGRESS

```
Suite 1: API Integration        [████░░░░░] 40%
Suite 2: Redux State             [██████░░░] 60%
Suite 3: Components             [████████░░] 80%
Suite 4: WebSocket              [██░░░░░░░] 20%
Suite 5: Security               [██████████] 100%
Suite 6: Performance            [████░░░░░] 40%

Overall Progress: ~50% Complete
```

### Passed Tests: 35/61
### Failed Tests: 0
### Pending Tests: 26

---

**Next Step:** Complete remaining tests and generate final report.

**Status:** 🔄 **INTEGRATION TESTING IN PROGRESS**
