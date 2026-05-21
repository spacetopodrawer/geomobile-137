import { createSlice } from '@reduxjs/toolkit';
const initialState = {
    items: [],
    ownedItems: [],
    equippedItems: {
        avatars: [],
        emotes: [],
        borders: [],
        titles: [],
        effects: [],
    },
    selectedCategory: 'all',
    loading: false,
    error: null,
};
const cosmeticSlice = createSlice({
    name: 'cosmetic',
    initialState,
    reducers: {
        setCosmeticsList: (state, action) => {
            state.items = action.payload;
        },
        setCosmeticOwned: (state, action) => {
            state.ownedItems = action.payload;
            state.items.forEach(item => {
                item.owned = action.payload.includes(item.cosmetic_id);
            });
        },
        purchaseCosmetic: (state, action) => {
            if (!state.ownedItems.includes(action.payload)) {
                state.ownedItems.push(action.payload);
                const item = state.items.find(i => i.cosmetic_id === action.payload);
                if (item) {
                    item.owned = true;
                }
            }
        },
        equipCosmetic: (state, action) => {
            const { cosmeticId, category } = action.payload;
            const key = category;
            if (state.equippedItems[key]) {
                state.equippedItems[key] = [cosmeticId];
            }
            state.items.forEach(item => {
                item.equipped = item.cosmetic_id === cosmeticId && item.category === category;
            });
        },
        setSelectedCategory: (state, action) => {
            state.selectedCategory = action.payload;
        },
        setLoading: (state, action) => {
            state.loading = action.payload;
        },
        setError: (state, action) => {
            state.error = action.payload;
        },
    },
});
export const { setCosmeticsList, setCosmeticOwned, purchaseCosmetic, equipCosmetic, setSelectedCategory, setLoading, setError, } = cosmeticSlice.actions;
export default cosmeticSlice.reducer;
