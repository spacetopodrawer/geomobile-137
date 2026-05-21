import { create } from 'zustand';

export interface WebSocketMessage {
  type: string;
  device_id?: string;
  user_id?: string;
  payload?: Record<string, any>;
  timestamp?: number;
}

interface WebSocketStore {
  ws: WebSocket | null;
  connected: boolean;
  connecting: boolean;
  error: string | null;
  deviceId: string;
  userId: string;
  messageHistory: WebSocketMessage[];
  connect: () => void;
  disconnect: () => void;
  send: (message: WebSocketMessage) => void;
  setError: (error: string | null) => void;
  addMessage: (message: WebSocketMessage) => void;
}

export const useWebSocketStore = create<WebSocketStore>((set, get) => ({
  ws: null,
  connected: false,
  connecting: false,
  error: null,
  deviceId: `device-${Math.random().toString(36).substr(2, 8)}`,
  userId: `user-${Math.random().toString(36).substr(2, 8)}`,
  messageHistory: [],

  connect: () => {
    set({ connecting: true, error: null });

    const { deviceId, userId } = get();
    const wsBase = import.meta.env.VITE_WS_URL ?? `ws://${window.location.host}`;
    const url = `${wsBase}/ws/geometry?device_id=${deviceId}&user_id=${userId}`;

    try {
      const ws = new WebSocket(url);

      ws.onopen = () => {
        console.log('✅ WebSocket connected');
        set({ connected: true, connecting: false, ws });
      };

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          get().addMessage(message);
          console.log('📨 Message received:', message.type || message.Type);
        } catch (e) {
          console.error('Failed to parse message:', e);
        }
      };

      ws.onerror = (event) => {
        const errorMsg = `WebSocket error`;
        console.error(errorMsg, event);
        set({ error: errorMsg, connected: false, connecting: false });
      };

      ws.onclose = () => {
        console.log('🔌 WebSocket disconnected');
        set({ connected: false, ws: null });
      };
    } catch (e) {
      const errorMsg = `Connection failed: ${e}`;
      set({ error: errorMsg, connecting: false });
    }
  },

  disconnect: () => {
    const { ws } = get();
    if (ws) {
      ws.close();
    }
    set({ connected: false, ws: null });
  },

  send: (message) => {
    const { ws, connected } = get();
    if (ws && connected) {
      ws.send(JSON.stringify(message));
      console.log('📤 Message sent:', message.type);
    } else {
      console.warn('WebSocket not connected');
    }
  },

  setError: (error) => set({ error }),

  addMessage: (message) => {
    set((state) => ({
      messageHistory: [...state.messageHistory, message].slice(-100),
    }));
  },
}));

export const useWebSocket = () => useWebSocketStore();
