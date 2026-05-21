import { createSlice } from '@reduxjs/toolkit';
const initialState = {
    userId: localStorage.getItem('userId'),
    isAuthenticated: !!localStorage.getItem('userId'),
    loading: false,
    error: null,
};
const authSlice = createSlice({
    name: 'auth',
    initialState,
    reducers: {
        setUserId: (state, action) => {
            state.userId = action.payload;
            state.isAuthenticated = true;
            localStorage.setItem('userId', action.payload);
        },
        logout: (state) => {
            state.userId = null;
            state.isAuthenticated = false;
            localStorage.removeItem('userId');
        },
        setLoading: (state, action) => {
            state.loading = action.payload;
        },
        setError: (state, action) => {
            state.error = action.payload;
        },
    },
});
export const { setUserId, logout, setLoading, setError } = authSlice.actions;
export default authSlice.reducer;
