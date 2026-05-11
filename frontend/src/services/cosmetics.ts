import apiClient from './api';

export interface Cosmetic {
  cosmetic_id: string;
  name: string;
  description: string;
  category: string;
  price: number;
  tier_requirement: number;
  image_url: string;
}

export const cosmeticsService = {
  async getCosmeticsList(): Promise<Cosmetic[]> {
    const response = await apiClient.get('/cosmetics');
    return response.data.cosmetics || [];
  },

  async getUserCosmetics(): Promise<string[]> {
    const response = await apiClient.get('/user/cosmetics');
    return response.data.cosmetic_ids || [];
  },

  async purchaseCosmetic(cosmeticId: string, transactionId: string): Promise<any> {
    const response = await apiClient.post('/cosmetic/purchase', {
      cosmetic_id: cosmeticId,
      transaction_id: transactionId,
    });
    return response.data;
  },

  async equipCosmetic(cosmeticId: string, category: string): Promise<void> {
    await apiClient.post('/cosmetic/equip', {
      cosmetic_id: cosmeticId,
      category,
    });
  },
};
