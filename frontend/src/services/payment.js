import apiClient from './api';
export const paymentService = {
    async initiateTierUpgrade(targetTier, durationDays, email, phone, redirectUrl) {
        const response = await apiClient.post('/payment/tier-upgrade', {
            target_tier: targetTier,
            duration_days: durationDays,
            email,
            phone,
            redirect_url: redirectUrl,
        });
        return response.data.upgrade_payment;
    },
    async initiateCosmeticPurchase(cosmeticId, cosmeticName, price, email, phone, redirectUrl) {
        const response = await apiClient.post('/payment/cosmetic-purchase', {
            cosmetic_id: cosmeticId,
            cosmetic_name: cosmeticName,
            price,
            email,
            phone,
            redirect_url: redirectUrl,
        });
        return response.data.cosmetic_payment;
    },
    async verifyPayment(transactionId) {
        const response = await apiClient.get(`/payment/verify/${transactionId}`);
        return response.data;
    },
};
