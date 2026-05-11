import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface AuthState {
  userId: string | null;
  isAuthenticated: boolean;
  loading: boolean;
  error: string | null;
}

const initialState: AuthState = {
  userId: localStorage.getItem('userId'),
  isAuthenticated: !!localStorage.getItem('userId'),
  loading: false,
  error: null,
};

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    setUserId: (state, action: PayloadAction<string>) => {
      state.userId = action.payload;
      state.isAuthenticated = true;
      localStorage.setItem('userId', action.payload);
    },
    logout: (state) => {
      state.userId = null;
      state.isAuthenticated = false;
      localStorage.removeItem('userId');
    },
    setLoading: (state, action: PayloadAction<boolean>) => {
      state.loading = action.payload;
    },
    setError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },
  },
});

export const { setUserId, logout, setLoading, setError } = authSlice.actions;
export default authSlice.reducer;
