import React, { useState } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';

export const ActivityStream: React.FC = () => {
  const { messageHistory } = useWebSocket();
  const [isOpen, setIsOpen] = useState(true);

  return (
    <div className="fixed bottom-4 right-4 w-96 bg-white rounded-lg shadow-lg">
      <div
        className="p-3 border-b flex justify-between items-center cursor-pointer hover:bg-gray-50"
        onClick={() => setIsOpen(!isOpen)}
      >
        <h3 className="font-bold">Activity Stream</h3>
        <span className="text-xs text-gray-500">({messageHistory.length})</span>
      </div>

      {isOpen && (
        <div className="max-h-72 overflow-y-auto p-3 space-y-2 bg-gray-50">
          {messageHistory.length === 0 ? (
            <div className="text-gray-400 text-sm text-center py-8">
              Waiting for messages...
            </div>
          ) : (
            messageHistory
              .slice()
              .reverse()
              .map((msg, idx) => (
                <div
                  key={idx}
                  className="text-sm border-l-2 border-blue-400 pl-2 bg-white p-2 rounded"
                >
                  <div className="font-semibold text-xs text-blue-600">
                    {msg.type}
                  </div>
                  <div className="text-gray-700 text-xs">
                    {msg.device_id || 'server'}
                  </div>
                  <div className="text-xs text-gray-400">
                    {new Date(msg.timestamp || 0).toLocaleTimeString()}
                  </div>
                </div>
              ))
          )}
        </div>
      )}
    </div>
  );
};
