import apiClient from './api';

export interface TierUpgradePayment {
  transaction_id: string;
  target_tier: number;
  amount: number;
  duration_days: number;
  payment_link: string;
}

export interface CosmeticPurchasePayment {
  transaction_id: string;
  cosmetic_id: string;
  amount: number;
  discount_amount: number;
  final_price: number;
  payment_link: string;
}

export const paymentService = {
  async initiateTierUpgrade(
    targetTier: number,
    durationDays: number,
    email: string,
    phone: string,
    redirectUrl: string
  ): Promise<TierUpgradePayment> {
    const response = await apiClient.post('/payment/tier-upgrade', {
      target_tier: targetTier,
      duration_days: durationDays,
      email,
      phone,
      redirect_url: redirectUrl,
    });
    return response.data.upgrade_payment;
  },

  async initiateCosmeticPurchase(
    cosmeticId: string,
    cosmeticName: string,
    price: number,
    email: string,
    phone: string,
    redirectUrl: string
  ): Promise<CosmeticPurchasePayment> {
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

  async verifyPayment(transactionId: string): Promise<any> {
    const response = await apiClient.get(`/payment/verify/${transactionId}`);
    return response.data;
  },
};
