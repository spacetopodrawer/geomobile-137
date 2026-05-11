import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface Objective {
  id: string;
  description: string;
  completed: boolean;
  xp_reward: number;
}

interface Quest {
  quest_id: string;
  title: string;
  description: string;
  difficulty: string;
  xp_reward: number;
  min_tier: number;
  min_xp: number;
  estimated_duration: number;
  region: string;
  type: string;
}

interface QuestSession {
  session_id: string;
  quest_id: string;
  user_id: string;
  status: string;
  objectives: Objective[];
  xp_earned: number;
  started_at: string;
  completed_at: string | null;
}

interface QuestState {
  availableQuests: Quest[];
  activeSession: QuestSession | null;
  loading: boolean;
  error: string | null;
}

const initialState: QuestState = {
  availableQuests: [],
  activeSession: null,
  loading: false,
  error: null,
};

const questSlice = createSlice({
  name: 'quest',
  initialState,
  reducers: {
    setAvailableQuests: (state, action: PayloadAction<Quest[]>) => {
      state.availableQuests = action.payload;
    },
    startQuest: (state, action: PayloadAction<QuestSession>) => {
      state.activeSession = action.payload;
    },
    updateObjective: (state, action: PayloadAction<{ objectiveId: string; completed: boolean }>) => {
      if (state.activeSession) {
        const objective = state.activeSession.objectives.find(o => o.id === action.payload.objectiveId);
        if (objective) {
          objective.completed = action.payload.completed;
        }
      }
    },
    completeQuest: (state, action: PayloadAction<{ sessionId: string; xpEarned: number }>) => {
      if (state.activeSession && state.activeSession.session_id === action.payload.sessionId) {
        state.activeSession.status = 'completed';
        state.activeSession.xp_earned = action.payload.xpEarned;
        state.activeSession.completed_at = new Date().toISOString();
      }
    },
    abandonQuest: (state) => {
      state.activeSession = null;
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
  setAvailableQuests,
  startQuest,
  updateObjective,
  completeQuest,
  abandonQuest,
  setLoading,
  setError,
} = questSlice.actions;

export default questSlice.reducer;
