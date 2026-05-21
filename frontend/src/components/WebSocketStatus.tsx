import React from 'react';
import { useWebSocket } from '../hooks/useWebSocket';

export const WebSocketStatus: React.FC = () => {
  const store = useWebSocket();
  const { connected, connecting, error, deviceId } = store;

  return (
    <div className="fixed top-4 right-4 p-4 rounded-lg shadow-lg max-w-xs bg-white">
      {connecting && (
        <div className="bg-yellow-100 text-yellow-900 p-2 rounded">
          ⟳ Connecting...
        </div>
      )}

      {connected && (
        <div className="bg-green-100 text-green-900 p-2 rounded">
          ✅ Connected ({deviceId.slice(0, 8)})
        </div>
      )}

      {error && (
        <div className="bg-red-100 text-red-900 p-2 rounded">
          ❌ Error: {error}
        </div>
      )}

      {!connected && !connecting && !error && (
        <div className="bg-gray-100 text-gray-900 p-2 rounded">
          ⚫ Disconnected
        </div>
      )}
    </div>
  );
};
