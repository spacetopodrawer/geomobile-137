import React, { useEffect } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { setUserId } from './redux/slices/authSlice';
import { RootState } from './redux/store';
import ErrorBoundary from './components/Common/ErrorBoundary';
import { WebSocketStatus } from './components/WebSocketStatus';
import { PresenceList } from './components/PresenceList';
import { ActivityStream } from './components/ActivityStream';
import { useWebSocket } from './hooks/useWebSocket';
import { usePresenceStore } from './stores/presenceStore';
import { LoginPage } from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';

/**
 * AuthenticatedApp — rendu uniquement après login.
 * Le WebSocket se connecte ici, jamais sur la page Login.
 */
const AuthenticatedApp: React.FC = () => {
  const wsStore = useWebSocket();
  const { connect } = wsStore;
  const addUser = usePresenceStore((state) => state.addUser);

  // Connexion WS au montage (une seule fois)
  useEffect(() => {
    connect();
  }, []);

  // Ajouter l'utilisateur courant à la liste de présence
  useEffect(() => {
    if (wsStore.connected) {
      addUser({
        deviceId: wsStore.deviceId,
        userId: wsStore.userId,
        status: 'active',
        lastSeen: Date.now(),
      });
    }
  }, [wsStore.connected, wsStore.deviceId, wsStore.userId]);

  return (
    <div className="min-h-screen bg-gray-50">
      <Routes>
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
      {/* Overlays WebSocket — uniquement pour les utilisateurs connectés */}
      <WebSocketStatus />
      <PresenceList />
      <ActivityStream />
    </div>
  );
};

const App: React.FC = () => {
  const dispatch = useDispatch();
  const { isAuthenticated, userId } = useSelector((state: RootState) => state.auth);

  useEffect(() => {
    const storedUserId = localStorage.getItem('userId');
    if (storedUserId && !userId) {
      dispatch(setUserId(storedUserId));
    }
  }, [dispatch, userId]);

  if (!isAuthenticated) {
    return (
      <ErrorBoundary>
        <Router>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="*" element={<Navigate to="/login" replace />} />
          </Routes>
        </Router>
      </ErrorBoundary>
    );
  }

  return (
    <ErrorBoundary>
      <Router>
        <AuthenticatedApp />
      </Router>
    </ErrorBoundary>
  );
};

export default App;
