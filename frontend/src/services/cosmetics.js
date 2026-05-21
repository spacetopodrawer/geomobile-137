import apiClient from './api';
export const cosmeticsService = {
    async getCosmeticsList() {
        const response = await apiClient.get('/cosmetics');
        return response.data.cosmetics || [];
    },
    async getUserCosmetics() {
        const response = await apiClient.get('/user/cosmetics');
        return response.data.cosmetic_ids || [];
    },
    async purchaseCosmetic(cosmeticId, transactionId) {
        const response = await apiClient.post('/cosmetic/purchase', {
            cosmetic_id: cosmeticId,
            transaction_id: transactionId,
        });
        return response.data;
    },
    async equipCosmetic(cosmeticId, category) {
        await apiClient.post('/cosmetic/equip', {
            cosmetic_id: cosmeticId,
            category,
        });
    },
};
