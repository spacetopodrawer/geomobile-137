import React, { useEffect } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { setUserId } from './redux/slices/authSlice';
import { RootState } from './redux/store';
import ErrorBoundary from './components/Common/ErrorBoundary';
import Header from './components/Navigation/Header';
import QuestsPage from './pages/QuestsPage';
import MapPage from './pages/MapPage';
import ShopPage from './pages/ShopPage';
import LeaderboardPage from './pages/LeaderboardPage';
import ProfilePage from './pages/ProfilePage';

const LoginPage = () => <div className="p-8"><h2 className="text-2xl font-bold">Login</h2></div>;

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
        <div className="min-h-screen bg-gray-50">
          <Header />
          <main className="max-w-7xl mx-auto">
            <Routes>
              <Route path="/quests" element={<QuestsPage />} />
              <Route path="/map" element={<MapPage />} />
              <Route path="/shop" element={<ShopPage />} />
              <Route path="/leaderboard" element={<LeaderboardPage />} />
              <Route path="/profile" element={<ProfilePage />} />
              <Route path="/" element={<Navigate to="/quests" replace />} />
            </Routes>
          </main>
        </div>
      </Router>
    </ErrorBoundary>
  );
};

export default App;
