import React from 'react';
import { usePresenceStore } from '../stores/presenceStore';

export const PresenceList: React.FC = () => {
  const users = usePresenceStore((state) => state.getUsers());

  return (
    <div className="fixed left-4 top-4 w-56 bg-white rounded-lg shadow-lg p-4">
      <h3 className="font-bold text-lg mb-3">
        Online Users: {users.length}
      </h3>

      <div className="space-y-2 max-h-64 overflow-y-auto">
        {users.map((user) => (
          <div
            key={user.deviceId}
            className="flex items-center gap-2 p-2 bg-gray-50 rounded text-sm"
          >
            <div className="w-2 h-2 bg-green-500 rounded-full"></div>
            <span className="font-mono text-xs">
              {user.deviceId.slice(0, 12)}
            </span>
            <span className="text-xs text-gray-500">
              {user.status === 'active' ? '👁️' : '😴'}
            </span>
          </div>
        ))}
      </div>

      {users.length === 0 && (
        <div className="text-gray-400 text-sm">
          Waiting for other users...
        </div>
      )}
    </div>
  );
};
