import { createSlice } from '@reduxjs/toolkit';
const initialState = {
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
        initiateTierUpgrade: (state, action) => {
            state.tierUpgradePayment = action.payload;
            state.transactionId = action.payload.transaction_id;
            state.status = 'processing';
        },
        initiateCosmeticPurchase: (state, action) => {
            state.cosmeticPayment = action.payload;
            state.transactionId = action.payload.transaction_id;
            state.status = 'processing';
        },
        setPaymentStatus: (state, action) => {
            state.status = action.payload;
        },
        setPaymentMethod: (state, action) => {
            state.paymentMethod = action.payload;
        },
        completePayment: (state) => {
            state.status = 'completed';
            state.tierUpgradePayment = null;
            state.cosmeticPayment = null;
        },
        failPayment: (state, action) => {
            state.status = 'failed';
            state.error = action.payload;
        },
        resetPayment: (state) => {
            return initialState;
        },
        setLoading: (state, action) => {
            state.loading = action.payload;
        },
        setError: (state, action) => {
            state.error = action.payload;
        },
    },
});
export const { initiateTierUpgrade, initiateCosmeticPurchase, setPaymentStatus, setPaymentMethod, completePayment, failPayment, resetPayment, setLoading, setError, } = paymentSlice.actions;
export default paymentSlice.reducer;
