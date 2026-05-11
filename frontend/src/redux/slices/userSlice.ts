import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface UserState {
  userId: string | null;
  level: number;
  total_xp: number;
  tier_level: number;
  tier_expiration: string | null;
  completed_quests: number;
  global_rank: number;
  region_rank: number;
  region: string;
  badges_earned: string[];
  loading: boolean;
  error: string | null;
}

const initialState: UserState = {
  userId: null,
  level: 1,
  total_xp: 0,
  tier_level: 0,
  tier_expiration: null,
  completed_quests: 0,
  global_rank: 0,
  region_rank: 0,
  region: '',
  badges_earned: [],
  loading: false,
  error: null,
};

const userSlice = createSlice({
  name: 'user',
  initialState,
  reducers: {
    setUserProgress: (state, action: PayloadAction<Partial<UserState>>) => {
      return { ...state, ...action.payload };
    },
    addXP: (state, action: PayloadAction<number>) => {
      state.total_xp += action.payload;
      state.level = Math.floor(state.total_xp / 1000) + 1;
    },
    incrementQuestCompletion: (state) => {
      state.completed_quests += 1;
    },
    updateTier: (state, action: PayloadAction<{ tier: number; expiration: string }>) => {
      state.tier_level = action.payload.tier;
      state.tier_expiration = action.payload.expiration;
    },
    updateRanks: (state, action: PayloadAction<{ globalRank: number; regionRank: number }>) => {
      state.global_rank = action.payload.globalRank;
      state.region_rank = action.payload.regionRank;
    },
    addBadge: (state, action: PayloadAction<string>) => {
      if (!state.badges_earned.includes(action.payload)) {
        state.badges_earned.push(action.payload);
      }
    },
    setLoading: (state, action: PayloadAction<boolean>) => {
      state.loading = action.payload;
    },
    setError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },
  },
});

export const {
  setUserProgress,
  addXP,
  incrementQuestCompletion,
  updateTier,
  updateRanks,
  addBadge,
  setLoading,
  setError,
} = userSlice.actions;

export default userSlice.reducer;
