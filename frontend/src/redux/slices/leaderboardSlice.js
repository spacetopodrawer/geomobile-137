import { createSlice } from '@reduxjs/toolkit';
const initialState = {
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
        setGlobalLeaderboard: (state, action) => {
            state.global = action.payload;
            state.lastUpdated = new Date().toISOString();
        },
        setRegionalLeaderboard: (state, action) => {
            state.regional = action.payload;
            state.lastUpdated = new Date().toISOString();
        },
        setWeeklyLeaderboard: (state, action) => {
            state.weekly = action.payload;
            state.lastUpdated = new Date().toISOString();
        },
        setScope: (state, action) => {
            state.scope = action.payload;
        },
        setRegion: (state, action) => {
            state.region = action.payload;
        },
        updateRanking: (state, action) => {
            const entry = action.payload;
            const scopes = ['global', 'regional', 'weekly'];
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
        setLoading: (state, action) => {
            state.loading = action.payload;
        },
        setError: (state, action) => {
            state.error = action.payload;
        },
    },
});
export const { setGlobalLeaderboard, setRegionalLeaderboard, setWeeklyLeaderboard, setScope, setRegion, updateRanking, setLoading, setError, } = leaderboardSlice.actions;
export default leaderboardSlice.reducer;
