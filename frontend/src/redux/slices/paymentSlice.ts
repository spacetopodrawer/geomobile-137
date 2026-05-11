import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface TierUpgradePayment {
  transaction_id: string;
  target_tier: number;
  amount: number;
  duration_days: number;
  payment_link: string;
}

interface CosmeticPurchasePayment {
  transaction_id: string;
  cosmetic_id: string;
  amount: number;
  discount_amount: number;
  final_price: number;
  payment_link: string;
}

interface PaymentState {
  tierUpgradePayment: TierUpgradePayment | null;
  cosmeticPayment: CosmeticPurchasePayment | null;
  transactionId: string | null;
  status: 'idle' | 'processing' | 'completed' | 'failed';
  paymentMethod: string | null;
  loading: boolean;
  error: string | null;
}

const initialState: PaymentState = {
  tierUpgradePayment: null,
  cosmeticPayment: null,
  transactionId: null,
  status: 'idle',
  paymentMethod: null,
  loading: false,
  error: null,
};

const paymentSlice = createSlice({
  name: 'payment',
  initialState,
  reducers: {
    initiateTierUpgrade: (state, action: PayloadAction<TierUpgradePayment>) => {
      state.tierUpgradePayment = action.payload;
      state.transactionId = action.payload.transaction_id;
      state.status = 'processing';
    },
    initiateCosmeticPurchase: (state, action: PayloadAction<CosmeticPurchasePayment>) => {
      state.cosmeticPayment = action.payload;
      state.transactionId = action.payload.transaction_id;
      state.status = 'processing';
    },
    setPaymentStatus: (state, action: PayloadAction<'idle' | 'processing' | 'completed' | 'failed'>) => {
      state.status = action.payload;
    },
    setPaymentMethod: (state, action: PayloadAction<string>) => {
      state.paymentMethod = action.payload;
    },
    completePayment: (state) => {
      state.status = 'completed';
      state.tierUpgradePayment = null;
      state.cosmeticPayment = null;
    },
    failPayment: (state, action: PayloadAction<string>) => {
      state.status = 'failed';
      state.error = action.payload;
    },
    resetPayment: (state) => {
      return initialState;
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
  initiateTierUpgrade,
  initiateCosmeticPurchase,
  setPaymentStatus,
  setPaymentMethod,
  completePayment,
  failPayment,
  resetPayment,
  setLoading,
  setError,
} = paymentSlice.actions;

export default paymentSlice.reducer;
