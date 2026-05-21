import { create } from 'zustand';
export const usePresenceStore = create((set, get) => ({
    users: new Map(),
    addUser: (user) => set((state) => {
        const newMap = new Map(state.users);
        newMap.set(user.deviceId, user);
        return { users: newMap };
    }),
    removeUser: (deviceId) => set((state) => {
        const newMap = new Map(state.users);
        newMap.delete(deviceId);
        return { users: newMap };
    }),
    updateUser: (deviceId, updates) => set((state) => {
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
