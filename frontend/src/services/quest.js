import apiClient from './api';
export const questService = {
    async getAvailableQuests(limit = 20) {
        const response = await apiClient.get('/quest/available', {
            params: { limit },
        });
        return response.data.quests || [];
    },
    async startQuest(questId) {
        const response = await apiClient.post('/quest/start', { quest_id: questId });
        return response.data.session;
    },
    async completeObjective(sessionId, objectiveId) {
        await apiClient.post('/quest/objective-complete', {
            session_id: sessionId,
            objective_id: objectiveId,
        });
    },
    async completeQuest(sessionId) {
        const response = await apiClient.post('/quest/complete', { session_id: sessionId });
        return response.data;
    },
    async abandonQuest(sessionId) {
        await apiClient.post('/quest/abandon', { session_id: sessionId });
    },
    async getQuestSession(sessionId) {
        const response = await apiClient.get(`/quest/session/${sessionId}`);
        return response.data.session;
    },
};
