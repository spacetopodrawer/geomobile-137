import apiClient from './api';

export interface UserProgress {
  user_id: string;
  level: number;
  total_xp: number;
  tier_level: number;
  tier_expiration: string | null;
  completed_quests: number;
  global_rank: number;
  region_rank: number;
  region: string;
  badges_earned: string[];
}

export const userService = {
  async getUserProgress(): Promise<UserProgress> {
    const response = await apiClient.get('/user/progress');
    return response.data.progress;
  },

  async upgradeTier(newTier: number, durationDays: number): Promise<any> {
    const response = await apiClient.post('/user/tier-upgrade', {
      new_tier: newTier,
      duration_days: durationDays,
    });
    return response.data;
  },

  async getLeaderboard(scope: string = 'global', region?: string, limit: number = 50): Promise<any> {
    const params: any = { scope, limit };
    if (region) {
      params.region = region;
    }
    const response = await apiClient.get('/leaderboard', { params });
    return response.data;
  },
};
