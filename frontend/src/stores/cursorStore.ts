import { create } from 'zustand';

export interface CursorPosition {
  deviceId: string;
  userId: string;
  x: number;
  y: number;
  color: string;
  timestamp: number;
}

interface CursorStore {
  cursors: Map<string, CursorPosition>;
  updateCursor: (cursor: CursorPosition) => void;
  removeCursor: (deviceId: string) => void;
  getCursors: () => CursorPosition[];
}

export const useCursorStore = create<CursorStore>((set, get) => ({
  cursors: new Map(),

  updateCursor: (cursor) =>
    set((state) => {
      const newMap = new Map(state.cursors);
      newMap.set(cursor.deviceId, cursor);
      return { cursors: newMap };
    }),

  removeCursor: (deviceId) =>
    set((state) => {
      const newMap = new Map(state.cursors);
      newMap.delete(deviceId);
      return { cursors: newMap };
    }),

  getCursors: () => Array.from(get().cursors.values()),
}));
