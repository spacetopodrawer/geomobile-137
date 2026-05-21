import apiClient from './api';
export const userService = {
    async getUserProgress() {
        const response = await apiClient.get('/user/progress');
        return response.data.progress;
    },
    async upgradeTier(newTier, durationDays) {
        const response = await apiClient.post('/user/tier-upgrade', {
            new_tier: newTier,
            duration_days: durationDays,
        });
        return response.data;
    },
    async getLeaderboard(scope = 'global', region, limit = 50) {
        const params = { scope, limit };
        if (region) {
            params.region = region;
        }
        const response = await apiClient.get('/leaderboard', { params });
        return response.data;
    },
};
