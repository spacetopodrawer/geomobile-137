import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface Cosmetic {
  cosmetic_id: string;
  name: string;
  description: string;
  category: string;
  price: number;
  original_price: number;
  discount_percentage: number;
  tier_requirement: number;
  image_url: string;
  owned: boolean;
  equipped: boolean;
}

interface CosmeticState {
  items: Cosmetic[];
  ownedItems: string[];
  equippedItems: {
    avatars: string[];
    emotes: string[];
    borders: string[];
    titles: string[];
    effects: string[];
  };
  selectedCategory: string;
  loading: boolean;
  error: string | null;
}

const initialState: CosmeticState = {
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
    setCosmeticsList: (state, action: PayloadAction<Cosmetic[]>) => {
      state.items = action.payload;
    },
    setCosmeticOwned: (state, action: PayloadAction<string[]>) => {
      state.ownedItems = action.payload;
      state.items.forEach(item => {
        item.owned = action.payload.includes(item.cosmetic_id);
      });
    },
    purchaseCosmetic: (state, action: PayloadAction<string>) => {
      if (!state.ownedItems.includes(action.payload)) {
        state.ownedItems.push(action.payload);
        const item = state.items.find(i => i.cosmetic_id === action.payload);
        if (item) {
          item.owned = true;
        }
      }
    },
    equipCosmetic: (state, action: PayloadAction<{ cosmeticId: string; category: string }>) => {
      const { cosmeticId, category } = action.payload;
      const key = category as keyof typeof state.equippedItems;

      if (state.equippedItems[key]) {
        state.equippedItems[key] = [cosmeticId];
      }

      state.items.forEach(item => {
        item.equipped = item.cosmetic_id === cosmeticId && item.category === category;
      });
    },
    setSelectedCategory: (state, action: PayloadAction<string>) => {
      state.selectedCategory = action.payload;
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
  setCosmeticsList,
  setCosmeticOwned,
  purchaseCosmetic,
  equipCosmetic,
  setSelectedCategory,
  setLoading,
  setError,
} = cosmeticSlice.actions;

export default cosmeticSlice.reducer;
