import { configureStore } from '@reduxjs/toolkit';
import authReducer from './slices/authSlice';
import questReducer from './slices/questSlice';
import userReducer from './slices/userSlice';
import leaderboardReducer from './slices/leaderboardSlice';
import cosmeticReducer from './slices/cosmeticSlice';
import paymentReducer from './slices/paymentSlice';
import websocketMiddleware from './middleware/websocketMiddleware';

const store = configureStore({
  reducer: {
    auth: authReducer,
    quest: questReducer,
    user: userReducer,
    leaderboard: leaderboardReducer,
    cosmetic: cosmeticReducer,
    payment: paymentReducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware().concat(websocketMiddleware),
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;

export default store;
