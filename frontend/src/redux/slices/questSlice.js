import { createSlice } from '@reduxjs/toolkit';
const initialState = {
    availableQuests: [],
    activeSession: null,
    loading: false,
    error: null,
};
const questSlice = createSlice({
    name: 'quest',
    initialState,
    reducers: {
        setAvailableQuests: (state, action) => {
            state.availableQuests = action.payload;
        },
        startQuest: (state, action) => {
            state.activeSession = action.payload;
        },
        updateObjective: (state, action) => {
            if (state.activeSession) {
                const objective = state.activeSession.objectives.find(o => o.id === action.payload.objectiveId);
                if (objective) {
                    objective.completed = action.payload.completed;
                }
            }
        },
        completeQuest: (state, action) => {
            if (state.activeSession && state.activeSession.session_id === action.payload.sessionId) {
                state.activeSession.status = 'completed';
                state.activeSession.xp_earned = action.payload.xpEarned;
                state.activeSession.completed_at = new Date().toISOString();
            }
        },
        abandonQuest: (state) => {
            state.activeSession = null;
        },
        setLoading: (state, action) => {
            state.loading = action.payload;
        },
        setError: (state, action) => {
            state.error = action.payload;
        },
    },
});
export const { setAvailableQuests, startQuest, updateObjective, completeQuest, abandonQuest, setLoading, setError, } = questSlice.actions;
export default questSlice.reducer;
