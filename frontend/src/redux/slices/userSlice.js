import { createSlice } from '@reduxjs/toolkit';
const initialState = {
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
        setUserProgress: (state, action) => {
            return { ...state, ...action.payload };
        },
        addXP: (state, action) => {
            state.total_xp += action.payload;
            state.level = Math.floor(state.total_xp / 1000) + 1;
        },
        incrementQuestCompletion: (state) => {
            state.completed_quests += 1;
        },
        updateTier: (state, action) => {
            state.tier_level = action.payload.tier;
            state.tier_expiration = action.payload.expiration;
        },
        updateRanks: (state, action) => {
            state.global_rank = action.payload.globalRank;
            state.region_rank = action.payload.regionRank;
        },
        addBadge: (state, action) => {
            if (!state.badges_earned.includes(action.payload)) {
                state.badges_earned.push(action.payload);
            }
        },
        setLoading: (state, action) => {
            state.loading = action.payload;
        },
        setError: (state, action) => {
            state.error = action.payload;
        },
    },
});
export const { setUserProgress, addXP, incrementQuestCompletion, updateTier, updateRanks, addBadge, setLoading, setError, } = userSlice.actions;
export default userSlice.reducer;
