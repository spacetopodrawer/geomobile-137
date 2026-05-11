import apiClient from './api';

export interface Quest {
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

export interface QuestSession {
  session_id: string;
  quest_id: string;
  user_id: string;
  status: string;
  objectives: any[];
  xp_earned: number;
  started_at: string;
  completed_at: string | null;
}

export const questService = {
  async getAvailableQuests(limit: number = 20): Promise<Quest[]> {
    const response = await apiClient.get('/quest/available', {
      params: { limit },
    });
    return response.data.quests || [];
  },

  async startQuest(questId: string): Promise<QuestSession> {
    const response = await apiClient.post('/quest/start', { quest_id: questId });
    return response.data.session;
  },

  async completeObjective(sessionId: string, objectiveId: string): Promise<void> {
    await apiClient.post('/quest/objective-complete', {
      session_id: sessionId,
      objective_id: objectiveId,
    });
  },

  async completeQuest(sessionId: string): Promise<any> {
    const response = await apiClient.post('/quest/complete', { session_id: sessionId });
    return response.data;
  },

  async abandonQuest(sessionId: string): Promise<void> {
    await apiClient.post('/quest/abandon', { session_id: sessionId });
  },

  async getQuestSession(sessionId: string): Promise<QuestSession> {
    const response = await apiClient.get(`/quest/session/${sessionId}`);
    return response.data.session;
  },
};
