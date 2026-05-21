import { create } from 'zustand';

export interface UserPresence {
  deviceId: string;
  userId: string;
  status: 'active' | 'idle' | 'away';
  lastSeen: number;
}

interface PresenceStore {
  users: Map<string, UserPresence>;
  addUser: (user: UserPresence) => void;
  removeUser: (deviceId: string) => void;
  updateUser: (deviceId: string, updates: Partial<UserPresence>) => void;
  getUsers: () => UserPresence[];
  isUserOnline: (deviceId: string) => boolean;
}

export const usePresenceStore = create<PresenceStore>((set, get) => ({
  users: new Map(),

  addUser: (user) =>
    set((state) => {
      const newMap = new Map(state.users);
      newMap.set(user.deviceId, user);
      return { users: newMap };
    }),

  removeUser: (deviceId) =>
    set((state) => {
      const newMap = new Map(state.users);
      newMap.delete(deviceId);
      return { users: newMap };
    }),

  updateUser: (deviceId, updates) =>
    set((state) => {
      const newMap = new Map(state.users);
      const user = newMap.get(deviceId);
      if (user) {
        newMap.set(deviceId, { ...user, ...updates });
      }
      return { users: newMap };
    }),

  getUsers: () => Array.from(get().users.values()),

  isUserOnline: (deviceId) => get().users.has(deviceId),
}));
