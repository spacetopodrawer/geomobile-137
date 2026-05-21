import { create } from 'zustand';
export const useCursorStore = create((set, get) => ({
    cursors: new Map(),
    updateCursor: (cursor) => set((state) => {
        const newMap = new Map(state.cursors);
        newMap.set(cursor.deviceId, cursor);
        return { cursors: newMap };
    }),
    removeCursor: (deviceId) => set((state) => {
        const newMap = new Map(state.cursors);
        newMap.delete(deviceId);
        return { cursors: newMap };
    }),
    getCursors: () => Array.from(get().cursors.values()),
}));
