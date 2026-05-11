import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface RankingEntry {
  rank: number;
  user_id: string;
  username: string;
  total_xp: number;
  tier_level: number;
  completed_quests: number;
}

interface LeaderboardState {
  global: RankingEntry[];
  regional: RankingEntry[];
  weekly: RankingEntry[];
  scope: 'global' | 'regional' | 'weekly';
  region: string;
  loading: boolean;
  error: string | null;
  lastUpdated: string | null;
}

const initialState: LeaderboardState = {
  global: [],
  regional: [],
  weekly: [],
  scope: 'global',
  region: '',
  loading: false,
  error: null,
  lastUpdated: null,
};

const leaderboardSlice = createSlice({
  name: 'leaderboard',
  initialState,
  reducers: {
    setGlobalLeaderboard: (state, action: PayloadAction<RankingEntry[]>) => {
      state.global = action.payload;
      state.lastUpdated = new Date().toISOString();
    },
    setRegionalLeaderboard: (state, action: PayloadAction<RankingEntry[]>) => {
      state.regional = action.payload;
      state.lastUpdated = new Date().toISOString();
    },
    setWeeklyLeaderboard: (state, action: PayloadAction<RankingEntry[]>) => {
      state.weekly = action.payload;
      state.lastUpdated = new Date().toISOString();
    },
    setScope: (state, action: PayloadAction<'global' | 'regional' | 'weekly'>) => {
      state.scope = action.payload;
    },
    setRegion: (state, action: PayloadAction<string>) => {
      state.region = action.payload;
    },
    updateRanking: (state, action: PayloadAction<RankingEntry>) => {
      const entry = action.payload;
      const scopes = ['global', 'regional', 'weekly'] as const;

      scopes.forEach(scope => {
        const leaderboard = state[scope];
        const index = leaderboard.findIndex(e => e.user_id === entry.user_id);
        if (index !== -1) {
          leaderboard[index] = entry;
          leaderboard.sort((a, b) => b.total_xp - a.total_xp);
          leaderboard.forEach((e, i) => {
            e.rank = i + 1;
          });
        }
      });
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
  setGlobalLeaderboard,
  setRegionalLeaderboard,
  setWeeklyLeaderboard,
  setScope,
  setRegion,
  updateRanking,
  setLoading,
  setError,
} = leaderboardSlice.actions;

export default leaderboardSlice.reducer;
