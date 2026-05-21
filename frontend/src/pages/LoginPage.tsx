import React, { useState } from 'react';
import { useDispatch } from 'react-redux';
import { setUserId } from '../redux/slices/authSlice';

export const LoginPage: React.FC = () => {
  const dispatch = useDispatch();
  const [username, setUsername] = useState('');
  const [error, setError] = useState('');

  const handleLogin = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = username.trim();
    if (!trimmed) {
      setError('Veuillez entrer un identifiant.');
      return;
    }
    // Auth locale : stocke l'userId dans Redux + localStorage
    dispatch(setUserId(trimmed));
  };

  const handleQuickLogin = () => {
    const demoId = `user-demo-${Math.random().toString(36).slice(2, 7)}`;
    dispatch(setUserId(demoId));
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="bg-white rounded-xl shadow-lg p-8 w-full max-w-sm">
        <div className="text-center mb-6">
          <h1 className="text-2xl font-bold text-gray-800">🗺️ GeoMobile137</h1>
          <p className="text-sm text-gray-500 mt-1">Cadastral Management System</p>
        </div>

        <form onSubmit={handleLogin} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Identifiant utilisateur
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => { setUsername(e.target.value); setError(''); }}
              placeholder="ex: admin, user-001…"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm
                         focus:outline-none focus:ring-2 focus:ring-blue-500"
              autoFocus
            />
          </div>

          {error && (
            <p className="text-red-600 text-xs">{error}</p>
          )}

          <button
            type="submit"
            className="w-full bg-blue-600 hover:bg-blue-700 text-white font-semibold
                       py-2 rounded-lg text-sm transition-colors"
          >
            Se connecter
          </button>
        </form>

        <div className="mt-3 text-center">
          <button
            onClick={handleQuickLogin}
            className="text-xs text-gray-400 hover:text-blue-600 underline"
          >
            Accès démo rapide
          </button>
        </div>

        <p className="text-xs text-gray-400 text-center mt-6">
          Phase 3 — Real-Time Collaboration
        </p>
      </div>
    </div>
  );
};

export default LoginPage;
