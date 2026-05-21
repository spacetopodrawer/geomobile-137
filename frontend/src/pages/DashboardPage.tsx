import React, { useState } from 'react';
import { useDispatch } from 'react-redux';
import { logout } from '../redux/slices/authSlice';
import { useWebSocket } from '../hooks/useWebSocket';

type Tab = 'realtime' | 'parcels' | 'devices';

const RealtimeTab: React.FC = () => {
  const { connected, messageHistory, deviceId, userId, send } = useWebSocket();

  const sendTestBroadcast = () => {
    send({ type: 'broadcast', payload: { msg: 'Test depuis dashboard', ts: Date.now() } });
  };

  const sendTestCursor = () => {
    send({ type: 'cursor', payload: { x: Math.random(), y: Math.random() } });
  };

  return (
    <div className="space-y-4">
      {/* Status bar */}
      <div className="flex items-center gap-4 p-3 bg-white rounded-lg border text-sm">
        <span className={`font-semibold ${connected ? 'text-green-600' : 'text-red-600'}`}>
          {connected ? '🟢 Connecté' : '🔴 Déconnecté'}
        </span>
        {deviceId && <span className="text-gray-500 font-mono">device: <strong>{deviceId}</strong></span>}
        {userId   && <span className="text-gray-500 font-mono">user: <strong>{userId}</strong></span>}
        {connected && (
          <div className="ml-auto flex gap-2">
            <button onClick={sendTestBroadcast}
              className="px-3 py-1 text-xs border rounded hover:bg-gray-50">
              📢 Broadcast
            </button>
            <button onClick={sendTestCursor}
              className="px-3 py-1 text-xs border rounded hover:bg-gray-50">
              🖱️ Curseur
            </button>
          </div>
        )}
      </div>

      {/* Message log */}
      <div className="bg-white rounded-lg border overflow-hidden">
        <div className="px-4 py-2 border-b bg-gray-50 flex items-center justify-between">
          <span className="font-semibold text-sm">📡 Flux temps réel</span>
          <span className="text-xs text-gray-400">{messageHistory.length} message(s)</span>
        </div>
        <div className="max-h-96 overflow-y-auto font-mono text-xs">
          {messageHistory.length === 0 ? (
            <div className="text-center text-gray-400 py-12">
              {connected ? '⏳ En attente de messages…' : '🔌 WebSocket déconnecté'}
            </div>
          ) : (
            [...messageHistory].reverse().map((msg, i) => (
              <div key={i} className="px-4 py-2 border-b last:border-b-0 hover:bg-gray-50">
                <span className="text-blue-600 font-bold">{msg.type}</span>
                {msg.device_id && <span className="text-gray-500 ml-2">· {msg.device_id}</span>}
                {msg.payload && (
                  <span className="text-gray-400 ml-2">{JSON.stringify(msg.payload)}</span>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
};

const DashboardPage: React.FC = () => {
  const dispatch = useDispatch();
  const [activeTab, setActiveTab] = useState<Tab>('realtime');

  const tabs: { id: Tab; label: string; icon: string }[] = [
    { id: 'realtime', label: 'Temps Réel', icon: '📡' },
    { id: 'parcels',  label: 'Parcelles',  icon: '📋' },
    { id: 'devices',  label: 'Appareils',  icon: '📱' },
  ];

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Header */}
      <header className="bg-white border-b px-6 py-3 flex items-center justify-between">
        <div>
          <h1 className="text-lg font-bold text-gray-800">🗺️ GeoMobile137</h1>
          <p className="text-xs text-gray-500">Cadastral Management — Phase 3</p>
        </div>
        <button
          onClick={() => dispatch(logout())}
          className="text-sm text-gray-500 hover:text-red-600 border px-3 py-1 rounded"
        >
          Déconnexion
        </button>
      </header>

      {/* Tabs */}
      <div className="bg-white border-b px-6">
        <div className="flex gap-1">
          {tabs.map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors
                ${activeTab === tab.id
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-800'}`}
            >
              {tab.icon} {tab.label}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <main className="max-w-5xl mx-auto p-6">
        {activeTab === 'realtime' && <RealtimeTab />}
        {activeTab === 'parcels'  && (
          <div className="bg-white rounded-lg border p-8 text-center text-gray-400">
            📋 Gestion des parcelles — à implémenter (Phase 3B)
          </div>
        )}
        {activeTab === 'devices'  && (
          <div className="bg-white rounded-lg border p-8 text-center text-gray-400">
            📱 Liste des appareils — à implémenter (Phase 3B)
          </div>
        )}
      </main>
    </div>
  );
};

export default DashboardPage;
