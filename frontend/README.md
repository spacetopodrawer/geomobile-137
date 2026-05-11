# Geo-Mobile137 Frontend MVP

React-based web frontend for the Geo-Mobile137 cadastral modernization system.

## 🚀 Quick Start

### Prerequisites
- Node.js 18+
- npm 8+

### Installation

```bash
cd frontend
npm install
```

### Development

```bash
npm run dev
```

Server starts at `http://localhost:3000`

API proxy configured for `http://localhost:8080/api/v1`

### Build

```bash
npm run build
```

Production bundle in `dist/` directory

### Preview

```bash
npm run preview
```

## 📁 Project Structure

```
src/
├── components/          # React components by feature
│   ├── Common/         # Shared components (Loading, ErrorBoundary)
│   ├── Navigation/     # Header, SideNav, BottomNav
│   ├── QuestUI/        # Quest listing and tracking
│   ├── MapUI/          # Leaflet map integration
│   ├── ShopUI/         # Cosmetics shop
│   ├── LeaderboardUI/  # Rankings display
│   ├── UserDashboard/  # User progression
│   └── PaymentUI/      # Payment flows
├── pages/              # Page components
├── redux/              # Redux store, slices, middleware
├── services/           # API client services
├── hooks/              # Custom React hooks
├── utils/              # Utilities, constants, formatters
└── styles/             # Global CSS/Tailwind
```

## 🔌 API Integration

API client automatically adds `X-User-ID` header for all requests.

### Key Endpoints

- `GET /api/v1/quest/available` — Available quests
- `POST /api/v1/quest/start` — Start quest session
- `POST /api/v1/quest/complete` — Complete quest
- `GET /api/v1/user/progress` — User progression
- `GET /api/v1/leaderboard` — Rankings
- `POST /api/v1/payment/tier-upgrade` — Tier upgrade
- `POST /api/v1/cosmetics` — Cosmetics list

## 🎨 Features Implemented (Phase 3 Day 1)

✅ Redux state management with 6 slices (auth, quest, user, leaderboard, cosmetic, payment)
✅ Core navigation and layout
✅ API client with interceptors
✅ WebSocket integration for real-time updates
✅ Custom hooks for quests, users, leaderboards
✅ Global Tailwind CSS styling
✅ Error boundaries and loading states
✅ TypeScript strict mode

## 📝 Configuration

Create `.env` file:

```
VITE_API_URL=http://localhost:8080/api/v1
VITE_WEBSOCKET_URL=http://localhost:8080
```

## 🧪 Testing

Run tests with:

```bash
npm run test
```

## 📚 Technology Stack

- **Framework:** React 18
- **State Management:** Redux Toolkit
- **Routing:** React Router v6
- **Styling:** Tailwind CSS
- **Maps:** Leaflet.js + React Leaflet
- **Charts:** Recharts
- **UI Components:** Shadcn/UI
- **Build Tool:** Vite
- **Language:** TypeScript
- **Real-time:** Socket.io Client

## 🔐 Security

- X-User-ID header for authentication
- No sensitive data in localStorage except userId
- HTTPS ready for production
- CORS properly configured

## 📊 Performance

- Code splitting with React Router
- Lazy loading for heavy components
- Redux selectors for optimal re-renders
- Leaflet tile caching
- WebSocket for real-time updates

## 🚢 Deployment

```bash
npm run build
# Deploy dist/ to CDN or static hosting
```

## 📋 Remaining Tasks (Phase 3)

- [ ] Day 2: Quest UI, Map integration
- [ ] Day 3: Cosmetics shop, Leaderboard display
- [ ] Day 4: Payment flows, Polish, Responsive design

## 💡 Contributing

Follow TypeScript strict mode, use Redux selectors, test components with Vitest.

## 📄 License

MIT
